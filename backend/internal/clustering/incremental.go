package clustering

import (
	"context"
	"errors"
	"math"
	"regexp"
	"strings"
	"time"

	"github.com/MatejGadjo/NewsAggregator/internal/logger"
	"github.com/MatejGadjo/NewsAggregator/internal/ranking"
	"github.com/MatejGadjo/NewsAggregator/internal/storage"
	"github.com/MatejGadjo/NewsAggregator/internal/storage/postgres"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	// SimilarityThreshold is the minimum similarity score for clustering.
	// When using embeddings (primary): 0.65 provides good precision for semantic similarity
	// Recommended range: 0.60-0.70 for embedding-based clustering
	SimilarityThreshold = 0.60
	// DefaultClusteringHours is the default time window for clustering articles (24 hours).
	DefaultClusteringHours = 24
	// MaxANNNeighbors limits how many nearest neighbors we pull per article for ANN clustering.
	MaxANNNeighbors = 25

	clusterDisplayTitleEmbeddingWeight        = 1.3
	clusterDisplayTitleMinEmbeddingArticles   = 2
	clusterDisplayTitleMinEmbeddingSimilarity = 0.28
)

type clusterTitleCandidate struct {
	title     string
	score     float64
	index     int
	embedding []float32
}

var clusterDisplayTitleStopWords = map[string]struct{}{
	"all": {}, "about": {}, "after": {}, "again": {}, "against": {}, "also": {}, "an": {}, "and": {}, "any": {}, "are": {}, "as": {}, "at": {},
	"because": {}, "been": {}, "before": {}, "being": {}, "between": {}, "both": {}, "but": {}, "by": {}, "for": {}, "from": {}, "had": {}, "has": {},
	"have": {}, "having": {}, "her": {}, "how": {}, "if": {}, "in": {}, "into": {}, "is": {}, "it": {}, "its": {}, "just": {}, "may": {}, "more": {},
	"most": {}, "no": {}, "not": {}, "of": {}, "on": {}, "or": {}, "our": {}, "out": {}, "so": {}, "some": {}, "such": {}, "that": {}, "the": {},
	"their": {}, "them": {}, "then": {}, "there": {}, "these": {}, "they": {}, "this": {}, "those": {}, "through": {}, "to": {}, "too": {}, "under": {},
	"until": {}, "up": {}, "was": {}, "were": {}, "what": {}, "when": {}, "where": {}, "which": {}, "while": {}, "who": {}, "why": {}, "will": {}, "with": {},
	"would": {}, "you": {}, "video": {}, "images": {}, "photo": {}, "photos": {}, "watch": {}, "live": {}, "update": {}, "updates": {}, "breaking": {}, "latest": {},
	"report": {}, "reports": {}, "here": {}, "today": {}, "tonight": {}, "analysis": {}, "analysis:": {},
	"и": {}, "на": {}, "за": {}, "со": {}, "во": {}, "до": {}, "од": {}, "е": {}, "се": {}, "да": {}, "не": {}, "ќе": {}, "ти": {}, "ни": {}, "им": {},
}

var clusterTitleTokenRE = regexp.MustCompile(`[\p{L}0-9]+`)

// ErrMissingEmbedding indicates clustering was skipped due to a missing embedding.
var ErrMissingEmbedding = errors.New("clustering skipped: missing embedding (embedding_status must be ready)")

