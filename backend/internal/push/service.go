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

	"github.com/MatejGadjo/NewsAggregator/domain"
	"github.com/MatejGadjo/NewsAggregator/internal/logger"
)

const (
	canonicalBaseURL = "https://jadro.mk"
	defaultImage     = canonicalBaseURL + "/og-image.png"
	notificationIcon = canonicalBaseURL + "/favicon-32x32.png"

	candidateClusters = 10            // pool scanned for the day's story
	dedupWindowHours  = 24            // don't push the same cluster within this window
	bodyMax           = 120           // notification body truncation
	notificationTag   = "jadro-daily" // collapses to one notification on-device
	sendConcurrency   = 8             // bounded fan-out
)

// Config holds the push runtime settings the service and sender need.
type Config struct {
	VAPIDPublic  string
	VAPIDPrivate string
	VAPIDSubject string
	MinSources   int
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
	store      Store
	feed       FeedReader
	sender     Sender
	log        *logger.Logger
	minSources int
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
	return &Service{store: store, feed: feed, sender: sender, log: log, minSources: min}
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
		Title: clusterTitle(cl),
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
	summary := truncate(cl.Summary, bodyMax)
	if cl.SourceCount >= 2 {
		prefix := strconv.Itoa(cl.SourceCount) + " извори"
		if summary == "" {
			return prefix
		}
		return prefix + " · " + summary
	}
	return summary
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
