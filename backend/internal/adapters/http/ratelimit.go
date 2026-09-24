package httpapi

import (
	"net"
	"net/http"
	"sync"
	"time"

	"flashcard-backend/internal/apperror"
)

// ipRateLimiter is a basic fixed-window limiter keyed by client IP,
// sufficient to blunt accidental or naive abuse of the session endpoint
// for a single-instance MVP (spec section 16). It is not a distributed
// rate limiter and resets if the process restarts; a multi-instance
// deployment would need a shared store instead.
type ipRateLimiter struct {
	mu       sync.Mutex
	limit    int
	window   time.Duration
	counters map[string]*windowCounter
}

type windowCounter struct {
	count      int
	windowEnds time.Time
}

func newIPRateLimiter(limit int, window time.Duration) *ipRateLimiter {
	return &ipRateLimiter{
		limit:    limit,
		window:   window,
		counters: make(map[string]*windowCounter),
	}
}

func (l *ipRateLimiter) allow(key string, now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	c, ok := l.counters[key]
	if !ok || now.After(c.windowEnds) {
		c = &windowCounter{count: 0, windowEnds: now.Add(l.window)}
		l.counters[key] = c
	}

	c.count++
	return c.count <= l.limit
}

func withRateLimit(limiter *ipRateLimiter, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !limiter.allow(clientIP(r), time.Now()) {
			writeError(w, r, apperror.RateLimited("rateLimit.exceeded", "too many requests, please try again later"))
			return
		}
		next.ServeHTTP(w, r)
	}
}

func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
