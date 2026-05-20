package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/MatejGadjo/NewsAggregator/internal/logger"
	"github.com/MatejGadjo/NewsAggregator/internal/storage"
	"github.com/MatejGadjo/NewsAggregator/internal/storage/postgres"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ClusterHandler struct {
	log  *logger.Logger
	pool *pgxpool.Pool
}

func NewClusterHandler(log *logger.Logger, pool *pgxpool.Pool) *ClusterHandler {
	return &ClusterHandler{log: log, pool: pool}
}

type clusterSourceDTO struct {
	ID      int64   `json:"id"`
	Name    string  `json:"name"`
	BaseURL string  `json:"base_url"`
	RSSURL  string  `json:"rss_url"`
	LogoURL *string `json:"logo_url,omitempty"`
}

type clusterArticleDTO struct {
	ID          int64            `json:"id"`
	URL         string           `json:"url"`
	ImageURL    *string          `json:"image_url,omitempty"`
	Title       string           `json:"title"`
	Snippet     string           `json:"snippet"`
	PublishedAt *time.Time       `json:"published_at,omitempty"`
	FetchedAt   time.Time        `json:"fetched_at"`
	Source      clusterSourceDTO `json:"source"`
}

const (
	latestArticlesDefault = 8
	latestArticlesMax     = 20
)

// GetLatestArticles returns the most recently published article per cluster across all topics.
// Used by the "Свежо" sidebar. Accepts optional ?limit=N (default 8, max 20).
func (h *ClusterHandler) GetLatestArticles(c *gin.Context) {
	limit := latestArticlesDefault
	if l := c.Query("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 {
			if n > latestArticlesMax {
				n = latestArticlesMax
			}
			limit = n
		}
	}

	ctx := c.Request.Context()

	articles, err := postgres.ListLatestArticles(ctx, h.pool, h.log, limit)
	if err != nil {
		h.log.Errorw("failed to fetch latest articles", "limit", limit, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch latest articles"})
		return
	}

	if articles == nil {
		articles = []storage.LatestArticle{}
	}

	c.JSON(http.StatusOK, gin.H{"articles": articles})
}

// PatchClearImage sets image_url to NULL for an article when the frontend reports a broken URL.
// Fire-and-forget from the client — returns 204 on success, 404 if article not found.
func (h *ClusterHandler) PatchClearImage(c *gin.Context) {
	articleID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || articleID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid article id"})
		return
	}

	ctx := c.Request.Context()
	if err := postgres.ClearArticleImageURL(ctx, h.pool, h.log, articleID); err != nil {
		if err == pgx.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "article not found"})
			return
		}
		h.log.Errorw("failed to clear article image", "article_id", articleID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to clear image"})
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *ClusterHandler) GetCluster(c *gin.Context) {
	idStr := c.Param("id")
	clusterID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || clusterID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid cluster id",
		})
		return
	}

	ctx := c.Request.Context()

	cluster, err := postgres.GetClusterByID(ctx, h.pool, h.log, clusterID)
	if err != nil {
		h.log.Errorw("failed to fetch cluster", "cluster_id", clusterID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch cluster"})
		return
	}
	if cluster == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "cluster not found"})
		return
	}

	articles, err := postgres.GetArticlesInCluster(ctx, h.pool, h.log, clusterID)
	if err != nil {
		h.log.Errorw("failed to fetch articles in cluster", "cluster_id", clusterID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch cluster articles"})
		return
	}

	// Enrich articles with source metadata. Cache sources to avoid repeat DB calls.
	sourceCache := make(map[int64]clusterSourceDTO, 16)
	outArticles := make([]clusterArticleDTO, 0, len(articles))
	for _, a := range articles {
		src, ok := sourceCache[a.SourceID]
		if !ok {
			s, err := postgres.GetSourceByID(ctx, h.pool, h.log, a.SourceID)
			if err != nil {
				h.log.Errorw("failed to fetch source for article", "source_id", a.SourceID, "article_id", a.ID, "error", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch article source"})
				return
			}
			if s == nil {
				h.log.Warnw("source missing for article", "source_id", a.SourceID, "article_id", a.ID)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "source missing for article"})
				return
			}
			src = clusterSourceDTO{
				ID:      s.ID,
				Name:    s.Name,
				BaseURL: s.BaseURL,
				RSSURL:  s.RSSURL,
				LogoURL: s.LogoURL,
			}
			sourceCache[a.SourceID] = src
		}

		outArticles = append(outArticles, clusterArticleDTO{
			ID:          a.ID,
			URL:         a.URL,
			ImageURL:    a.ImageURL,
			Title:       a.Title,
			Snippet:     a.Snippet,
			PublishedAt: a.PublishedAt,
			FetchedAt:   a.FetchedAt,
			Source:      src,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"cluster":  cluster,
		"articles": outArticles,
	})
}
