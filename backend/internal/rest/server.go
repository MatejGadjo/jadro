package rest

import (
	"context"
	"fmt"
	nethttp "net/http"
	"time"

	"github.com/MatejGadjo/jadro/internal/config"
	"github.com/MatejGadjo/jadro/internal/logger"
	"github.com/MatejGadjo/jadro/internal/rest/handlers"
	"github.com/MatejGadjo/jadro/internal/rest/middleware"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

const handlerTimeout = 10 * time.Second

// maxRequestBody comfortably fits a push subscription (endpoint + keys).
const maxRequestBody = 16 << 10

const (
	defaultReadHeaderTimeout = 5 * time.Second
	defaultReadTimeout       = 10 * time.Second
	defaultIdleTimeout       = 60 * time.Second
	defaultMaxHeaderBytes    = 32 << 10
)

func NewServer(
	lc fx.Lifecycle,
	cfg *config.Config,
	log *logger.Logger,
	feed *handlers.FeedHandler,
	health *handlers.HealthHandler,
	cluster *handlers.ClusterHandler,
	prerender *handlers.PrerenderHandler,
	push *handlers.PushHandler,
	rl *middleware.RateLimiter,
) (*nethttp.Server, error) {
	r := gin.New()
	// Trust X-Forwarded-For only from the nginx container (Docker bridge subnet).
	// This makes ClientIP() return the real visitor IP for per-IP rate limiting.
	if err := r.SetTrustedProxies(cfg.Security.TrustedProxies); err != nil {
		return nil, fmt.Errorf("invalid TRUSTED_PROXIES: %w", err)
	}
	r.Use(middleware.CORS(cfg.Security.AllowedOrigins))
	r.Use(gin.Recovery())
	r.Use(middleware.Timeout(handlerTimeout))
	r.Use(middleware.BodyLimit(maxRequestBody))

	RegisterRoutes(r, feed, health, cluster, prerender, push, rl)

	srv := &nethttp.Server{
		Addr:              cfg.HTTP.Addr,
		Handler:           r,
		ReadHeaderTimeout: defaultReadHeaderTimeout,
		ReadTimeout:       defaultReadTimeout,
		IdleTimeout:       defaultIdleTimeout,
		MaxHeaderBytes:    defaultMaxHeaderBytes,
	}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			log.Infow("http server starting", "addr", cfg.HTTP.Addr)
			go func() {
				if err := srv.ListenAndServe(); err != nil && err != nethttp.ErrServerClosed {
					log.Errorw("http server stopped unexpectedly", "error", err)
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			stopCtx := ctx
			if cfg.HTTP.ShutdownTimeout > 0 {
				var cancel context.CancelFunc
				stopCtx, cancel = context.WithTimeout(ctx, cfg.HTTP.ShutdownTimeout)
				defer cancel()
			}

			log.Infow("http server shutting down")
			return srv.Shutdown(stopCtx)
		},
	})

	return srv, nil
}
