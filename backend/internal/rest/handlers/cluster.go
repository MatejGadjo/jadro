package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/MatejGadjo/NewsAggregator/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

type ClusterHandler struct {
	svc ClusterServicer
}

func NewClusterHandler(svc ClusterServicer) *ClusterHandler {
	return &ClusterHandler{svc: svc}
}

func (h *ClusterHandler) GetCluster(c *gin.Context) {
	clusterID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || clusterID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid cluster id"})
		return
	}

	raw, status, err := h.svc.GetCluster(c.Request.Context(), clusterID)
	if err != nil {
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	c.Data(http.StatusOK, "application/json; charset=utf-8", raw)
}

func (h *ClusterHandler) GetLatestArticles(c *gin.Context) {
	limit := 8 // default when ?limit= is not provided, capped at 20
	if l := c.Query("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 {
			if n > 20 {
				n = 20
			}
			limit = n
		}
	}

	articles, err := h.svc.GetLatestArticles(c.Request.Context(), limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch latest articles"})
		return
	}
	if articles == nil {
		articles = []service.LatestArticleItem{}
	}
	c.JSON(http.StatusOK, gin.H{"articles": articles})
}

func (h *ClusterHandler) PatchClearImage(c *gin.Context) {
	articleID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || articleID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid article id"})
		return
	}

	if err := h.svc.ClearArticleImage(c.Request.Context(), articleID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "article not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to clear image"})
		return
	}
	c.Status(http.StatusNoContent)
}
