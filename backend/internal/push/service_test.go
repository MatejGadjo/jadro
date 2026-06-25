package push

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"testing"

	"github.com/MatejGadjo/NewsAggregator/domain"
	"github.com/MatejGadjo/NewsAggregator/internal/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- fakes ---

type fakeStore struct {
	mu          sync.Mutex
	subs        []domain.PushSubscription
	recent      map[int64]bool // cluster ids "sent within window"
	deleted     []string
	sentCluster int64
	sentCalled  bool
}

func (f *fakeStore) ListPushSubscriptions(context.Context) ([]domain.PushSubscription, error) {
	return f.subs, nil
}
func (f *fakeStore) DeletePushSubscriptionByEndpoint(_ context.Context, endpoint string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.deleted = append(f.deleted, endpoint)
	return nil
}
func (f *fakeStore) InsertPushSend(_ context.Context, clusterID int64) error {
	f.sentCluster = clusterID
	f.sentCalled = true
	return nil
}
func (f *fakeStore) ClusterSentRecently(_ context.Context, clusterID int64, _ int) (bool, error) {
	return f.recent[clusterID], nil
}

type fakeFeed struct {
	clusters []domain.Cluster
	articles map[int64][]domain.ClusterArticleSource
}

func (f *fakeFeed) ListTopClustersByScore(_ context.Context, limit int32) ([]domain.Cluster, error) {
	if int(limit) < len(f.clusters) {
		return f.clusters[:limit], nil
	}
	return f.clusters, nil
}
func (f *fakeFeed) ListClusterArticlesWithSources(_ context.Context, clusterID int64) ([]domain.ClusterArticleSource, error) {
	return f.articles[clusterID], nil
}

type fakeSender struct {
	mu       sync.Mutex
	statusBy map[string]int // endpoint -> status (default 201)
	payloads [][]byte
	calls    []string
}

func (f *fakeSender) Send(_ context.Context, payload []byte, sub domain.PushSubscription) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, sub.Endpoint)
	f.payloads = append(f.payloads, payload)
	if s, ok := f.statusBy[sub.Endpoint]; ok {
		return s, nil
	}
	return http.StatusCreated, nil
}

func testLogger(t *testing.T) *logger.Logger {
	t.Helper()
	log, err := logger.NewLogger()
	require.NoError(t, err)
	return log
}

func strptr(s string) *string { return &s }

func sub(endpoint string) domain.PushSubscription {
	return domain.PushSubscription{Endpoint: endpoint, P256dh: "p", Auth: "a"}
}

func decodePayload(t *testing.T, b []byte) payload {
	t.Helper()
	var p payload
	require.NoError(t, json.Unmarshal(b, &p))
	return p
}

// --- tests ---

func TestSendDaily_PicksTopClusterClearingGate(t *testing.T) {
	store := &fakeStore{subs: []domain.PushSubscription{sub("e1")}, recent: map[int64]bool{}}
	feed := &fakeFeed{
		clusters: []domain.Cluster{
			{ID: 1, Headline: "weak", SourceCount: 1, Score: 9},     // below gate
			{ID: 2, Headline: "strong", SourceCount: 4, Score: 8},   // first to clear gate
			{ID: 3, Headline: "stronger", SourceCount: 5, Score: 7}, // lower score
		},
		articles: map[int64][]domain.ClusterArticleSource{
			2: {{ImageURL: strptr("http://cdn/x.jpg")}},
		},
	}
	sender := &fakeSender{}
	svc := newService(Config{MinSources: 3}, store, feed, sender, testLogger(t))

	require.NoError(t, svc.SendDaily(context.Background()))

	require.Len(t, sender.payloads, 1)
	p := decodePayload(t, sender.payloads[0])
	assert.Equal(t, "strong", p.Title)
	assert.Contains(t, p.URL, "/cluster/2")
	assert.Equal(t, "https://cdn/x.jpg", p.Image, "http image upgraded to https")
	assert.Contains(t, p.Body, "4 извори")
	assert.True(t, store.sentCalled)
	assert.Equal(t, int64(2), store.sentCluster)
}

func TestSendDaily_SkipsDedupedCluster(t *testing.T) {
	store := &fakeStore{subs: []domain.PushSubscription{sub("e1")}, recent: map[int64]bool{2: true}}
	feed := &fakeFeed{
		clusters: []domain.Cluster{
			{ID: 2, Headline: "already-sent", SourceCount: 5, Score: 9},
			{ID: 3, Headline: "next", SourceCount: 3, Score: 8},
		},
		articles: map[int64][]domain.ClusterArticleSource{},
	}
	sender := &fakeSender{}
	svc := newService(Config{MinSources: 3}, store, feed, sender, testLogger(t))

	require.NoError(t, svc.SendDaily(context.Background()))

	p := decodePayload(t, sender.payloads[0])
	assert.Equal(t, "next", p.Title)
	assert.Equal(t, int64(3), store.sentCluster)
}

func TestSendDaily_DigestFallbackWhenNoneQualify(t *testing.T) {
	store := &fakeStore{subs: []domain.PushSubscription{sub("e1")}, recent: map[int64]bool{}}
	feed := &fakeFeed{
		clusters: []domain.Cluster{
			{ID: 1, Headline: "weak1", SourceCount: 1},
			{ID: 2, Headline: "weak2", SourceCount: 2},
		},
		articles: map[int64][]domain.ClusterArticleSource{},
	}
	sender := &fakeSender{}
	svc := newService(Config{MinSources: 3}, store, feed, sender, testLogger(t))

	require.NoError(t, svc.SendDaily(context.Background()))

	p := decodePayload(t, sender.payloads[0])
	assert.Equal(t, "Денес во Македонија", p.Title)
	assert.Equal(t, canonicalBaseURL+"/", p.URL)
	assert.Equal(t, defaultImage, p.Image)
	assert.True(t, store.sentCalled)
	assert.Equal(t, int64(0), store.sentCluster, "digest send records NULL cluster (0)")
}

func TestSendDaily_PrunesExpiredSubscriptions(t *testing.T) {
	store := &fakeStore{
		subs:   []domain.PushSubscription{sub("good"), sub("gone"), sub("good2")},
		recent: map[int64]bool{},
	}
	feed := &fakeFeed{
		clusters: []domain.Cluster{{ID: 5, Headline: "story", SourceCount: 3}},
		articles: map[int64][]domain.ClusterArticleSource{},
	}
	sender := &fakeSender{statusBy: map[string]int{"gone": http.StatusGone}}
	svc := newService(Config{MinSources: 3}, store, feed, sender, testLogger(t))

	require.NoError(t, svc.SendDaily(context.Background()))

	assert.Len(t, sender.calls, 3, "every subscription is attempted")
	assert.Equal(t, []string{"gone"}, store.deleted, "only the 410 endpoint is pruned")
}

func TestSendDaily_NoSubscriptionsIsNoOp(t *testing.T) {
	store := &fakeStore{subs: nil, recent: map[int64]bool{}}
	feed := &fakeFeed{}
	sender := &fakeSender{}
	svc := newService(Config{MinSources: 3}, store, feed, sender, testLogger(t))

	require.NoError(t, svc.SendDaily(context.Background()))
	assert.Empty(t, sender.calls)
	assert.False(t, store.sentCalled, "no send recorded when there are no subscribers")
}
