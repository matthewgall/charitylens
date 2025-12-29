package api

import (
	"context"
	"sync"
	"time"
)

// RateLimiter implements a token bucket rate limiter for API calls with context support.
type RateLimiter struct {
	tokens         int
	maxTokens      int
	refillInterval time.Duration
	lastRefill     time.Time
	mu             sync.Mutex
	requestHistory []time.Time
}

// NewRateLimiter creates a new rate limiter with the specified requests per second.
func NewRateLimiter(requestsPerSecond int) *RateLimiter {
	return &RateLimiter{
		tokens:         requestsPerSecond,
		maxTokens:      requestsPerSecond,
		refillInterval: time.Second / time.Duration(requestsPerSecond),
		lastRefill:     time.Now(),
		requestHistory: make([]time.Time, 0, 100),
	}
}

// Wait blocks until a token is available, respecting context cancellation.
// Returns an error if the context is cancelled.
func (rl *RateLimiter) Wait(ctx context.Context) error {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	// Refill tokens based on time elapsed
	now := time.Now()
	elapsed := now.Sub(rl.lastRefill)
	tokensToAdd := int(elapsed / rl.refillInterval)

	if tokensToAdd > 0 {
		rl.tokens = min(rl.maxTokens, rl.tokens+tokensToAdd)
		rl.lastRefill = now
	}

	// Wait until token is available
	for rl.tokens <= 0 {
		rl.mu.Unlock()
		select {
		case <-ctx.Done():
			rl.mu.Lock() // Re-lock before returning so defer can unlock
			return ctx.Err()
		case <-time.After(rl.refillInterval):
		}
		rl.mu.Lock()

		now = time.Now()
		elapsed = now.Sub(rl.lastRefill)
		tokensToAdd = int(elapsed / rl.refillInterval)
		if tokensToAdd > 0 {
			rl.tokens = min(rl.maxTokens, rl.tokens+tokensToAdd)
			rl.lastRefill = now
		}
	}

	// Consume token
	rl.tokens--
	rl.requestHistory = append(rl.requestHistory, now)

	// Keep only last 100 requests
	if len(rl.requestHistory) > 100 {
		rl.requestHistory = rl.requestHistory[len(rl.requestHistory)-100:]
	}

	return nil
}

// GetStats returns the number of requests in the last minute and last second.
func (rl *RateLimiter) GetStats() (requestsLastMinute int, requestsLastSecond int) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	oneMinuteAgo := now.Add(-time.Minute)
	oneSecondAgo := now.Add(-time.Second)

	for _, t := range rl.requestHistory {
		if t.After(oneMinuteAgo) {
			requestsLastMinute++
		}
		if t.After(oneSecondAgo) {
			requestsLastSecond++
		}
	}

	return
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
