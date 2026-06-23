package app

import (
	"context"
	"net/http"

	"github.com/MatejGadjo/NewsAggregator/internal/cache"
	"github.com/MatejGadjo/NewsAggregator/internal/clustering"
	"github.com/MatejGadjo/NewsAggregator/internal/config"
	"github.com/MatejGadjo/NewsAggregator/internal/ingestion/rss"
	"github.com/MatejGadjo/NewsAggregator/internal/jobs"
	"github.com/MatejGadjo/NewsAggregator/internal/logger"
	"github.com/MatejGadjo/NewsAggregator/internal/rest"
	"github.com/MatejGadjo/NewsAggregator/internal/rest/handlers"
	"github.com/MatejGadjo/NewsAggregator/internal/rest/middleware"
	"github.com/MatejGadjo/NewsAggregator/internal/service"
	"github.com/MatejGadjo/NewsAggregator/internal/storage"
	"github.com/alitto/pond/v2"
	"go.uber.org/fx"
)

func Initialize() *fx.App {
	opts := []fx.Option{
		// config
		fx.Provide(config.NewConfig),
		fx.Provide(func(cfg *config.Config) config.ONNXConfig { return cfg.ONNX }),
		fx.Provide(func(cfg *config.Config) config.JobsConfig { return cfg.Jobs }),

		// infrastructure
		fx.Provide(logger.NewLogger),
		fx.Provide(storage.NewClient),
		fx.Provide(func(cfg *config.Config) (*cache.Client, error) { return cache.NewCache(cfg.Cache.RedisURL) }),

		// clustering
		fx.Provide(clustering.NewEmbeddingGenerator),
		fx.Provide(clustering.NewEmbeddingService),
		fx.Provide(clustering.NewClusteringService),
		fx.Provide(func(svc *clustering.EmbeddingService) jobs.Embedder { return svc }),
		fx.Provide(func(svc *clustering.ClusteringService) jobs.ClusterAssigner { return svc }),

		// jobs
		fx.Provide(func(lc fx.Lifecycle, cfg config.ONNXConfig) pond.Pool {
			pool := pond.NewPool(cfg.Workers)
			lc.Append(fx.Hook{
				OnStop: func(_ context.Context) error {
					pool.StopAndWait()
					return nil
				},
			})
			return pool
		}),
		fx.Provide(jobs.NewPipeline),
		fx.Provide(jobs.NewRescoreJob),
		fx.Provide(jobs.NewCleanupJob),
		fx.Provide(func(log *logger.Logger, store rss.IngestionStore, pipeline *jobs.Pipeline, rescore *jobs.RescoreJob) *jobs.IngestRunner {
			ingest := func(ctx context.Context) (int, error) {
				return rss.StartRSSIngestion(ctx, store, log, pipeline)
			}
			return jobs.NewIngestRunner(log, ingest, rescore)
		}),
		fx.Provide(func(svc *service.FeedService) jobs.AfterIngestFunc { return svc.WarmCache }),
		fx.Provide(jobs.NewCron),

		// store interfaces
		fx.Provide(func(c *storage.Client) service.FeedStore { return c }),
		fx.Provide(func(c *storage.Client) service.ClusterStore { return c }),
		fx.Provide(func(c *storage.Client) clustering.ClusteringStore { return c }),
		fx.Provide(func(c *storage.Client) clustering.EmbeddingStore { return c }),
		fx.Provide(func(c *storage.Client) rss.IngestionStore { return c }),
		fx.Provide(func(c *storage.Client) jobs.RescoreStore { return c }),
		fx.Provide(func(c *storage.Client) jobs.CleanupStore { return c }),
		fx.Provide(func(c *storage.Client) service.HealthStore { return c }),
		// nil guards: returning a nil *cache.Client through an interface would
		// produce a non-nil interface value, defeating the services' nil checks.
		fx.Provide(func(c *cache.Client) service.HealthCache {
			if c == nil {
				return nil
			}
			return c
		}),
		fx.Provide(func(c *cache.Client) service.Cache {
			if c == nil {
				return nil
			}
			return c
		}),
		fx.Provide(service.NewFeedService),
		fx.Provide(service.NewClusterService),
		fx.Provide(service.NewHealthService),

		// handlers
		fx.Provide(func(svc *service.FeedService) handlers.FeedServicer { return svc }),
		fx.Provide(func(svc *service.ClusterService) handlers.ClusterServicer { return svc }),
		fx.Provide(func(svc *service.HealthService) handlers.HealthServicer { return svc }),
		fx.Provide(handlers.NewFeedHandler),
		fx.Provide(handlers.NewHealthHandler),
		fx.Provide(handlers.NewClusterHandler),
		fx.Provide(handlers.NewPrerenderHandler),

		// http
		fx.Provide(middleware.NewRateLimiter),
		fx.Provide(rest.NewServer),

		// startup
		fx.Invoke(func(c *storage.Client) error {
			return storage.Migrate(c)
		}),
		fx.Invoke(func(*jobs.Cron) {}),
		fx.Invoke(func(*http.Server) {}),
	}

	return fx.New(opts...)
}
