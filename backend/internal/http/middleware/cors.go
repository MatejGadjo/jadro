package middleware

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// CORSConfig holds configuration for CORS middleware.
type CORSConfig struct {
	// AllowedOrigins is a list of origins that are allowed to make requests.
	// If empty, all origins are allowed (use with caution in production).
	AllowedOrigins []string

	// AllowedMethods is a list of HTTP methods that are allowed.
	// Defaults to common methods if empty.
	AllowedMethods []string

	// AllowedHeaders is a list of headers that are allowed in requests.
	// Defaults to common headers if empty.
	AllowedHeaders []string

	// ExposedHeaders is a list of headers that can be exposed to the client.
	ExposedHeaders []string

	// AllowCredentials indicates whether credentials can be included in requests.
	AllowCredentials bool

	// MaxAge is the maximum age (in seconds) for preflight requests.
	MaxAge int
}

// DefaultCORSConfig returns a CORS configuration with sensible defaults.
func DefaultCORSConfig() *CORSConfig {
	return &CORSConfig{
		AllowedOrigins:   []string{"*"}, // Allow all origins by default
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS", "HEAD"},
		AllowedHeaders:   []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Requested-With"},
		ExposedHeaders:   []string{},
		AllowCredentials: false,
		MaxAge:           3600, // 1 hour
	}
}

// CORS returns a Gin middleware handler that implements CORS.
func CORS(config *CORSConfig) gin.HandlerFunc {
	if config == nil {
		config = DefaultCORSConfig()
	}

	// Set defaults if not provided
	if len(config.AllowedMethods) == 0 {
		config.AllowedMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS", "HEAD"}
	}
	if len(config.AllowedHeaders) == 0 {
		config.AllowedHeaders = []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Requested-With"}
	}

	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		// Handle preflight OPTIONS request
		if c.Request.Method == "OPTIONS" {
			setCORSHeaders(c, config, origin)
			c.AbortWithStatus(204) // No Content
			return
		}

		// Set CORS headers for actual requests
		setCORSHeaders(c, config, origin)
		c.Next()
	}
}

// setCORSHeaders sets the appropriate CORS headers based on the configuration.
func setCORSHeaders(c *gin.Context, config *CORSConfig, origin string) {
	// Determine allowed origin
	allowedOrigin := determineAllowedOrigin(config.AllowedOrigins, origin)
	if allowedOrigin != "" {
		c.Header("Access-Control-Allow-Origin", allowedOrigin)
	}

	// Set credentials header if enabled
	if config.AllowCredentials && allowedOrigin != "*" {
		c.Header("Access-Control-Allow-Credentials", "true")
	}

	// Set allowed methods
	if len(config.AllowedMethods) > 0 {
		c.Header("Access-Control-Allow-Methods", strings.Join(config.AllowedMethods, ", "))
	}

	// Set allowed headers
	if len(config.AllowedHeaders) > 0 {
		c.Header("Access-Control-Allow-Headers", strings.Join(config.AllowedHeaders, ", "))
	}

	// Set exposed headers
	if len(config.ExposedHeaders) > 0 {
		c.Header("Access-Control-Expose-Headers", strings.Join(config.ExposedHeaders, ", "))
	}

	// Set max age for preflight
	if config.MaxAge > 0 {
		c.Header("Access-Control-Max-Age", strconv.Itoa(config.MaxAge))
	}
}

// determineAllowedOrigin determines which origin to allow based on the configuration.
func determineAllowedOrigin(allowedOrigins []string, requestOrigin string) string {
	// If no origins specified, allow all
	if len(allowedOrigins) == 0 {
		return "*"
	}

	// If wildcard is present, allow all
	for _, allowed := range allowedOrigins {
		if allowed == "*" {
			return "*"
		}
	}

	// If no request origin, return empty (no CORS header)
	if requestOrigin == "" {
		return ""
	}

	// Check if the request origin is in the allowed list
	for _, allowed := range allowedOrigins {
		if allowed == requestOrigin {
			return requestOrigin
		}
	}

	// Origin not allowed
	return ""
}
