package middleware

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// RateLimiter implements token-bucket rate limiting with a lifecycle tied to
// a context so its janitor goroutine can be shut down cleanly on server exit.
type RateLimiter struct {
	visitors map[string]*visitor
	mu       sync.Mutex
	rate     int // tokens added per second
	burst    int // bucket capacity
	done     chan struct{}
	stopOnce sync.Once
}

type visitor struct {
	tokens     int
	lastSeen   time.Time
	lastRefill time.Time
}

// NewRateLimiter creates a new rate limiter. Cancelling ctx stops the
// background cleanup goroutine.
func NewRateLimiter(ctx context.Context, rate, burst int) *RateLimiter {
	rl := &RateLimiter{
		visitors: make(map[string]*visitor),
		rate:     rate,
		burst:    burst,
		done:     make(chan struct{}),
	}
	go rl.cleanupLoop(ctx)
	return rl
}

// Stop halts the cleanup goroutine. Safe to call multiple times.
func (rl *RateLimiter) Stop() {
	rl.stopOnce.Do(func() {
		close(rl.done)
	})
}

// Allow reports whether a request from ip is permitted, consuming one token
// if so.
func (rl *RateLimiter) Allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	v, exists := rl.visitors[ip]
	if !exists {
		rl.visitors[ip] = &visitor{
			tokens:     rl.burst - 1,
			lastSeen:   now,
			lastRefill: now,
		}
		return true
	}

	elapsed := now.Sub(v.lastRefill)
	if add := int(elapsed.Seconds()) * rl.rate; add > 0 {
		v.tokens += add
		if v.tokens > rl.burst {
			v.tokens = rl.burst
		}
		v.lastRefill = now
	}

	v.lastSeen = now
	if v.tokens > 0 {
		v.tokens--
		return true
	}
	return false
}

// cleanupLoop periodically evicts stale visitors. It exits when ctx is
// cancelled or Stop is called.
func (rl *RateLimiter) cleanupLoop(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-rl.done:
			return
		case <-ticker.C:
			rl.evictStale()
		}
	}
}

func (rl *RateLimiter) evictStale() {
	cutoff := time.Now().Add(-10 * time.Minute)
	rl.mu.Lock()
	defer rl.mu.Unlock()
	for ip, v := range rl.visitors {
		if v.lastSeen.Before(cutoff) {
			delete(rl.visitors, ip)
		}
	}
}

// RateLimitMiddleware wraps an existing RateLimiter as gin middleware.
func RateLimitMiddleware(rl *RateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !rl.Allow(c.ClientIP()) {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error":   "rate_limit_exceeded",
				"message": "too many requests, please try again later",
			})
			return
		}
		c.Next()
	}
}
