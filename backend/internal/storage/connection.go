package storage

import (
	"context"
	"embed"
	"time"

	"github.com/MatejGadjo/jadro/internal/config"
	"github.com/MatejGadjo/jadro/internal/logger"
	"github.com/MatejGadjo/jadro/internal/storage/sqlc"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"go.uber.org/fx"
)

const defaultPingTimeout = 5 * time.Second

//go:embed migrations/*.sql
var migrationsFS embed.FS

type Client struct {
	q    *sqlc.Queries
	Pool *pgxpool.Pool
}

func NewClient(lc fx.Lifecycle, cfg *config.Config, log *logger.Logger) (*Client, error) {
	pool, err := pgxpool.New(context.Background(), cfg.DB.URL)
	if err != nil {
		return nil, err
	}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			pingCtx, cancel := context.WithTimeout(ctx, defaultPingTimeout)
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

	return &Client{q: sqlc.New(pool), Pool: pool}, nil
}

func (c *Client) Ping(ctx context.Context) error {
	return c.Pool.Ping(ctx)
}

func (c *Client) ArticleCount(ctx context.Context) (int, error) {
	n, err := c.q.CountArticles(ctx)
	return int(n), err
}

func (c *Client) ClusterCount(ctx context.Context) (int, error) {
	n, err := c.q.CountClusters(ctx)
	return int(n), err
}

func Migrate(c *Client) error {
	db := stdlib.OpenDBFromPool(c.Pool)

	goose.SetBaseFS(migrationsFS)
	goose.SetLogger(goose.NopLogger())

	if err := goose.SetDialect("postgres"); err != nil {
		return err
	}

	return goose.Up(db, "migrations")
}
