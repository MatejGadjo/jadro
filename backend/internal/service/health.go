package service

import (
	"context"
	"net/http"
	"time"

	"github.com/MatejGadjo/jadro/internal/config"
	"github.com/MatejGadjo/jadro/internal/logger"
)

type HealthStatus struct {
	OK               bool   `json:"ok"`
	DB               string `json:"db"`
	Redis            string `json:"redis"`
	ArticleCount     int    `json:"article_count"`
	ClusterCount     int    `json:"cluster_count"`
	EmbeddingEnabled bool   `json:"embedding_enabled"`
	Duration         string `json:"duration"`
}

type HealthService struct {
	store HealthStore
	cache HealthCache
	cfg   *config.Config
	log   *logger.Logger
}

func NewHealthService(store HealthStore, cache HealthCache, cfg *config.Config, log *logger.Logger) *HealthService {
	return &HealthService{
		store: store,
		cache: cache,
		cfg:   cfg,
		log:   log,
	}
}

func (s *HealthService) Check(ctx context.Context) (HealthStatus, int) {
	start := time.Now()

	pingCtx := ctx
	if s.cfg.DB.HealthTimeout > 0 {
		var cancel context.CancelFunc
		pingCtx, cancel = context.WithTimeout(ctx, s.cfg.DB.HealthTimeout)
		defer cancel()
	}

	if err := s.store.Ping(pingCtx); err != nil {
		s.log.Warnw("health check failed", "component", "postgres", "error", err)
		return HealthStatus{
			OK:       false,
			DB:       "down",
			Duration: time.Since(start).String(),
		}, http.StatusServiceUnavailable
	}

	articleCount, _ := s.store.ArticleCount(pingCtx)
	clusterCount, _ := s.store.ClusterCount(pingCtx)

	redisStatus := "disabled"
	if s.cache != nil {
		if err := s.cache.Ping(pingCtx); err != nil {
			redisStatus = "down"
		} else {
			redisStatus = "up"
		}
	}

	status := HealthStatus{
		OK:               true,
		DB:               "up",
		Redis:            redisStatus,
		ArticleCount:     articleCount,
		ClusterCount:     clusterCount,
		EmbeddingEnabled: s.cfg.ONNX.RuntimeLibPath != "",
		Duration:         time.Since(start).String(),
	}

	return status, http.StatusOK
}
