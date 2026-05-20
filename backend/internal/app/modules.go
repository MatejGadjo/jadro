package app

import (
	"context"
	"strconv"
	"time"

	"github.com/MatejGadjo/NewsAggregator/internal/cache"
	"github.com/MatejGadjo/NewsAggregator/internal/clustering"
	"github.com/MatejGadjo/NewsAggregator/internal/config"
	"github.com/MatejGadjo/NewsAggregator/internal/db"
	"github.com/MatejGadjo/NewsAggregator/internal/http"
	"github.com/MatejGadjo/NewsAggregator/internal/http/handlers"
	"github.com/MatejGadjo/NewsAggregator/internal/http/middleware"
	"github.com/MatejGadjo/NewsAggregator/internal/ingestion/rss"
	"github.com/MatejGadjo/NewsAggregator/internal/jobs"
	"github.com/MatejGadjo/NewsAggregator/internal/logger"
	"github.com/MatejGadjo/NewsAggregator/internal/storage/postgres"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/fx"
)

// newCacheClient builds a Redis cache client if REDIS_URL is set; returns nil otherwise (caching disabled).
func newCacheClient(cfg *config.Config, log *logger.Logger) *cache.Client {
	if cfg.RedisURL == "" {
		log.Infow("REDIS_URL not set, feed caching disabled")
		return nil
	}
	c, err := cache.New(cfg.RedisURL)
	if err != nil {
		log.Warnw("failed to connect to Redis, feed caching disabled", "error", err)
		return nil
	}
	return c
}

// Module wires the application dependencies.
var Module = fx.Options(
	fx.Invoke(func(pool *pgxpool.Pool) error {
		return db.Migrate(pool)
	}),
	fx.Provide(
		config.Load,
		logger.NewLogger,
		postgres.NewPool,
		newCacheClient,
		// keep defaults inside NewDispatcher; worker count is configured at start.
		clustering.NewEmbeddingGenerator,
		jobs.NewDispatcher,
		jobs.NewPipeline,
		jobs.NewRescoreJob,
		func(cfg *config.Config, log *logger.Logger, pool *pgxpool.Pool, pipeline *jobs.Pipeline, rescore *jobs.RescoreJob, cacheClient *cache.Client) *jobs.Scheduler {
			ingest := func(ctx context.Context) (int, error) {
				return rss.StartRSSIngestion(ctx, pool, log, pipeline)
			}
			s := jobs.NewScheduler(log, ingest, rescore)
			if cacheClient != nil {
				s.SetPostIngestHook(func(ctx context.Context) {
					version := strconv.FormatInt(time.Now().Unix(), 10)
					if err := cacheClient.Set(ctx, "meta:feed_version", []byte(version), 0); err != nil {
						log.Warnw("failed to write feed version", "error", err)
					}
					if err := cacheClient.DeleteByPattern(ctx, "feed:*"); err != nil {
						log.Warnw("feed cache invalidation failed", "error", err)
					}
				})
			}
			return s
		},
		jobs.NewCleanupJob,
		handlers.NewFeedHandler,
		handlers.NewHealthHandler,
		handlers.NewClusterHandler,
		middleware.NewRateLimiter,
		http.NewServer,
	),
	fx.Invoke(func(lc fx.Lifecycle, cfg *config.Config, dispatcher *jobs.Dispatcher, pipeline *jobs.Pipeline) {
		lc.Append(fx.Hook{
			OnStart: func(ctx context.Context) error {
				dispatcher.Start(cfg.EmbeddingWorkers)
				pipeline.RecoverStaleEmbeddings()
				return nil
			},
			OnStop: func(ctx context.Context) error {
				dispatcher.Stop()
				return nil
			},
		})
	}),
	fx.Invoke(func(lc fx.Lifecycle, cfg *config.Config, scheduler *jobs.Scheduler) {
		lc.Append(fx.Hook{
			OnStart: func(ctx context.Context) error {
				return scheduler.Start(cfg.IngestCron)
			},
			OnStop: func(ctx context.Context) error {
				scheduler.Stop()
				return nil
			},
		})
	}),
	fx.Invoke(func(lc fx.Lifecycle, cfg *config.Config, cleanup *jobs.CleanupJob) {
		lc.Append(fx.Hook{
			OnStart: func(ctx context.Context) error {
				return cleanup.Start(cfg.CleanupCron, cfg.CleanupRetentionDays)
			},
			OnStop: func(ctx context.Context) error {
				cleanup.Stop()
				return nil
			},
		})
	}),
)
