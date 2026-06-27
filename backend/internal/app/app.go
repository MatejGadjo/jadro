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
	"github.com/MatejGadjo/NewsAggregator/internal/push"
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
		fx.Provide(func(cfg *config.Config) config.PushConfig { return cfg.Push }),

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
		// push: store + service + daily job. Service is constructed regardless;
		// the cron only registers the daily send when VAPID keys are present.
		fx.Provide(func(c *storage.Client) push.Store { return c }),
		fx.Provide(func(c *storage.Client) push.FeedReader { return c }),
		fx.Provide(func(cfg config.PushConfig, store push.Store, feed push.FeedReader, log *logger.Logger) *push.Service {
			return push.NewService(push.Config{
				VAPIDPublic:  cfg.VAPIDPublic,
				VAPIDPrivate: cfg.VAPIDPrivate,
				VAPIDSubject: cfg.VAPIDSubject,
				MinSources:   cfg.MinSources,
				FreshHours:   cfg.FreshHours,
			}, store, feed, log)
		}),
		fx.Provide(func(svc *push.Service, log *logger.Logger) *jobs.PushJob {
			return jobs.NewPushJob(svc, log)
		}),
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
		fx.Provide(func(c *storage.Client) handlers.PushStore { return c }),
		fx.Provide(func(cfg config.PushConfig, store handlers.PushStore, log *logger.Logger) *handlers.PushHandler {
			return handlers.NewPushHandler(store, cfg.VAPIDPublic, log)
		}),

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
