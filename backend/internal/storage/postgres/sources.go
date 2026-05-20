package postgres

import (
	"context"
	"errors"

	"github.com/MatejGadjo/NewsAggregator/internal/logger"
	"github.com/MatejGadjo/NewsAggregator/internal/storage"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ListActiveSources returns all active sources from the database.
func ListActiveSources(ctx context.Context, pool *pgxpool.Pool, log *logger.Logger) ([]storage.Source, error) {
	query := `
		SELECT s.id, s.name, s.base_url, s.rss_url,
		       to_jsonb(s)->>'logo_url' AS logo_url,
		       s.active, s.rate_limit_per_min, s.created_at
		FROM sources s
		WHERE s.active = TRUE
		ORDER BY s.created_at ASC
	`

	rows, err := pool.Query(ctx, query)
	if err != nil {
		log.Errorw("failed to query active sources", "error", err)
		return nil, err
	}
	defer rows.Close()

	var sources []storage.Source
	for rows.Next() {
		var s storage.Source
		if err := rows.Scan(
			&s.ID,
			&s.Name,
			&s.BaseURL,
			&s.RSSURL,
			&s.LogoURL,
			&s.Active,
			&s.RateLimitPerMin,
			&s.CreatedAt,
		); err != nil {
			log.Errorw("failed to scan source row", "error", err)
			return nil, err
		}
		sources = append(sources, s)
	}

	if err := rows.Err(); err != nil {
		log.Errorw("error iterating source rows", "error", err)
		return nil, err
	}

	return sources, nil
}

// GetSourceByID retrieves a source by its ID.
func GetSourceByID(ctx context.Context, pool *pgxpool.Pool, log *logger.Logger, id int64) (*storage.Source, error) {
	query := `
		SELECT s.id, s.name, s.base_url, s.rss_url,
		       to_jsonb(s)->>'logo_url' AS logo_url,
		       s.active, s.rate_limit_per_min, s.created_at
		FROM sources s
		WHERE s.id = $1
	`

	var s storage.Source
	err := pool.QueryRow(ctx, query, id).Scan(
		&s.ID,
		&s.Name,
		&s.BaseURL,
		&s.RSSURL,
		&s.LogoURL,
		&s.Active,
		&s.RateLimitPerMin,
		&s.CreatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			log.Warnw("source not found", "source_id", id)
			return nil, err
		}
		log.Errorw("failed to get source by id", "source_id", id, "error", err)
		return nil, err
	}

	return &s, nil
}

// UpdateSourceLogoURL updates the source logo URL from RSS metadata (or clears it when missing).
func UpdateSourceLogoURL(ctx context.Context, pool *pgxpool.Pool, log *logger.Logger, sourceID int64, logoURL *string) error {
	query := `
		UPDATE sources
		SET logo_url = $2
		WHERE id = $1
		  AND logo_url IS DISTINCT FROM $2
	`

	_, err := pool.Exec(ctx, query, sourceID, logoURL)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "42703" {
			log.Warnw("logo_url column missing, skipping source logo update", "source_id", sourceID)
			return nil
		}
		log.Errorw("failed to update source logo url", "source_id", sourceID, "logo_url", logoURL, "error", err)
		return err
	}

	return nil
}

// DisableSource sets the active flag to false for a source.
func DisableSource(ctx context.Context, pool *pgxpool.Pool, log *logger.Logger, id int64) error {
	query := `
		UPDATE sources
		SET active = FALSE
		WHERE id = $1
	`

	result, err := pool.Exec(ctx, query, id)
	if err != nil {
		log.Errorw("failed to disable source", "source_id", id, "error", err)
		return err
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		log.Warnw("no source found to disable", "source_id", id)
		return pgx.ErrNoRows
	}

	return nil
}
