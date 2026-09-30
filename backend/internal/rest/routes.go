package rest

import (
	"github.com/MatejGadjo/jadro/internal/rest/handlers"
	"github.com/MatejGadjo/jadro/internal/rest/middleware"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis_rate/v10"
)

func RegisterRoutes(
	r *gin.Engine,
	feed *handlers.FeedHandler,
	health *handlers.HealthHandler,
	cluster *handlers.ClusterHandler,
	prerender *handlers.PrerenderHandler,
	push *handlers.PushHandler,
	rl *middleware.RateLimiter,
) {
	api := r.Group("/api")
	api.GET("/health", health.Get)
	api.HEAD("/health", health.Get)
	api.GET("/feed", rl.Limit(redis_rate.PerMinute(30)), feed.GetFeed)
	api.GET("/articles/latest", rl.Limit(redis_rate.PerMinute(60)), cluster.GetLatestArticles)
	api.GET("/clusters/:id", rl.Limit(redis_rate.PerMinute(60)), cluster.GetCluster)
	api.PATCH("/articles/:id/clear-image", rl.LimitStrict(redis_rate.PerMinute(10)), cluster.PatchClearImage)

	// Web Push: read the VAPID key (fail-open), subscribe/unsubscribe (writes,
	// fail-closed like clear-image). All 404 when push is disabled (no keys).
	api.GET("/push/config", rl.Limit(redis_rate.PerMinute(60)), push.GetConfig)
	api.POST("/push/subscribe", rl.LimitStrict(redis_rate.PerMinute(20)), push.Subscribe)
	api.POST("/push/unsubscribe", rl.LimitStrict(redis_rate.PerMinute(20)), push.Unsubscribe)

	// Crawler-only server-rendered pages. nginx forwards bot requests for page
	// routes here as /_prerender<original-path>; humans are served the SPA and
	// never reach this. Catch-all so /, /cluster/:id and /:topic share one route
	// without httprouter static/param conflicts.
	r.GET("/_prerender/*path", rl.Limit(redis_rate.PerMinute(60)), prerender.Render)

	// Dynamic XML sitemap (homepage + topics + recent clusters). nginx routes
	// /sitemap.xml here (exact-match location); referenced by robots.txt.
	r.GET("/sitemap.xml", rl.Limit(redis_rate.PerMinute(60)), feed.GetSitemap)
}
