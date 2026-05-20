package storage

import "time"

// Source represents a news source.
type Source struct {
	ID              int64     `json:"id"`
	Name            string    `json:"name"`
	BaseURL         string    `json:"base_url"`
	RSSURL          string    `json:"rss_url"`
	LogoURL         *string   `json:"logo_url,omitempty"`
	Active          bool      `json:"active"`
	RateLimitPerMin int       `json:"rate_limit_per_min"`
	CreatedAt       time.Time `json:"created_at"`
}

// Article represents a raw ingested article.
type Article struct {
	ID                 int64      `json:"id"`
	SourceID           int64      `json:"source_id"`
	FeedID             *int64     `json:"feed_id,omitempty"`
	FeedURL            string     `json:"feed_url,omitempty"`
	SourceDomain       string     `json:"source_domain,omitempty"`
	URL                string     `json:"url"`
	ImageURL           *string    `json:"image_url,omitempty"`
	Title              string     `json:"title"`
	Snippet            string     `json:"snippet"`
	NormalizedTitle    string     `json:"normalized_title"`
	NormalizedSnippet  string     `json:"normalized_snippet"`
	RSSCategories      []string   `json:"rss_categories,omitempty"`
	Category           string     `json:"category"`
	CategoryConfidence float64    `json:"category_confidence"`
	CategoryMethod     string     `json:"category_method"`
	CategorySignals    []byte     `json:"category_signals,omitempty"`
	Topic              string     `json:"topic"`
	PublishedAt        *time.Time `json:"published_at"`
	FetchedAt          time.Time  `json:"fetched_at"`
	EmbeddingStatus    string     `json:"embedding_status"`
	Embedding          []float32  `json:"embedding,omitempty"` // Semantic embedding vector (384 dimensions)
}

// ClusterArticleWithSource joins article content with source metadata for homepage sections.
type ClusterArticleWithSource struct {
	ArticleID       int64      `json:"article_id"`
	ArticleURL      string     `json:"article_url"`
	ArticleImageURL *string    `json:"article_image_url,omitempty"`
	ArticleTitle    string     `json:"article_title"`
	Snippet         string     `json:"snippet"`
	PublishedAt     *time.Time `json:"published_at,omitempty"`
	FetchedAt       time.Time  `json:"fetched_at"`
	SourceID        int64      `json:"source_id"`
	SourceName      string     `json:"source_name"`
	SourceBaseURL   string     `json:"source_base_url"`
	SourceRSSURL    string     `json:"source_rss_url"`
	SourceLogoURL   *string    `json:"source_logo_url,omitempty"`
}

// ClusterNeighbor represents a similar cluster with its similarity score.
type ClusterNeighbor struct {
	ClusterID  int64   `json:"cluster_id"`
	Similarity float64 `json:"similarity"`
}

// LatestArticle is a lightweight article summary for the "Свежо" sidebar feed.
// One per cluster (newest article per cluster), ordered by recency.
type LatestArticle struct {
	ID            int64      `json:"id"`
	URL           string     `json:"url"`
	Title         string     `json:"title"`
	Topic         string     `json:"topic"`
	PublishedAt   *time.Time `json:"published_at,omitempty"`
	FetchedAt     time.Time  `json:"fetched_at"`
	SourceName    string     `json:"source_name"`
	SourceLogoURL *string    `json:"source_logo_url,omitempty"`
}

// ClusterWithRepresentativeArticle holds a cluster plus a single best article for homepage category sections.
type ClusterWithRepresentativeArticle struct {
	Cluster
	ArticleID       int64
	ArticleURL      string
	ArticleImageURL *string
	ArticleTitle    string
	ArticleSnippet  string
	PublishedAt     *time.Time
	FetchedAt       time.Time
	SourceID        int64
	SourceName      string
	SourceBaseURL   string
	SourceRSSURL    string
	SourceLogoURL   *string
}

// Cluster represents a grouped real-world news story.
type Cluster struct {
	ID              int64     `json:"id"`
	Headline        string    `json:"headline"`
	DisplayTitle    string    `json:"display_title"`
	Summary         string    `json:"summary"`
	Topic           string    `json:"topic"`
	Score           float64   `json:"score"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
	LastSeenAt      time.Time `json:"last_seen_at"`
	ArticleCount    int      `json:"article_count"`
	SourceCount     int      `json:"source_count"`
	ArticlesLast6h  int      `json:"articles_last_6h"`
	TopicConfidence *float64 `json:"topic_confidence,omitempty"`
}

