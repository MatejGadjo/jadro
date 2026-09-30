package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

type fakeFeedService struct {
	gotPage int
}

func (f *fakeFeedService) GetFeed(_ context.Context, _ *string, page int) ([]byte, error) {
	f.gotPage = page
	return []byte(`{}`), nil
}
func (f *fakeFeedService) Sitemap(context.Context) ([]byte, error) { return nil, nil }
func (f *fakeFeedService) WarmCache(context.Context)               {}

func TestGetFeed_PageValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cases := []struct {
		query    string
		wantCode int
		wantPage int
	}{
		{"", http.StatusOK, 0},
		{"?page=0", http.StatusOK, 0},
		{"?page=3", http.StatusOK, 3},
		{"?page=100", http.StatusOK, 100},
		{"?page=101", http.StatusBadRequest, 0},
		{"?page=-1", http.StatusBadRequest, 0},
		{"?page=abc", http.StatusBadRequest, 0},
		{"?page=99999999999999999999", http.StatusBadRequest, 0},
	}
	for _, tc := range cases {
		svc := &fakeFeedService{}
		r := gin.New()
		r.GET("/api/feed", NewFeedHandler(svc).GetFeed)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/feed"+tc.query, nil))
		if w.Code != tc.wantCode {
			t.Errorf("%s: code %d, want %d", tc.query, w.Code, tc.wantCode)
		}
		if tc.wantCode == http.StatusOK && svc.gotPage != tc.wantPage {
			t.Errorf("%s: page %d, want %d", tc.query, svc.gotPage, tc.wantPage)
		}
	}
}
