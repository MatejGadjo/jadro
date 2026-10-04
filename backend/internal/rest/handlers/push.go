package handlers

import (
	"context"
	"net/http"
	"net/url"
	"strings"

	"github.com/MatejGadjo/jadro/domain"
	"github.com/MatejGadjo/jadro/internal/logger"
	"github.com/gin-gonic/gin"
)

// isAllowedPushEndpoint restricts subscription endpoints to known push-service
// hosts over HTTPS. Without this, the daily send job (which POSTs to whatever
// endpoint is stored) could be used as an SSRF primitive by registering an
// internal/arbitrary URL. New push services would need a line added here.
func isAllowedPushEndpoint(endpoint string) bool {
	u, err := url.Parse(endpoint)
	if err != nil || u.Scheme != "https" || u.Host == "" {
		return false
	}
	host := strings.ToLower(u.Hostname())
	switch {
	case host == "fcm.googleapis.com" || host == "android.googleapis.com":
		return true // Chrome / Android (FCM)
	case strings.HasSuffix(host, ".push.services.mozilla.com"):
		return true // Firefox
	case strings.HasSuffix(host, ".notify.windows.com"):
		return true // Edge / Windows (WNS)
	case host == "web.push.apple.com" || strings.HasSuffix(host, ".push.apple.com"):
		return true // Safari / iOS
	default:
		return false
	}
}

// PushHandler exposes the browser-facing Web Push endpoints: the VAPID public
// key (so a browser can subscribe) and subscribe/unsubscribe. When push is
// disabled (no VAPID keys) every endpoint returns 404 so the frontend can
// silently no-op.
type PushHandler struct {
	store       PushStore
	vapidPublic string
	log         *logger.Logger
}

// PushStore persists browser push subscriptions.
type PushStore interface {
	UpsertPushSubscription(ctx context.Context, sub domain.PushSubscription) error
	DeletePushSubscriptionByEndpoint(ctx context.Context, endpoint string) error
}

func NewPushHandler(store PushStore, vapidPublic string, log *logger.Logger) *PushHandler {
	return &PushHandler{store: store, vapidPublic: vapidPublic, log: log}
}

func (h *PushHandler) enabled() bool { return h.vapidPublic != "" }

// GetConfig returns the VAPID public key the browser needs to subscribe.
func (h *PushHandler) GetConfig(c *gin.Context) {
	if !h.enabled() {
		c.Status(http.StatusNotFound)
		return
	}
	c.JSON(http.StatusOK, gin.H{"vapidPublicKey": h.vapidPublic})
}

// Real endpoints are a few hundred bytes and keys under 100; the caps keep
// junk subscriptions from bloating the table.
const (
	maxEndpointLen = 1024
	maxKeyLen      = 256
)

type subscribeRequest struct {
	Endpoint string `json:"endpoint"`
	Keys     struct {
		P256dh string `json:"p256dh"`
		Auth   string `json:"auth"`
	} `json:"keys"`
}

// Subscribe upserts a browser push subscription (keyed by endpoint).
func (h *PushHandler) Subscribe(c *gin.Context) {
	if !h.enabled() {
		c.Status(http.StatusNotFound)
		return
	}
	var req subscribeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Status(http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.Endpoint) == "" || req.Keys.P256dh == "" || req.Keys.Auth == "" {
		c.Status(http.StatusBadRequest)
		return
	}
	if len(req.Endpoint) > maxEndpointLen || len(req.Keys.P256dh) > maxKeyLen || len(req.Keys.Auth) > maxKeyLen {
		c.Status(http.StatusBadRequest)
		return
	}
	if !isAllowedPushEndpoint(req.Endpoint) {
		c.Status(http.StatusBadRequest)
		return
	}
	err := h.store.UpsertPushSubscription(c.Request.Context(), domain.PushSubscription{
		Endpoint: req.Endpoint,
		P256dh:   req.Keys.P256dh,
		Auth:     req.Keys.Auth,
	})
	if err != nil {
		h.log.Errorw("push: subscribe failed", "error", err)
		c.Status(http.StatusInternalServerError)
		return
	}
	c.Status(http.StatusNoContent)
}

type unsubscribeRequest struct {
	Endpoint string `json:"endpoint"`
}

// Unsubscribe removes a subscription by endpoint.
func (h *PushHandler) Unsubscribe(c *gin.Context) {
	if !h.enabled() {
		c.Status(http.StatusNotFound)
		return
	}
	var req unsubscribeRequest
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Endpoint) == "" {
		c.Status(http.StatusBadRequest)
		return
	}
	if err := h.store.DeletePushSubscriptionByEndpoint(c.Request.Context(), req.Endpoint); err != nil {
		h.log.Errorw("push: unsubscribe failed", "error", err)
		c.Status(http.StatusInternalServerError)
		return
	}
	c.Status(http.StatusNoContent)
}
