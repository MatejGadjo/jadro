package http

import (
	"context"
	nethttp "net/http"
	"time"

	"github.com/MatejGadjo/NewsAggregator/internal/config"
	"github.com/MatejGadjo/NewsAggregator/internal/http/handlers"
	"github.com/MatejGadjo/NewsAggregator/internal/http/middleware"
	"github.com/MatejGadjo/NewsAggregator/internal/logger"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

const handlerTimeout = 10 * time.Second

const (
	defaultReadHeaderTimeout = 5 * time.Second
)

func NewServer(
	lc fx.Lifecycle,
	cfg *config.Config,
	log *logger.Logger,
	feed *handlers.FeedHandler,
	health *handlers.HealthHandler,
	cluster *handlers.ClusterHandler,
	internal *handlers.InternalHandler,
	rl *middleware.RateLimiter,
) *nethttp.Server {
	r := gin.New()
	// Trust no proxy headers by default — RemoteAddr is used as client IP.
	// In production behind nginx/Cloudflare, set to the upstream proxy IPs.
	r.SetTrustedProxies(nil) //nolint:errcheck
	r.Use(middleware.CORS(&middleware.CORSConfig{
		AllowedOrigins:   cfg.AllowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS", "HEAD"},
		AllowedHeaders:   []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Requested-With"},
		AllowCredentials: false,
		MaxAge:           3600,
	}))
	r.Use(gin.Recovery())
	r.Use(middleware.Timeout(handlerTimeout))

	RegisterRoutes(r, cfg, feed, health, cluster, internal, rl)

	srv := &nethttp.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           r,
		ReadHeaderTimeout: defaultReadHeaderTimeout,
	}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			log.Infow("http server starting", "addr", cfg.HTTPAddr)
			go func() {
				if err := srv.ListenAndServe(); err != nil && err != nethttp.ErrServerClosed {
					log.Errorw("http server stopped unexpectedly", "error", err)
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			stopCtx := ctx
			if cfg.HTTPShutdownTimeout > 0 {
				var cancel context.CancelFunc
				stopCtx, cancel = context.WithTimeout(ctx, cfg.HTTPShutdownTimeout)
				defer cancel()
			}

			log.Infow("http server shutting down")
			return srv.Shutdown(stopCtx)
		},
	})

	return srv
}
