package handlers

import "github.com/gin-gonic/gin"

type HealthHandler struct {
	svc HealthServicer
}

func NewHealthHandler(svc HealthServicer) *HealthHandler {
	return &HealthHandler{svc: svc}
}

func (h *HealthHandler) Get(c *gin.Context) {
	status, httpStatus := h.svc.Check(c.Request.Context())
	c.JSON(httpStatus, status)
}
