package middleware

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis_rate/v10"
)

func init() { gin.SetMode(gin.TestMode) }

func newRouter(mw ...gin.HandlerFunc) *gin.Engine {
	r := gin.New()
	r.Use(mw...)
	handler := func(c *gin.Context) {
		if c.Request.Body != nil {
			if _, err := io.ReadAll(c.Request.Body); err != nil {
				c.Status(http.StatusRequestEntityTooLarge)
				return
			}
		}
		c.Status(http.StatusOK)
	}
	r.GET("/x", handler)
	r.POST("/x", handler)
	return r
}

func TestCORS(t *testing.T) {
	r := newRouter(CORS([]string{"https://example.com"}))

	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("Origin", "https://example.com")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "https://example.com" {
		t.Fatalf("allowed origin: got %q", got)
	}

	req = httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("Origin", "https://evil.example")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("disallowed origin must get no ACAO header, got %q", got)
	}

	req = httptest.NewRequest(http.MethodOptions, "/x", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNoContent {
		t.Fatalf("preflight: got %d", w.Code)
	}
}

func TestCORS_EmptyAllowsAll(t *testing.T) {
	r := newRouter(CORS(nil))
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("Origin", "https://anything.example")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Fatalf("got %q, want *", got)
	}
}

func TestBodyLimit(t *testing.T) {
	r := newRouter(BodyLimit(16))

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/x", bytes.NewReader(make([]byte, 8))))
	if w.Code != http.StatusOK {
		t.Fatalf("small body: got %d", w.Code)
	}

	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/x", bytes.NewReader(make([]byte, 64))))
	if w.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("declared large body: got %d", w.Code)
	}

	// Chunked body with no Content-Length is still capped while reading.
	req := httptest.NewRequest(http.MethodPost, "/x", io.NopCloser(bytes.NewReader(make([]byte, 64))))
	req.ContentLength = -1
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("streamed large body: got %d", w.Code)
	}
}

func TestRateLimiter_NilIsNoop(t *testing.T) {
	var rl *RateLimiter
	r := newRouter(rl.Limit(redis_rate.PerMinute(1)), rl.LimitStrict(redis_rate.PerMinute(1)))
	for range 3 {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/x", nil))
		if w.Code != http.StatusOK {
			t.Fatalf("nil limiter must not block, got %d", w.Code)
		}
	}
}
