package push

import (
	"context"
	"io"
	"net/http"
	"strings"

	"github.com/MatejGadjo/jadro/domain"
	webpush "github.com/SherClockHolmes/webpush-go"
)

const notificationTTL = 24 * 60 * 60 // seconds; one day — the daily digest cadence

// webpushSender is the production Sender backed by github.com/SherClockHolmes/webpush-go.
type webpushSender struct {
	vapidPublic  string
	vapidPrivate string
	subscriber   string
}

func newWebpushSender(cfg Config) *webpushSender {
	return &webpushSender{
		vapidPublic:  cfg.VAPIDPublic,
		vapidPrivate: cfg.VAPIDPrivate,
		// webpush-go adds "mailto:" to anything that isn't an https URL, so
		// drop ours to avoid sending "mailto:mailto:...".
		subscriber: strings.TrimPrefix(cfg.VAPIDSubject, "mailto:"),
	}
}

func (s *webpushSender) Send(ctx context.Context, payload []byte, sub domain.PushSubscription) (int, error) {
	resp, err := webpush.SendNotificationWithContext(ctx, payload, &webpush.Subscription{
		Endpoint: sub.Endpoint,
		Keys:     webpush.Keys{P256dh: sub.P256dh, Auth: sub.Auth},
	}, &webpush.Options{
		Subscriber:      s.subscriber,
		VAPIDPublicKey:  s.vapidPublic,
		VAPIDPrivateKey: s.vapidPrivate,
		TTL:             notificationTTL,
		Urgency:         webpush.UrgencyNormal,
	})
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body) // drain so the connection can be reused
	return resp.StatusCode, nil
}

// isExpired reports whether the push endpoint says the subscription is gone and
// should be pruned (RFC 8030: 404 Not Found / 410 Gone).
func isExpired(status int) bool {
	return status == http.StatusNotFound || status == http.StatusGone
}
