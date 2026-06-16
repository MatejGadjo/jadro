package rss

import (
	"sync"
	"time"
)

type rateLimiter struct {
	interval time.Duration
	lastCall time.Time
	mu       sync.Mutex
}

func newRateLimiter(requestsPerMin int) *rateLimiter {
	interval := time.Minute / time.Duration(requestsPerMin)
	if interval < time.Second {
		interval = time.Second
	}
	return &rateLimiter{interval: interval}
}

func (rl *rateLimiter) wait() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	if !rl.lastCall.IsZero() {
		if elapsed := now.Sub(rl.lastCall); elapsed < rl.interval {
			time.Sleep(rl.interval - elapsed)
		}
	}
	rl.lastCall = time.Now()
}
