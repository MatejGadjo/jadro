package middleware

import (
	"net/http"
	"strconv"

	"github.com/MatejGadjo/NewsAggregator/internal/cache"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis_rate/v10"
)

type RateLimiter struct {
	limiter *redis_rate.Limiter
}

// NewRateLimiter creates a Redis-backed rate limiter. Returns nil if cache is nil (Redis disabled).
func NewRateLimiter(c *cache.Client) *RateLimiter {
	if c == nil {
		return nil
	}
	return &RateLimiter{limiter: redis_rate.NewLimiter(c.Client())}
}

// Limit returns a Gin middleware enforcing the given rate per client IP.
// Safe to call on a nil *RateLimiter — returns a no-op handler (Redis disabled).
func (rl *RateLimiter) Limit(limit redis_rate.Limit) gin.HandlerFunc {
	if rl == nil {
		return func(c *gin.Context) { c.Next() }
	}
	return func(c *gin.Context) {
		key := "rl:" + c.FullPath() + ":" + c.ClientIP()
		res, err := rl.limiter.Allow(c.Request.Context(), key, limit)
		if err != nil {
			// Redis error — fail open, don't block traffic
			c.Next()
			return
		}
		c.Header("X-RateLimit-Limit", strconv.Itoa(int(limit.Rate)))
		c.Header("X-RateLimit-Remaining", strconv.Itoa(res.Remaining))
		if res.Allowed == 0 {
			c.Header("Retry-After", strconv.Itoa(int(res.RetryAfter.Seconds())))
			c.AbortWithStatus(http.StatusTooManyRequests)
			return
		}
		c.Next()
	}
}
