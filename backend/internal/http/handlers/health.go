package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/MatejGadjo/NewsAggregator/internal/cache"
	"github.com/MatejGadjo/NewsAggregator/internal/config"
	"github.com/MatejGadjo/NewsAggregator/internal/jobs"
	"github.com/MatejGadjo/NewsAggregator/internal/logger"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

type HealthHandler struct {
	cfg        *config.Config
	log        *logger.Logger
	pool       *pgxpool.Pool
	scheduler  *jobs.Scheduler
	cleanup    *jobs.CleanupJob
	cache      *cache.Client
}

func NewHealthHandler(cfg *config.Config, log *logger.Logger, pool *pgxpool.Pool, scheduler *jobs.Scheduler, cleanup *jobs.CleanupJob, cache *cache.Client) *HealthHandler {
	return &HealthHandler{cfg: cfg, log: log, pool: pool, scheduler: scheduler, cleanup: cleanup, cache: cache}
}

func (h *HealthHandler) Get(c *gin.Context) {
	start := time.Now()

	pingCtx := c.Request.Context()
	if h.cfg.DBHealthTimeout > 0 {
		var cancel context.CancelFunc
		pingCtx, cancel = context.WithTimeout(pingCtx, h.cfg.DBHealthTimeout)
		defer cancel()
	}

	if err := h.pool.Ping(pingCtx); err != nil {
		h.log.Warnw("health check failed", "component", "postgres", "error", err)
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"ok":       false,
			"db":       "down",
			"duration": time.Since(start).String(),
		})
		return
	}

	var articleCount, clusterCount int
	h.pool.QueryRow(pingCtx, "SELECT COUNT(*) FROM articles").Scan(&articleCount)
	h.pool.QueryRow(pingCtx, "SELECT COUNT(*) FROM clusters").Scan(&clusterCount)

	redisStatus := "disabled"
	if h.cache != nil {
		if err := h.cache.Ping(pingCtx); err != nil {
			redisStatus = "down"
		} else {
			redisStatus = "up"
		}
	}

	resp := gin.H{
		"ok":                true,
		"db":                "up",
		"redis":             redisStatus,
		"article_count":     articleCount,
		"cluster_count":     clusterCount,
		"embedding_enabled": h.cfg.ONNXRuntimeLibPath != "",
		"duration":          time.Since(start).String(),
	}
	if t := h.scheduler.LastIngestAt(); t != nil {
		resp["last_ingest_at"] = t.UTC().Format(time.RFC3339)
	}
	if t := h.cleanup.LastCleanupAt(); t != nil {
		resp["last_cleanup_at"] = t.UTC().Format(time.RFC3339)
	}
	c.JSON(http.StatusOK, resp)
}
