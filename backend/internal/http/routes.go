package http

import (
	"github.com/MatejGadjo/NewsAggregator/internal/config"
	"github.com/MatejGadjo/NewsAggregator/internal/http/handlers"
	"github.com/MatejGadjo/NewsAggregator/internal/http/middleware"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis_rate/v10"
)

func RegisterRoutes(
	r *gin.Engine,
	cfg *config.Config,
	feed *handlers.FeedHandler,
	health *handlers.HealthHandler,
	cluster *handlers.ClusterHandler,
	internal *handlers.InternalHandler,
	rl *middleware.RateLimiter,
) {
	api := r.Group("/api")
	api.GET("/health", health.Get)
	api.GET("/feed", rl.Limit(redis_rate.PerMinute(30)), feed.GetFeed)
	api.GET("/articles/latest", rl.Limit(redis_rate.PerMinute(60)), cluster.GetLatestArticles)
	api.GET("/clusters/:id", rl.Limit(redis_rate.PerMinute(60)), cluster.GetCluster)
	api.PATCH("/articles/:id/clear-image", rl.Limit(redis_rate.PerMinute(20)), cluster.PatchClearImage)

	internalGroup := r.Group("/internal")
	internalGroup.Use(middleware.BearerAuth(cfg.InternalAPIToken))
	internalGroup.POST("/ingest", internal.TriggerIngest)
}
