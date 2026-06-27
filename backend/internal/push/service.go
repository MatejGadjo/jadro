// Package push sends the once-daily "Денес во Македонија" Web Push notification:
// the single highest-ranked story of the day (with a digest-card fallback on a
// weak-news day), delivered to every browser subscription. No user accounts —
// subscriptions are keyed by push endpoint.
package push

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/MatejGadjo/NewsAggregator/domain"
	"github.com/MatejGadjo/NewsAggregator/internal/logger"
)

const (
	canonicalBaseURL = "https://jadro.mk"
	defaultImage     = canonicalBaseURL + "/og-image.png"
	// 180×180 brand icon — crisp on high-DPI phones (the 32px favicon looked
	// pixelated upscaled to the Android notification icon size).
	notificationIcon = canonicalBaseURL + "/apple-touch-icon.png"

	candidateClusters = 20            // pool scanned for the day's story (wide enough that a fresh, lower-scored story isn't missed)
	dedupWindowHours  = 24            // don't push the same cluster within this window
	bodyMax           = 120           // notification body truncation
	notificationTag   = "jadro-daily" // collapses to one notification on-device
	sendConcurrency   = 8             // bounded fan-out
	defaultFreshHours = 16            // daily story must have a new article within this many hours
)

// Config holds the push runtime settings the service and sender need.
type Config struct {
	VAPIDPublic  string
	VAPIDPrivate string
	VAPIDSubject string
	MinSources   int
	FreshHours   int
}

// payload is the JSON contract delivered to the service worker (public/sw.js).
type payload struct {
	Title string `json:"title"`
	Body  string `json:"body"`
	Icon  string `json:"icon"`
	Image string `json:"image"`
	URL   string `json:"url"`
	Tag   string `json:"tag"`
}

// Service builds and sends the daily notification.
type Service struct {
	store       Store
	feed        FeedReader
	sender      Sender
	log         *logger.Logger
	minSources  int
	freshWindow time.Duration
}

// NewService wires the production service with a webpush-backed sender.
func NewService(cfg Config, store Store, feed FeedReader, log *logger.Logger) *Service {
	return newService(cfg, store, feed, newWebpushSender(cfg), log)
}

// newService is the testable constructor (inject a fake Sender).
func newService(cfg Config, store Store, feed FeedReader, sender Sender, log *logger.Logger) *Service {
	min := cfg.MinSources
	if min < 1 {
		min = 1
	}
	freshHours := cfg.FreshHours
	if freshHours < 1 {
		freshHours = defaultFreshHours
	}
	return &Service{
		store:       store,
		feed:        feed,
		sender:      sender,
		log:         log,
		minSources:  min,
		freshWindow: time.Duration(freshHours) * time.Hour,
	}
}

// SendDaily selects the day's story (or the digest fallback) and fans it out to
// every subscription, pruning any that the push service reports as gone.
func (s *Service) SendDaily(ctx context.Context) error {
	subs, err := s.store.ListPushSubscriptions(ctx)
	if err != nil {
		return err
	}
	if len(subs) == 0 {
		s.log.Infow("push: no subscriptions, skipping daily send")
		return nil
	}

	p, clusterID := s.buildPayload(ctx)
	body, err := json.Marshal(p)
	if err != nil {
		return err
	}

	sent, pruned := s.fanOut(ctx, body, subs)

	if err := s.store.InsertPushSend(ctx, clusterID); err != nil {
		s.log.Warnw("push: failed to record send", "error", err)
	}

	s.log.Infow("push: daily send complete",
		"cluster_id", clusterID, "title", p.Title,
		"subscriptions", len(subs), "delivered", sent, "pruned", pruned,
	)
	return nil
}

// buildPayload returns the notification and the cluster id it represents (0 for
// the digest fallback). It picks the highest-scored cluster that clears the
// source-count gate and wasn't already pushed within the dedup window.
func (s *Service) buildPayload(ctx context.Context) (payload, int64) {
	clusters, err := s.feed.ListTopClustersByScore(ctx, candidateClusters)
	if err != nil {
		s.log.Warnw("push: failed to list top clusters, using digest fallback", "error", err)
		return s.digestPayload(), 0
	}

	for _, cl := range clusters {
		if cl.SourceCount < s.minSources {
			continue
		}
		// Freshness: the story must have a new article within the window, so the
		// 09:00 push is "what happened recently" and not a day-old topper.
		if time.Since(cl.LastSeenAt) > s.freshWindow {
			continue
		}
		recent, err := s.store.ClusterSentRecently(ctx, cl.ID, dedupWindowHours)
		if err != nil {
			s.log.Warnw("push: dedup check failed", "cluster_id", cl.ID, "error", err)
			continue
		}
		if recent {
			continue
		}
		return s.storyPayload(ctx, cl), cl.ID
	}

	return s.digestPayload(), 0
}

