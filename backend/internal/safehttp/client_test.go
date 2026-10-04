package safehttp

import (
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestIsPublicIP(t *testing.T) {
	public := []string{"8.8.8.8", "1.1.1.1", "2606:4700::1111"}
	blocked := []string{
		"127.0.0.1", "10.0.0.5", "172.18.0.1", "192.168.1.1", "169.254.169.254",
		"100.100.1.1", "0.0.0.0", "::1", "fe80::1", "fd00::1", "::ffff:127.0.0.1",
	}
	for _, s := range public {
		if !IsPublicIP(net.ParseIP(s)) {
			t.Errorf("%s should be public", s)
		}
	}
	for _, s := range blocked {
		if IsPublicIP(net.ParseIP(s)) {
			t.Errorf("%s should be blocked", s)
		}
	}
}

func TestNewClient_BlocksLoopback(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	defer srv.Close()

	c := NewClient(Options{Timeout: 2 * time.Second, MaxRedirects: 3})
	resp, err := c.Get(srv.URL)
	if err == nil {
		_ = resp.Body.Close()
	}
	if !errors.Is(err, ErrBlockedAddress) {
		t.Fatalf("want ErrBlockedAddress, got %v", err)
	}
}

func TestNewClient_RedirectLimit(t *testing.T) {
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, srv.URL+"/again", http.StatusFound)
	}))
	defer srv.Close()

	c := NewClient(Options{Timeout: 2 * time.Second, MaxRedirects: 2, AllowPrivate: true})
	resp, err := c.Get(srv.URL)
	if err == nil {
		_ = resp.Body.Close()
		t.Fatal("expected redirect loop to be cut off")
	}
}
