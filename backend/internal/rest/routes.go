package rest

import (
	"github.com/MatejGadjo/NewsAggregator/internal/rest/handlers"
	"github.com/MatejGadjo/NewsAggregator/internal/rest/middleware"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis_rate/v10"
)

func RegisterRoutes(
	r *gin.Engine,
	feed *handlers.FeedHandler,
	health *handlers.HealthHandler,
	cluster *handlers.ClusterHandler,
	rl *middleware.RateLimiter,
) {
	api := r.Group("/api")
	api.GET("/health", health.Get)
	api.HEAD("/health", health.Get)
	api.GET("/feed", rl.Limit(redis_rate.PerMinute(30)), feed.GetFeed)
	api.GET("/articles/latest", rl.Limit(redis_rate.PerMinute(60)), cluster.GetLatestArticles)
	api.GET("/clusters/:id", rl.Limit(redis_rate.PerMinute(60)), cluster.GetCluster)
	api.PATCH("/articles/:id/clear-image", rl.LimitStrict(redis_rate.PerMinute(10)), cluster.PatchClearImage)
}