func (s *Service) storyPayload(ctx context.Context, cl domain.Cluster) payload {
	image := defaultImage
	if arts, err := s.feed.ListClusterArticlesWithSources(ctx, cl.ID); err == nil {
		if img := firstImage(arts); img != "" {
			image = img
		}
	}
	return payload{
		Title: capitalizeFirst(clusterTitle(cl)),
		Body:  storyBody(cl),
		Icon:  notificationIcon,
		Image: image,
		URL:   canonicalBaseURL + "/cluster/" + strconv.FormatInt(cl.ID, 10),
		Tag:   notificationTag,
	}
}

func (s *Service) digestPayload() payload {
	return payload{
		Title: "Денес во Македонија",
		Body:  "Најважните вести денес, собрани на едно место.",
		Icon:  notificationIcon,
		Image: defaultImage,
		URL:   canonicalBaseURL + "/",
		Tag:   notificationTag,
	}
}

// fanOut delivers the payload to all subscriptions with bounded concurrency and
// prunes any endpoint the push service reports as gone (404/410). Returns the
// delivered and pruned counts.
func (s *Service) fanOut(ctx context.Context, body []byte, subs []domain.PushSubscription) (int, int) {
	var (
		mu     sync.Mutex
		sent   int
		pruned int
		sem    = make(chan struct{}, sendConcurrency)
		wg     sync.WaitGroup
	)

	for _, sub := range subs {
		wg.Add(1)
		sem <- struct{}{}
		go func(sub domain.PushSubscription) {
			defer wg.Done()
			defer func() { <-sem }()

			status, err := s.sender.Send(ctx, body, sub)
			if err != nil {
				s.log.Warnw("push: send failed", "endpoint", truncateEndpoint(sub.Endpoint), "error", err)
				return
			}
			if isExpired(status) {
				if delErr := s.store.DeletePushSubscriptionByEndpoint(ctx, sub.Endpoint); delErr != nil {
					s.log.Warnw("push: prune failed", "endpoint", truncateEndpoint(sub.Endpoint), "error", delErr)
				}
				mu.Lock()
				pruned++
				mu.Unlock()
				return
			}
			if status >= 200 && status < 300 {
				mu.Lock()
				sent++
				mu.Unlock()
			}
		}(sub)
	}

	wg.Wait()
	return sent, pruned
}

// --- payload helpers (mirror prerender.go's title/image/truncate logic) ---

func clusterTitle(cl domain.Cluster) string {
	if strings.TrimSpace(cl.DisplayTitle) != "" {
		return cl.DisplayTitle
	}
	return cl.Headline
}

func storyBody(cl domain.Cluster) string {
	return capitalizeFirst(truncate(cl.Summary, bodyMax))
}

// capitalizeFirst upper-cases the first rune. Notification titles/bodies should
// not start lowercase, but RSS headlines/summaries occasionally do.
func capitalizeFirst(s string) string {
	if s == "" {
		return s
	}
	r := []rune(s)
	r[0] = unicode.ToUpper(r[0])
	return string(r)
}

func firstImage(arts []domain.ClusterArticleSource) string {
	for _, a := range arts {
		if a.ImageURL != nil && strings.TrimSpace(*a.ImageURL) != "" {
			return toHTTPS(*a.ImageURL)
		}
	}
	return ""
}

func toHTTPS(u string) string {
	if strings.HasPrefix(u, "http://") {
		return "https://" + strings.TrimPrefix(u, "http://")
	}
	return u
}

func truncate(s string, max int) string {
	s = strings.TrimSpace(s)
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return strings.TrimSpace(string(r[:max])) + "…"
}

// truncateEndpoint shortens a push endpoint for log lines (they carry an opaque
// token tail we don't need to log in full).
func truncateEndpoint(e string) string {
	const n = 60
	if len(e) <= n {
		return e
	}
	return e[:n] + "…"
}
