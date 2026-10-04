package service

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/MatejGadjo/jadro/internal/safehttp"
)

const (
	imageProbeTimeout      = 5 * time.Second
	imageProbeMaxRedirects = 3
	imageProbeUserAgent    = "JadroImageCheck/1.0 (+https://github.com/MatejGadjo/jadro)"
)

// ImageProber reports whether a remote image is definitively broken. It is the
// server-side check behind the public "clear image" endpoint: clients only
// *report* a broken image, the server decides.
type ImageProber interface {
	IsBroken(ctx context.Context, rawURL string) bool
}

type httpImageProber struct {
	client *http.Client
}

// NewHTTPImageProber returns a prober that refuses to connect to loopback,
// private, link-local or otherwise non-public addresses, so a stored image URL
// can never be used to reach internal services.
func NewHTTPImageProber() ImageProber {
	return newHTTPImageProber(false)
}

func newHTTPImageProber(allowPrivate bool) *httpImageProber {
	return &httpImageProber{client: safehttp.NewClient(safehttp.Options{
		Timeout:      imageProbeTimeout,
		MaxRedirects: imageProbeMaxRedirects,
		AllowPrivate: allowPrivate,
	})}
}

// IsBroken is deliberately conservative: only answers that a browser would
// also fail on count as broken (404/410, a non-image body, a domain that no
// longer resolves). Timeouts, 403s from bot protection, 5xx and our own
// network trouble all count as "not broken" so a valid image is never wiped.
func (p *httpImageProber) IsBroken(ctx context.Context, rawURL string) bool {
	u, err := url.Parse(rawURL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return true
	}

	resp, err := p.do(ctx, http.MethodHead, u.String())
	if err == nil && (resp.StatusCode == http.StatusMethodNotAllowed || resp.StatusCode == http.StatusNotImplemented) {
		_ = resp.Body.Close()
		resp, err = p.do(ctx, http.MethodGet, u.String())
	}
	if err != nil {
		var dnsErr *net.DNSError
		return errors.As(err, &dnsErr) && dnsErr.IsNotFound
	}
	defer resp.Body.Close()

	switch {
	case resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusGone:
		return true
	case resp.StatusCode >= 200 && resp.StatusCode < 300:
		ct := resp.Header.Get("Content-Type")
		return ct != "" && !strings.HasPrefix(strings.ToLower(ct), "image/")
	default:
		return false
	}
}

func (p *httpImageProber) do(ctx context.Context, method, target string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, target, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", imageProbeUserAgent)
	req.Header.Set("Accept", "image/*")
	if method == http.MethodGet {
		req.Header.Set("Range", "bytes=0-0")
	}
	return p.client.Do(req)
}