// AssignArticleToClusterIncremental assigns a new article to the best-matching cluster
// using ANN neighbors when embeddings are available.
// Returns the cluster ID and whether a new cluster was created.
func AssignArticleToClusterIncremental(
	ctx context.Context,
	pool *pgxpool.Pool,
	log *logger.Logger,
	article storage.Article,
	embedding []float32,
) (int64, bool, error) {
	if article.ID == 0 {
		return 0, false, nil
	}
	if article.EmbeddingStatus != "ready" {
		return 0, false, ErrMissingEmbedding
	}
	if len(embedding) == 0 {
		return 0, false, ErrMissingEmbedding
	}

	// Skip if already clustered (e.g., deduplicated URL).
	existing, err := postgres.GetClusterIDsForArticles(ctx, pool, log, []int64{article.ID})
	if err != nil {
		return 0, false, err
	}
	if clusterID, ok := existing[article.ID]; ok {
		return clusterID, false, nil
	}

	var (
		bestClusterID int64
		bestSim       float64
	)
	bestClusterID, bestSim, _, err = bestClusterByEmbedding(ctx, pool, log, embedding)
	if err != nil {
		return 0, false, err
	}

	attachable := bestClusterID != 0 && bestSim >= SimilarityThreshold
	if !attachable {
		clusterID, err := createClusterForArticle(ctx, pool, log, article)
		if err != nil {
			return 0, false, err
		}
		if err := postgres.AttachArticleToCluster(ctx, pool, log, clusterID, article.ID); err != nil {
			return 0, false, err
		}
		if err := postgres.InsertClusterVector(ctx, pool, log, clusterID, embedding, 1); err != nil {
			return 0, false, err
		}
		if err := updateClusterScorePreserveUpdatedAt(ctx, pool, log, clusterID); err != nil {
			log.Warnw("failed to update new cluster score", "cluster_id", clusterID, "error", err)
		}
		return clusterID, true, nil
	}

	tx, err := pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return 0, false, err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	attached, err := postgres.AttachArticleToClusterWithStatsTx(ctx, tx, log, bestClusterID, article.ID, article.SourceID, lastSeenAtForArticle(article))
	if err != nil {
		return 0, false, err
	}
	if !attached {
		if err := tx.Commit(ctx); err != nil {
			return 0, false, err
		}
		return bestClusterID, false, nil
	}
	if err := postgres.UpdateClusterVectorWithEmbeddingTx(ctx, tx, log, bestClusterID, embedding); err != nil {
		return 0, false, err
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, false, err
	}

	if err := updateClusterScorePreserveUpdatedAt(ctx, pool, log, bestClusterID); err != nil {
		log.Warnw("failed to update cluster score", "cluster_id", bestClusterID, "error", err)
	}
	if err := refreshClusterDisplayTitle(ctx, pool, log, bestClusterID); err != nil {
		log.Warnw("failed to refresh cluster display title", "cluster_id", bestClusterID, "error", err)
	}

	return bestClusterID, false, nil
}

func bestClusterByEmbedding(
	ctx context.Context,
	pool *pgxpool.Pool,
	log *logger.Logger,
	embedding []float32,
) (int64, float64, []storage.ClusterNeighbor, error) {
	if len(embedding) == 0 {
		return 0, 0, nil, nil
	}

	neighbors, err := postgres.FindSimilarClusterNeighborsForClustering(
		ctx, pool, log,
		embedding,
		DefaultClusteringHours,
		MaxANNNeighbors,
	)
	if err != nil || len(neighbors) == 0 {
		return 0, 0, nil, err
	}

	var bestClusterID int64
	var bestSim float64
	for _, neighbor := range neighbors {
		if neighbor.Similarity > bestSim {
			bestSim = neighbor.Similarity
			bestClusterID = neighbor.ClusterID
		}
	}

	return bestClusterID, bestSim, neighbors, nil
}

func createClusterForArticle(ctx context.Context, pool *pgxpool.Pool, log *logger.Logger, article storage.Article) (int64, error) {
	headline := article.Title
	if headline == "" {
		headline = article.NormalizedTitle
	}
	summary := article.Snippet
	if len(summary) > 320 {
		summary = summary[:320]
	}
	displayTitle := DeriveClusterDisplayTitleFromArticles([]storage.Article{article})

	return postgres.CreateClusterWithStats(ctx, pool, log, headline, summary, displayTitle, article.Topic, lastSeenAtForArticle(article), article.SourceID)
}

