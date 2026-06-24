package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/MatejGadjo/NewsAggregator/internal/topics"
	"github.com/gin-gonic/gin"
)

type FeedHandler struct {
	svc FeedServicer
}

func NewFeedHandler(svc FeedServicer) *FeedHandler {
	return &FeedHandler{svc: svc}
}

func (h *FeedHandler) GetFeed(c *gin.Context) {
	ctx := c.Request.Context()

	requestedTopic := strings.TrimSpace(c.Query("topic"))
	var topicFilter *string
	if requestedTopic != "" {
		if !topics.IsAllowedTopicName(requestedTopic) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid topic"})
			return
		}
		topicFilter = &requestedTopic
	}

	page := 0
	if p := strings.TrimSpace(c.Query("page")); p != "" {
		if n, err := strconv.Atoi(p); err == nil && n > 0 {
			page = n
		}
	}

	raw, err := h.svc.GetFeed(ctx, topicFilter, page)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch feed"})
		return
	}
	c.Data(http.StatusOK, "application/json; charset=utf-8", raw)
}

// GetSitemap serves the dynamic XML sitemap (homepage + topics + recent
// clusters). nginx routes /sitemap.xml here; crawlers fetch it from robots.txt.
func (h *FeedHandler) GetSitemap(c *gin.Context) {
	raw, err := h.svc.Sitemap(c.Request.Context())
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}
	c.Data(http.StatusOK, "application/xml; charset=utf-8", raw)
}
