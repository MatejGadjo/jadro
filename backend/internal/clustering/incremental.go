package clustering

import (
	"context"
	"errors"
	"math"
	"regexp"
	"strings"
	"time"

	"github.com/MatejGadjo/NewsAggregator/domain"
	"github.com/MatejGadjo/NewsAggregator/internal/ranking"
	"github.com/jackc/pgx/v5"
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

	clusterDisplayTitleEmbeddingWeight      = 1.3
	clusterDisplayTitleMinEmbeddingArticles = 2
)

type clusterTitleCandidate struct {
	title     string
	score     float64
	index     int
	embedding []float32
}

var clusterDisplayTitleStopWords = map[string]struct{}{
	// English — only words that appear in Macedonian news headlines
	"video": {}, "photo": {}, "photos": {}, "images": {}, "watch": {}, "live": {},
	"update": {}, "updates": {}, "breaking": {}, "latest": {}, "report": {}, "reports": {},
	"today": {}, "tonight": {}, "analysis": {},
	// Macedonian
	"и": {}, "на": {}, "за": {}, "со": {}, "во": {}, "до": {}, "од": {}, "по": {}, "при": {}, "без": {}, "над": {}, "под": {}, "меѓу": {},
	"е": {}, "се": {}, "да": {}, "не": {}, "ќе": {}, "ти": {}, "ни": {}, "им": {}, "ги": {}, "го": {}, "ја": {}, "си": {},
	"или": {}, "но": {}, "оти": {}, "дека": {}, "што": {},
	"кој": {}, "која": {}, "кои": {},
	"овој": {}, "оваа": {}, "ова": {}, "тоа": {}, "тој": {}, "таа": {},
	"еден": {}, "една": {}, "едно": {}, "нема": {},
}

var clusterTitleTokenRE = regexp.MustCompile(`[\p{L}0-9]+`)

// ErrMissingEmbedding indicates clustering was skipped due to a missing embedding.
var ErrMissingEmbedding = errors.New("clustering skipped: missing embedding (embedding_status must be ready)")

func (s *ClusteringService) bestClusterByEmbedding(
	ctx context.Context,
	embedding []float32,
) (int64, float64, error) {
	if len(embedding) == 0 {
		return 0, 0, nil
	}

	neighbors, err := s.store.FindSimilarClusterNeighbors(ctx, embedding, DefaultClusteringHours, MaxANNNeighbors)
	if err != nil || len(neighbors) == 0 {
		return 0, 0, err
	}

	var bestClusterID int64
	var bestSim float64
	for _, n := range neighbors {
		if n.Similarity > bestSim {
			bestSim = n.Similarity
			bestClusterID = n.ClusterID
		}
	}

	return bestClusterID, bestSim, nil
}

func (s *ClusteringService) createClusterForArticle(ctx context.Context, article domain.Article) (int64, error) {
	headline := article.Title
	if headline == "" {
		headline = article.NormalizedTitle
	}
	summary := article.Snippet
	if len(summary) > 320 {
		summary = summary[:320]
	}
	displayTitle := deriveClusterDisplayTitleFromArticles([]domain.Article{article})

	clusterID, err := s.store.CreateCluster(ctx, headline, summary, displayTitle, article.Topic, lastSeenAtForArticle(article))
	if err != nil {
		return 0, err
	}
	if err := s.store.InsertClusterSource(ctx, clusterID, article.SourceID); err != nil {
		return 0, err
	}
	return clusterID, nil
}

func (s *ClusteringService) updateClusterScorePreserveUpdatedAt(ctx context.Context, clusterID int64) error {
	cluster, err := s.store.GetClusterByID(ctx, clusterID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return pgx.ErrNoRows
		}
		return err
	}
	recentCount, err := s.store.GetRecentArticleCount(ctx, clusterID)
	if err != nil {
		s.log.Warnw("failed to get recent article count, using cached value", "cluster_id", clusterID, "error", err)
		recentCount = int64(cluster.ArticlesLast6h)
	}
	cluster.ArticlesLast6h = int(recentCount)
	score := computeClusterScoreFromStats(&cluster)
	return s.store.UpdateClusterScoreValue(ctx, clusterID, score)
}

func (s *ClusteringService) refreshClusterDisplayTitle(ctx context.Context, clusterID int64) error {
	_, err := s.store.GetClusterByID(ctx, clusterID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return pgx.ErrNoRows
		}
		return err
	}

	articles, err := s.store.GetArticlesInCluster(ctx, clusterID)
	if err != nil {
		return err
	}

	if err := s.hydrateArticleEmbeddings(ctx, articles); err != nil {
		return err
	}
	displayTitle := deriveClusterDisplayTitleFromArticles(articles)
	return s.store.UpdateClusterDisplayTitle(ctx, clusterID, displayTitle)
}

// hydrateArticleEmbeddings fills in the Embedding field for each article.
func (s *ClusteringService) hydrateArticleEmbeddings(
	ctx context.Context,
	articles []domain.Article,
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

	rows, err := s.store.GetArticleEmbeddingsByIDs(ctx, articleIDs)
	if err != nil {
		return err
	}

	embeddings := make(map[int64][]float32, len(rows))
	for _, r := range rows {
		if len(r.Embedding) > 0 {
			embeddings[r.ID] = r.Embedding
		}
	}

	for i := range articles {
		if embedding, ok := embeddings[articles[i].ID]; ok {
			articles[i].Embedding = embedding
		}
	}

	return nil
}

func deriveClusterDisplayTitleFromArticles(articles []domain.Article) string {
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

func cleanRepresentativeArticleTitle(article domain.Article) string {
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

func scoreRepresentativeArticleTitle(title string, article domain.Article) float64 {
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

func lastSeenAtForArticle(article domain.Article) time.Time {
	if article.PublishedAt != nil {
		return *article.PublishedAt
	}
	return article.FetchedAt
}

func computeClusterScoreFromStats(cluster *domain.Cluster) float64 {
	if cluster == nil {
		return 0
	}
	newest := cluster.LastSeenAt
	created := cluster.CreatedAt
	return ranking.ComputeClusterScore(cluster.ArticleCount, &newest, &created, cluster.SourceCount, cluster.ArticlesLast6h)
}
