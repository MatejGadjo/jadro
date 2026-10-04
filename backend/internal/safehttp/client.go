// Package safehttp builds HTTP clients for fetching third-party URLs (RSS
// feeds, article images). They refuse to connect to loopback, private,
// link-local, CGNAT or otherwise non-public addresses, so neither a stored URL
// nor a redirect from a compromised site can reach services on this host or
// its internal networks.
package safehttp

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"syscall"
	"time"
)

// ErrBlockedAddress is returned when a connection targets a non-public IP.
var ErrBlockedAddress = errors.New("safehttp: blocked non-public address")

// cgnat is 100.64.0.0/10 (carrier-grade NAT, also used by Tailscale), which
// net.IP.IsPrivate does not cover.
var cgnat = &net.IPNet{IP: net.IPv4(100, 64, 0, 0), Mask: net.CIDRMask(10, 32)}

// IsPublicIP reports whether ip is a globally routable unicast address.
func IsPublicIP(ip net.IP) bool {
	return ip.IsGlobalUnicast() && !ip.IsPrivate() && !ip.IsLoopback() &&
		!ip.IsLinkLocalUnicast() && !ip.IsUnspecified() && !cgnat.Contains(ip)
}

// Options configures NewClient.
type Options struct {
	Timeout      time.Duration
	MaxRedirects int
	// AllowPrivate disables the address check. Tests only.
	AllowPrivate bool
}

// NewClient returns an http.Client whose every connection, including each
// redirect hop, is checked against IsPublicIP after DNS resolution.
func NewClient(o Options) *http.Client {
	dialer := &net.Dialer{
		Timeout: o.Timeout,
		Control: func(_, address string, _ syscall.RawConn) error {
			if o.AllowPrivate {
				return nil
			}
			host, _, err := net.SplitHostPort(address)
			if err != nil {
				return err
			}
			if ip := net.ParseIP(host); ip == nil || !IsPublicIP(ip) {
				return ErrBlockedAddress
			}
			return nil
		},
	}
	transport := &http.Transport{
		// No Proxy: an environment HTTP proxy would bypass the address check.
		DialContext:           dialer.DialContext,
		TLSHandshakeTimeout:   o.Timeout,
		ResponseHeaderTimeout: o.Timeout,
		MaxIdleConns:          20,
		IdleConnTimeout:       30 * time.Second,
	}
	return &http.Client{
		Transport: transport,
		Timeout:   o.Timeout,
		CheckRedirect: func(_ *http.Request, via []*http.Request) error {
			if len(via) >= o.MaxRedirects {
				return fmt.Errorf("safehttp: stopped after %d redirects", o.MaxRedirects)
			}
			return nil
		},
	}
}
