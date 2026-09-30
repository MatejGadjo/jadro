package service

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/MatejGadjo/jadro/domain"
	"github.com/MatejGadjo/jadro/internal/logger"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
)

func TestHTTPImageProber_IsBroken(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/ok.jpg", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "image/jpeg")
	})
	mux.HandleFunc("/gone.jpg", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusGone)
	})
	mux.HandleFunc("/html.jpg", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
	})
	mux.HandleFunc("/forbidden.jpg", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	})
	mux.HandleFunc("/error.jpg", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	mux.HandleFunc("/nohead.jpg", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodHead {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	p := newHTTPImageProber(true)
	cases := map[string]bool{
		srv.URL + "/ok.jpg":        false,
		srv.URL + "/missing.jpg":   true, // mux 404
		srv.URL + "/gone.jpg":      true,
		srv.URL + "/html.jpg":      true,
		srv.URL + "/forbidden.jpg": false, // bot protection must not wipe images
		srv.URL + "/error.jpg":     false,
		srv.URL + "/nohead.jpg":    true, // falls back to GET
		"ftp://example.com/a.jpg":  true,
		"not a url":                true,
	}
	for u, want := range cases {
		if got := p.IsBroken(context.Background(), u); got != want {
			t.Errorf("IsBroken(%q) = %v, want %v", u, got, want)
		}
	}
}

func TestHTTPImageProber_BlocksPrivateAddresses(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	// The production prober refuses to dial loopback, so a 404 from an
	// internal service is never observed and the image is kept.
	p := newHTTPImageProber(false)
	if p.IsBroken(context.Background(), srv.URL+"/x.jpg") {
		t.Fatal("expected loopback target to be blocked, not reported broken")
	}
}

type fakeProber struct {
	broken bool
	calls  int
}

func (f *fakeProber) IsBroken(context.Context, string) bool {
	f.calls++
	return f.broken
}

type fakeClusterStore struct {
	ClusterStore
	article domain.Article
	getErr  error
	cleared bool
}

func (f *fakeClusterStore) GetArticleByID(context.Context, int64) (domain.Article, error) {
	return f.article, f.getErr
}

func (f *fakeClusterStore) ClearArticleImageURL(context.Context, int64) error {
	f.cleared = true
	return nil
}

type fakeCache struct {
	data map[string][]byte
}

func (f *fakeCache) Get(_ context.Context, key string) ([]byte, error) {
	if v, ok := f.data[key]; ok {
		return v, nil
	}
	return nil, errors.New("miss")
}

func (f *fakeCache) Set(_ context.Context, key string, val []byte, _ time.Duration) error {
	f.data[key] = val
	return nil
}

func (f *fakeCache) DeleteByPattern(context.Context, string) error { return nil }

func TestClearArticleImage(t *testing.T) {
	log := &logger.Logger{SugaredLogger: zap.NewNop().Sugar()}
	img := "https://example.com/a.jpg"

	t.Run("working image is kept and verdict cached", func(t *testing.T) {
		store := &fakeClusterStore{article: domain.Article{ImageURL: &img}}
		prober := &fakeProber{broken: false}
		cache := &fakeCache{data: map[string][]byte{}}
		svc := NewClusterService(store, cache, prober, log)

		for range 3 {
			if err := svc.ClearArticleImage(context.Background(), 1); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		}
		if store.cleared {
			t.Fatal("working image must not be cleared")
		}
		if prober.calls != 1 {
			t.Fatalf("expected 1 probe (then cached), got %d", prober.calls)
		}
	})

	t.Run("broken image is cleared", func(t *testing.T) {
		store := &fakeClusterStore{article: domain.Article{ImageURL: &img}}
		svc := NewClusterService(store, nil, &fakeProber{broken: true}, log)
		if err := svc.ClearArticleImage(context.Background(), 1); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !store.cleared {
			t.Fatal("broken image should be cleared")
		}
	})

	t.Run("missing article or image is not found", func(t *testing.T) {
		svc := NewClusterService(&fakeClusterStore{getErr: pgx.ErrNoRows}, nil, &fakeProber{}, log)
		if err := svc.ClearArticleImage(context.Background(), 1); !errors.Is(err, pgx.ErrNoRows) {
			t.Fatalf("want ErrNoRows, got %v", err)
		}
		svc = NewClusterService(&fakeClusterStore{}, nil, &fakeProber{}, log)
		if err := svc.ClearArticleImage(context.Background(), 1); !errors.Is(err, pgx.ErrNoRows) {
			t.Fatalf("want ErrNoRows for nil image, got %v", err)
		}
	})
}