func updateClusterScorePreserveUpdatedAt(ctx context.Context, pool *pgxpool.Pool, log *logger.Logger, clusterID int64) error {
	cluster, err := postgres.GetClusterByID(ctx, pool, log, clusterID)
	if err != nil {
		return err
	}
	if cluster == nil {
		return pgx.ErrNoRows
	}
	// Refresh velocity so the score reflects current coverage momentum.
	recentCount, err := postgres.GetRecentArticleCount(ctx, pool, log, clusterID)
	if err != nil {
		log.Warnw("failed to get recent article count, using cached value", "cluster_id", clusterID, "error", err)
		recentCount = cluster.ArticlesLast6h
	}
	cluster.ArticlesLast6h = recentCount
	score := computeClusterScoreFromStats(cluster)
	return postgres.UpdateClusterScoreValue(ctx, pool, log, cluster.ID, score)
}

func refreshClusterDisplayTitle(
	ctx context.Context,
	pool *pgxpool.Pool,
	log *logger.Logger,
	clusterID int64,
) error {
	cluster, err := postgres.GetClusterByID(ctx, pool, log, clusterID)
	if err != nil {
		return err
	}
	if cluster == nil {
		return pgx.ErrNoRows
	}

	articles, err := postgres.GetArticlesInCluster(ctx, pool, log, clusterID)
	if err != nil {
		return err
	}

	if err := HydrateArticleEmbeddings(ctx, pool, log, articles); err != nil {
		return err
	}
	displayTitle := DeriveClusterDisplayTitleFromArticles(articles)
	return postgres.UpdateClusterDisplayTitle(ctx, pool, log, clusterID, displayTitle)
}

func HydrateArticleEmbeddings(
	ctx context.Context,
	pool *pgxpool.Pool,
	log *logger.Logger,
	articles []storage.Article,
) error {
	if len(articles) == 0 {
		return nil
	}

	articleIDs := make([]int64, 0, len(articles))
	for _, article := range articles {
		if article.ID == 0 {
			continue
		}
		articleIDs = append(articleIDs, article.ID)
	}

	embeddings, err := postgres.GetArticleEmbeddingsByIDs(ctx, pool, log, articleIDs)
	if err != nil {
		return err
	}

	for i := range articles {
		if embedding, ok := embeddings[articles[i].ID]; ok {
			articles[i].Embedding = embedding
		}
	}

	return nil
}

func DeriveClusterDisplayTitleFromArticles(articles []storage.Article) string {
	if len(articles) == 0 {
		return fallbackClusterDisplayTitle()
	}

	candidates := make([]clusterTitleCandidate, 0, len(articles))
	for idx, article := range articles {
		title := cleanRepresentativeArticleTitle(article)
		if title == "" {
			continue
		}
		candidates = append(candidates, clusterTitleCandidate{
			title:     title,
			index:     idx,
			score:     scoreRepresentativeArticleTitle(title, article),
			embedding: article.Embedding,
		})
	}
	if len(candidates) == 0 {
		return fallbackClusterDisplayTitle()
	}

	centroid, hasCentroid := averageEmbeddingForTitleCandidates(candidates)
	best := candidates[0]
	bestScore := best.score

	for _, candidate := range candidates {
		combinedScore := candidate.score
		if hasCentroid && len(candidate.embedding) > 0 {
			if similarity, ok := cosineSimilarity(candidate.embedding, centroid); ok {
				combinedScore += similarity * clusterDisplayTitleEmbeddingWeight
			}
		}
		if combinedScore > bestScore || (combinedScore == bestScore && candidate.index < best.index) {
			best = candidate
			bestScore = combinedScore
		}
	}

	return best.title
}

