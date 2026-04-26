package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

// RetryConfig holds configuration for retry logic
type RetryConfig struct {
	MaxRetries     int
	InitialDelay   time.Duration
	MaxDelay       time.Duration
	BackoffFactor  float64
}

// DefaultRetryConfig returns default retry configuration
func DefaultRetryConfig() *RetryConfig {
	return &RetryConfig{
		MaxRetries:    3,
		InitialDelay:  100 * time.Millisecond,
		MaxDelay:      10 * time.Second,
		BackoffFactor: 2.0,
	}
}

// HTTPClient wraps http.Client with retry logic
type HTTPClient struct {
	client *http.Client
	config *RetryConfig
}

// NewHTTPClient creates a new HTTP client with retry logic
func NewHTTPClient(config *RetryConfig) *HTTPClient {
	if config == nil {
		config = DefaultRetryConfig()
	}
	return &HTTPClient{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		config: config,
	}
}

// RetryRequest executes an HTTP request with retry logic
func (c *HTTPClient) RetryRequest(ctx context.Context, method, url string, body io.Reader) (*http.Response, error) {
	var lastErr error
	delay := c.config.InitialDelay

	for attempt := 0; attempt <= c.config.MaxRetries; attempt++ {
		if attempt > 0 {
			log.Printf("Retry attempt %d/%d after %v", attempt, c.config.MaxRetries, delay)
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
			}
			delay = time.Duration(float64(delay) * c.config.BackoffFactor)
			if delay > c.config.MaxDelay {
				delay = c.config.MaxDelay
			}
		}

		req, err := http.NewRequestWithContext(ctx, method, url, body)
		if err != nil {
			lastErr = err
			continue
		}

		resp, err := c.client.Do(req)
		if err != nil {
			lastErr = err
			continue
		}

		// Retry on rate limiting (429) or server errors (5xx)
		if resp.StatusCode == 429 || (resp.StatusCode >= 500 && resp.StatusCode < 600) {
			resp.Body.Close()
			retryAfter := resp.Header.Get("Retry-After")
			if retryAfter != "" {
				if delaySec, err := time.ParseDuration(retryAfter + "s"); err == nil {
					delay = delaySec
				}
			}
			lastErr = fmt.Errorf("HTTP error: %d", resp.StatusCode)
			continue
		}

		return resp, nil
	}

	return nil, fmt.Errorf("max retries exceeded: %w", lastErr)
}

// RateLimiter implements token bucket rate limiting
type RateLimiter struct {
	tokens     float64
	maxTokens  float64
	refillRate float64
	lastRefill time.Time
	mu         float64
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(maxTokens float64, refillRate float64) *RateLimiter {
	return &RateLimiter{
		tokens:     maxTokens,
		maxTokens:  maxTokens,
		refillRate: refillRate,
		lastRefill: time.Now(),
	}
}

// Acquire attempts to acquire tokens, returns true if successful
func (r *RateLimiter) Acquire(tokens float64) bool {
	r.mu = 1
	defer func() { r.mu = 0 }()

	r.refill()

	if r.tokens >= tokens {
		r.tokens -= tokens
		return true
	}
	return false
}

// refill adds tokens based on time elapsed
func (r *RateLimiter) refill() {
	now := time.Now()
	elapsed := now.Sub(r.lastRefill).Seconds()
	newTokens := elapsed * r.refillRate
	r.tokens = min(r.maxTokens, r.tokens+newTokens)
	r.lastRefill = now
}

// APIError represents an API error with code and message
type APIError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e *APIError) Error() string {
	return fmt.Sprintf("API error [%d]: %s", e.Code, e.Message)
}

// ParseAPIError attempts to parse API error from response body
func ParseAPIError(resp *http.Response) error {
	var apiErr APIError
	if err := json.NewDecoder(resp.Body).Decode(&apiErr); err != nil {
		return fmt.Errorf("HTTP error: %d", resp.StatusCode)
	}
	return &apiErr
}

// IsRateLimited checks if error is a rate limit error
func IsRateLimited(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "429")
}

// IsServerError checks if error is a server error
func IsServerError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "5")
}
