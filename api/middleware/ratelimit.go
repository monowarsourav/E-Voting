package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// RateLimiter implements token bucket rate limiting
type RateLimiter struct {
	visitors map[string]*Visitor
	mu       sync.RWMutex
	rate     int
	burst    int
}

// Visitor represents a visitor's rate limit state
type Visitor struct {
	tokens     int
	lastSeen   time.Time
	lastRefill time.Time
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(rate, burst int) *RateLimiter {
	rl := &RateLimiter{
		visitors: make(map[string]*Visitor),
		rate:     rate,
		burst:    burst,
	}

	// Cleanup goroutine
	go rl.cleanupVisitors()

	return rl
}

// Allow checks if request is allowed
func (rl *RateLimiter) Allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	visitor, exists := rl.visitors[ip]
	now := time.Now()

	if !exists {
		rl.visitors[ip] = &Visitor{
			tokens:     rl.burst - 1,
			lastSeen:   now,
			lastRefill: now,
		}
		return true
	}

	// Refill tokens based on time elapsed
	elapsed := now.Sub(visitor.lastRefill)
	tokensToAdd := int(elapsed.Seconds()) * rl.rate
	if tokensToAdd > 0 {
		visitor.tokens += tokensToAdd
		if visitor.tokens > rl.burst {
			visitor.tokens = rl.burst
		}
		visitor.lastRefill = now
	}

	visitor.lastSeen = now

	if visitor.tokens > 0 {
		visitor.tokens--
		return true
	}

	return false
}

// cleanupVisitors removes inactive visitors
func (rl *RateLimiter) cleanupVisitors() {
	for {
		time.Sleep(5 * time.Minute)

		rl.mu.Lock()
		now := time.Now()
		for ip, visitor := range rl.visitors {
			if now.Sub(visitor.lastSeen) > 10*time.Minute {
				delete(rl.visitors, ip)
			}
		}
		rl.mu.Unlock()
	}
}

// RateLimitMiddleware creates rate limiting middleware
func RateLimitMiddleware(rate, burst int) gin.HandlerFunc {
	limiter := NewRateLimiter(rate, burst)

	return func(c *gin.Context) {
		ip := c.ClientIP()

		if !limiter.Allow(ip) {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":   "rate_limit_exceeded",
				"message": "too many requests, please try again later",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// StrictRateLimitMiddleware creates stricter rate limiting for sensitive operations
func StrictRateLimitMiddleware() gin.HandlerFunc {
	// 10 requests per minute, burst of 20
	return RateLimitMiddleware(10, 20)
}

// StandardRateLimitMiddleware creates standard rate limiting
func StandardRateLimitMiddleware() gin.HandlerFunc {
	// 100 requests per minute, burst of 200
	return RateLimitMiddleware(100, 200)
}
