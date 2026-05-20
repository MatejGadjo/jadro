package postgres

import (
	"context"

	"github.com/MatejGadjo/NewsAggregator/internal/logger"
	"github.com/MatejGadjo/NewsAggregator/internal/storage"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pgvector/pgvector-go"
)

// InsertClusterVector inserts the initial representative vector for a cluster.
func InsertClusterVector(ctx context.Context, pool *pgxpool.Pool, log *logger.Logger, clusterID int64, embedding []float32, articleCount int) error {
	if len(embedding) == 0 {
		log.Warnw("attempted to insert cluster vector with empty embedding", "cluster_id", clusterID)
		return nil
	}

	vec := pgvector.NewVector(embedding)
	query := `
		INSERT INTO cluster_vectors (cluster_id, vector, article_count, updated_at)
		VALUES ($1, $2, $3, now())
	`

	_, err := pool.Exec(ctx, query, clusterID, vec, articleCount)
	if err != nil {
		log.Errorw("failed to insert cluster vector", "cluster_id", clusterID, "error", err)
		return err
	}

	return nil
}

// UpdateClusterVectorWithEmbeddingTx updates the cluster centroid within a transaction.
func UpdateClusterVectorWithEmbeddingTx(ctx context.Context, tx pgx.Tx, log *logger.Logger, clusterID int64, embedding []float32) error {
	if len(embedding) == 0 {
		log.Warnw("attempted to update cluster vector with empty embedding", "cluster_id", clusterID)
		return nil
	}

	vec := pgvector.NewVector(embedding)

	// Compute weighted average: new_vector = (old_vector * count + new_vector) / (count + 1)
	// Since pgvector doesn't support scalar multiplication, we do it in Go
	query := `
		SELECT vector, article_count
		FROM cluster_vectors
		WHERE cluster_id = $1
	`

	var currentVec pgvector.Vector
	var currentCount int
	err := tx.QueryRow(ctx, query, clusterID).Scan(&currentVec, &currentCount)
	if err != nil {
		log.Errorw("failed to fetch current cluster vector", "cluster_id", clusterID, "error", err)
		return err
	}

	// Compute weighted average in Go
	currentSlice := currentVec.Slice()
	newSlice := vec.Slice()
	if len(currentSlice) != len(newSlice) {
		log.Errorw("vector dimension mismatch", "cluster_id", clusterID, "current_dim", len(currentSlice), "new_dim", len(newSlice))
		return err
	}

	avgSlice := make([]float32, len(currentSlice))
	for i := range currentSlice {
		avgSlice[i] = (currentSlice[i]*float32(currentCount) + newSlice[i]) / float32(currentCount+1)
	}
	avgVec := pgvector.NewVector(avgSlice)

	updateQuery := `
		UPDATE cluster_vectors
		SET vector = $2,
		    article_count = article_count + 1,
		    updated_at = now()
		WHERE cluster_id = $1
	`

	result, err := tx.Exec(ctx, updateQuery, clusterID, avgVec)
	if err != nil {
		log.Errorw("failed to update cluster vector", "cluster_id", clusterID, "error", err)
		return err
	}

	if result.RowsAffected() == 0 {
		log.Warnw("no cluster vector found to update", "cluster_id", clusterID)
		return pgx.ErrNoRows
	}

	return nil
}

// FindSimilarClusterNeighborsForClustering finds similar clusters using pgvector ANN with time filters.
// Returns cluster IDs with similarity, ordered by distance (closest first).
func FindSimilarClusterNeighborsForClustering(
	ctx context.Context,
	pool *pgxpool.Pool,
	log *logger.Logger,
	embedding []float32,
	hours int,
	limit int,
) ([]storage.ClusterNeighbor, error) {
	if len(embedding) == 0 {
		return nil, nil
	}

	vec := pgvector.NewVector(embedding)
	query := `
		SELECT cv.cluster_id, 1 - (cv.vector <=> $1::vector) as similarity
		FROM cluster_vectors cv
		INNER JOIN clusters c ON c.id = cv.cluster_id
		WHERE c.last_seen_at >= NOW() - make_interval(hours => $2)
		ORDER BY cv.vector <=> $1::vector
		LIMIT $3
	`

	rows, err := pool.Query(ctx, query, vec, hours, limit)
	if err != nil {
		log.Errorw("failed to query similar cluster neighbors by embedding", "error", err)
		return nil, err
	}
	defer rows.Close()

	var neighbors []storage.ClusterNeighbor
	for rows.Next() {
		var neighbor storage.ClusterNeighbor
		if err := rows.Scan(&neighbor.ClusterID, &neighbor.Similarity); err != nil {
			log.Errorw("failed to scan similar cluster neighbor", "error", err)
			return nil, err
		}
		neighbors = append(neighbors, neighbor)
	}

	if err := rows.Err(); err != nil {
		log.Errorw("error iterating similar cluster neighbors", "error", err)
		return nil, err
	}

	return neighbors, nil
}
