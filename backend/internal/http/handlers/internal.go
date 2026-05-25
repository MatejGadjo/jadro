package handlers

import (
	"net/http"

	"github.com/MatejGadjo/NewsAggregator/internal/jobs"
	"github.com/MatejGadjo/NewsAggregator/internal/logger"
	"github.com/gin-gonic/gin"
)

type InternalHandler struct {
	log       *logger.Logger
	scheduler *jobs.Scheduler
}

func NewInternalHandler(log *logger.Logger, scheduler *jobs.Scheduler) *InternalHandler {
	return &InternalHandler{log: log, scheduler: scheduler}
}

func (h *InternalHandler) TriggerIngest(c *gin.Context) {
	if !h.scheduler.RunOnce() {
		c.JSON(http.StatusConflict, gin.H{"ok": false, "message": "ingest already in progress"})
		return
	}
	h.log.Infow("manual ingest triggered", "remote_ip", c.ClientIP())
	c.JSON(http.StatusAccepted, gin.H{"ok": true, "message": "ingest triggered"})
}