func normalizeClusterDisplayTitle(value string) string {
	title := strings.TrimSpace(value)
	if title == "" {
		return ""
	}

	for _, sep := range []string{" - ", " | ", " — ", " – ", " • "} {
		if idx := strings.Index(title, sep); idx > 0 {
			title = strings.TrimSpace(title[:idx])
			break
		}
	}
	for _, prefix := range []string{"video:", "photo:", "photos:", "watch:", "live:", "breaking:", "update:", "updates:"} {
		if strings.HasPrefix(strings.ToLower(title), prefix) {
			title = strings.TrimSpace(title[len(prefix):])
		}
	}

	return strings.TrimSpace(title)
}


func averageEmbeddingForTitleCandidates(candidates []clusterTitleCandidate) ([]float32, bool) {
	var centroid []float32
	count := 0

	for _, candidate := range candidates {
		if len(candidate.embedding) == 0 {
			continue
		}
		if centroid == nil {
			centroid = make([]float32, len(candidate.embedding))
		}
		if len(candidate.embedding) != len(centroid) {
			continue
		}
		for i := range candidate.embedding {
			centroid[i] += candidate.embedding[i]
		}
		count++
	}

	if count < clusterDisplayTitleMinEmbeddingArticles || len(centroid) == 0 {
		return nil, false
	}

	invCount := float32(1.0 / float32(count))
	for i := range centroid {
		centroid[i] *= invCount
	}

	return centroid, true
}

func cosineSimilarity(a, b []float32) (float64, bool) {
	if len(a) == 0 || len(b) == 0 || len(a) != len(b) {
		return 0, false
	}

	var dot float64
	var normA float64
	var normB float64
	for i := range a {
		av := float64(a[i])
		bv := float64(b[i])
		dot += av * bv
		normA += av * av
		normB += bv * bv
	}

	if normA == 0 || normB == 0 {
		return 0, false
	}

	return dot / (math.Sqrt(normA) * math.Sqrt(normB)), true
}

func cleanRepresentativeArticleTitle(article storage.Article) string {
	title := strings.TrimSpace(article.Title)
	if title == "" {
		title = strings.TrimSpace(article.NormalizedTitle)
	}
	title = normalizeClusterDisplayTitle(title)
	if title == "" {
		return ""
	}

	locs := clusterTitleTokenRE.FindAllStringIndex(title, -1)
	for len(locs) > 0 {
		start, end := locs[0][0], locs[0][1]
		if !isClusterDisplayStopWord(strings.ToLower(title[start:end])) {
			break
		}
		title = strings.TrimSpace(title[end:])
		locs = locs[1:]
	}
	if title == "" {
		return ""
	}

	return title
}

func scoreRepresentativeArticleTitle(title string, article storage.Article) float64 {
	words := strings.Fields(title)
	score := 0.0

	wordCount := len(words)
	switch {
	case wordCount >= 6 && wordCount <= 12:
		score += 1.0
	case wordCount >= 4 && wordCount <= 14:
		score += 0.6
	default:
		score += 0.2
	}

	lower := strings.ToLower(title)
	for _, noisy := range []string{"breaking", "live", "video", "photos", "watch", "exclusive"} {
		if strings.Contains(lower, noisy) {
			score -= 0.35
		}
	}
	if article.PublishedAt != nil {
		score += 0.05
	}

	return score
}

func isClusterDisplayStopWord(value string) bool {
	_, exists := clusterDisplayTitleStopWords[value]
	return exists
}

func fallbackClusterDisplayTitle() string {
	return "Top Stories"
}

func lastSeenAtForArticle(article storage.Article) time.Time {
	if article.PublishedAt != nil {
		return *article.PublishedAt
	}
	return article.FetchedAt
}

func computeClusterScoreFromStats(cluster *storage.Cluster) float64 {
	if cluster == nil {
		return 0
	}
	newest := cluster.LastSeenAt
	created := cluster.CreatedAt
	return ranking.ComputeClusterScore(cluster.ArticleCount, &newest, &created, cluster.SourceCount, cluster.ArticlesLast6h)
}
