package postgres

import (
	"context"
	"time"

	"github.com/MatejGadjo/NewsAggregator/internal/config"
	"github.com/MatejGadjo/NewsAggregator/internal/logger"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/fx"
)

const (
	defaultConnectTimeout = 5 * time.Second
	defaultPingTimeout    = 5 * time.Second
)

func NewPool(lc fx.Lifecycle, cfg *config.Config, log *logger.Logger) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(context.Background(), cfg.DatabaseURL)
	if err != nil {
		return nil, err
	}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			timeout := defaultPingTimeout
			pingCtx, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()

			if err := pool.Ping(pingCtx); err != nil {
				log.Errorw("postgres ping failed: %v", err)
				return err
			}
			log.Infow("postgres pool connected")
			return nil
		},
		OnStop: func(ctx context.Context) error {
			pool.Close()
			log.Infow("postgres pool closed")
			return nil
		},
	})

	return pool, nil
}
