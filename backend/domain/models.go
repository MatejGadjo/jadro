package domain

import "time"

type EmbeddingStatus string

const (
	EmbeddingStatusPending    EmbeddingStatus = "pending"
	EmbeddingStatusProcessing EmbeddingStatus = "processing"
	EmbeddingStatusReady      EmbeddingStatus = "ready"
	EmbeddingStatusFailed     EmbeddingStatus = "failed"
)

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

type Article struct {
	ID                 int64           `json:"id"`
	SourceID           int64           `json:"source_id"`
	FeedID             *int64          `json:"feed_id,omitempty"`
	FeedURL            string          `json:"feed_url,omitempty"`
	SourceDomain       string          `json:"source_domain,omitempty"`
	URL                string          `json:"url"`
	ImageURL           *string         `json:"image_url,omitempty"`
	Title              string          `json:"title"`
	Snippet            string          `json:"snippet"`
	NormalizedTitle    string          `json:"normalized_title"`
	NormalizedSnippet  string          `json:"normalized_snippet"`
	RSSCategories      []string        `json:"rss_categories,omitempty"`
	Category           string          `json:"category"`
	CategoryConfidence float64         `json:"category_confidence"`
	CategoryMethod     string          `json:"category_method"`
	CategorySignals    []byte          `json:"category_signals,omitempty"`
	Topic              string          `json:"topic"`
	PublishedAt        *time.Time      `json:"published_at"`
	FetchedAt          time.Time       `json:"fetched_at"`
	EmbeddingStatus    EmbeddingStatus `json:"embedding_status"`
	Embedding          []float32       `json:"embedding,omitempty"`
}

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
	ArticleCount    int       `json:"article_count"`
	SourceCount     int       `json:"source_count"`
	ArticlesLast6h  int       `json:"articles_last_6h"`
	TopicConfidence *float64  `json:"topic_confidence,omitempty"`
}

type ClusterNeighbor struct {
	ClusterID  int64   `json:"cluster_id"`
	Similarity float64 `json:"similarity"`
}

// InsertArticleParams holds the fields needed to upsert an article.
type InsertArticleParams struct {
	SourceID           int64
	FeedID             int64
	FeedURL            *string
	SourceDomain       *string
	URL                string
	ImageURL           *string
	Title              string
	Snippet            *string
	NormalizedTitle    string
	NormalizedSnippet  *string
	RSSCategories      []string
	Category           string
	CategoryConfidence float64
	CategoryMethod     string
	CategorySignals    []byte
	Topic              string
	PublishedAt        *time.Time
	FetchedAt          time.Time
}

// InsertArticleResult is returned when inserting or upserting an article.
type InsertArticleResult struct {
	ID    int64
	IsNew bool
}

// LatestArticle is a summary row used in the "latest articles" feed.
type LatestArticle struct {
	ID            int64      `json:"id"`
	URL           string     `json:"url"`
	Title         string     `json:"title"`
	Topic         string     `json:"topic"`
	PublishedAt   *time.Time `json:"published_at"`
	FetchedAt     time.Time  `json:"fetched_at"`
	SourceName    string     `json:"source_name"`
	SourceLogoURL *string    `json:"source_logo_url,omitempty"`
}

// ClusterArticleSource is an article with its source, as listed within a cluster.
type ClusterArticleSource struct {
	ArticleID     int64      `json:"article_id"`
	URL           string     `json:"url"`
	ImageURL      *string    `json:"image_url,omitempty"`
	Title         string     `json:"title"`
	Snippet       string     `json:"snippet"`
	PublishedAt   *time.Time `json:"published_at"`
	FetchedAt     time.Time  `json:"fetched_at"`
	SourceID      int64      `json:"source_id"`
	SourceName    string     `json:"source_name"`
	SourceBaseURL string     `json:"source_base_url"`
	SourceRSSURL  string     `json:"source_rss_url"`
	SourceLogoURL *string    `json:"source_logo_url,omitempty"`
}

// ClusterWithFeaturedArticle is a cluster paired with its best representative article.
type ClusterWithFeaturedArticle struct {
	Cluster
	FeaturedArticle ClusterArticleSource
}

// ArticleClusterMapping maps an article ID to its cluster ID.
type ArticleClusterMapping struct {
	ArticleID int64
	ClusterID int64
}

// ArticleEmbedding pairs an article ID with its embedding vector.
type ArticleEmbedding struct {
	ID        int64
	Embedding []float32
}

// ClusterVector holds the centroid vector and article count for a cluster.
type ClusterVector struct {
	Vector       []float32
	ArticleCount int
}

// PushSubscription is a browser Web Push subscription (no user account; keyed by
// endpoint). p256dh/auth are the client's encryption keys from PushManager.
type PushSubscription struct {
	Endpoint string `json:"endpoint"`
	P256dh   string `json:"p256dh"`
	Auth     string `json:"auth"`
}
