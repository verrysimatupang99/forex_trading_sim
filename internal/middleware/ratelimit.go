package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// RateLimiter implements a token bucket rate limiter
type RateLimiter struct {
	requests map[string]*clientBucket
	mu       sync.RWMutex
	rate     int           // requests per window
	window   time.Duration // time window
}

type clientBucket struct {
	tokens    int
	lastReset time.Time
}

// NewRateLimiter creates a new rate limiter
// rate: number of requests allowed
// window: time duration for the rate limit
func NewRateLimiter(rate int, window time.Duration) *RateLimiter {
	rl := &RateLimiter{
		requests: make(map[string]*clientBucket),
		rate:     rate,
		window:   window,
	}

	// Cleanup old entries periodically
	go rl.cleanup()

	return rl
}

// RateLimit returns a Gin middleware for rate limiting
func (rl *RateLimiter) RateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get client identifier (IP address or user ID if authenticated)
		clientID := c.ClientIP()
		if userID, exists := c.Get("userID"); exists {
			clientID = "user:" + string(rune(userID.(uint)))
		}

		if !rl.allow(clientID) {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":       "rate limit exceeded",
				"retry_after": rl.window.Seconds(),
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// allow checks if the client can make a request
func (rl *RateLimiter) allow(clientID string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	bucket, exists := rl.requests[clientID]

	if !exists {
		rl.requests[clientID] = &clientBucket{
			tokens:    rl.rate - 1,
			lastReset: now,
		}
		return true
	}

	// Check if window has passed
	if now.Sub(bucket.lastReset) > rl.window {
		bucket.tokens = rl.rate - 1
		bucket.lastReset = now
		return true
	}

	// Check if tokens available
	if bucket.tokens > 0 {
		bucket.tokens--
		return true
	}

	return false
}

// cleanup removes old entries periodically
func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for clientID, bucket := range rl.requests {
			if now.Sub(bucket.lastReset) > rl.window*2 {
				delete(rl.requests, clientID)
			}
		}
		rl.mu.Unlock()
	}
}

// Global rate limiter instances

// DefaultLimiter limits requests by IP
var DefaultLimiter = NewRateLimiter(100, time.Minute)

// AuthLimiter stricter rate limiting for auth endpoints
var AuthLimiter = NewRateLimiter(5, time.Minute)

// RateLimitMiddleware returns the default rate limit middleware
func RateLimitMiddleware() gin.HandlerFunc {
	return DefaultLimiter.RateLimit()
}

// AuthRateLimitMiddleware returns stricter rate limit for auth endpoints
func AuthRateLimitMiddleware() gin.HandlerFunc {
	return AuthLimiter.RateLimit()
}

// IPRateLimiter creates an IP-based rate limiter with custom settings
func IPRateLimiter(requests int, window time.Duration) gin.HandlerFunc {
	limiter := NewRateLimiter(requests, window)
	return limiter.RateLimit()
}
