package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

const (
	baseURL           = "https://api.charitycommission.gov.uk/register/api"
	defaultTimeout    = 30 * time.Second
	defaultMaxRetries = 3
)

// Client is a client for the Charity Commission API with multi-key support.
type Client struct {
	apiKeys     []string
	keyIndex    uint64 // atomic counter for round-robin
	userAgent   string
	httpClient  *http.Client
	rateLimiter *RateLimiter
	maxRetries  int
	verbose     bool
	keyStats    map[string]*KeyStats
	mu          sync.RWMutex
}

// KeyStats tracks statistics for each API key.
type KeyStats struct {
	TotalRequests  uint64
	FailedRequests uint64
	LastUsed       time.Time
	mu             sync.Mutex
}

// ClientConfig holds configuration for the API client.
type ClientConfig struct {
	APIKeys     []string // Multiple API keys for load balancing
	APIKey      string   // Single API key (for backwards compatibility)
	UserAgent   string
	RateLimiter *RateLimiter
	MaxRetries  int
	Timeout     time.Duration
	Verbose     bool
}

// NewClient creates a new Charity Commission API client.
// Supports multiple API keys for load balancing.
func NewClient(config ClientConfig) *Client {
	if config.Timeout == 0 {
		config.Timeout = defaultTimeout
	}
	if config.MaxRetries == 0 {
		config.MaxRetries = defaultMaxRetries
	}
	if config.UserAgent == "" {
		config.UserAgent = "CharityLens/1.0"
	}

	// Support both single key and multiple keys
	apiKeys := config.APIKeys
	if len(apiKeys) == 0 && config.APIKey != "" {
		apiKeys = []string{config.APIKey}
	}

	// Initialize key stats
	keyStats := make(map[string]*KeyStats)
	for _, key := range apiKeys {
		keyStats[key] = &KeyStats{}
	}

	return &Client{
		apiKeys:     apiKeys,
		userAgent:   config.UserAgent,
		httpClient:  &http.Client{Timeout: config.Timeout},
		rateLimiter: config.RateLimiter,
		maxRetries:  config.MaxRetries,
		verbose:     config.Verbose,
		keyStats:    keyStats,
	}
}

// getNextAPIKey returns the next API key using round-robin.
func (c *Client) getNextAPIKey() string {
	if len(c.apiKeys) == 1 {
		return c.apiKeys[0]
	}

	// Atomic round-robin
	index := atomic.AddUint64(&c.keyIndex, 1)
	key := c.apiKeys[index%uint64(len(c.apiKeys))]

	// Update stats
	c.mu.RLock()
	stats := c.keyStats[key]
	c.mu.RUnlock()

	stats.mu.Lock()
	stats.TotalRequests++
	stats.LastUsed = time.Now()
	stats.mu.Unlock()

	return key
}

// GetKeyStats returns statistics for all API keys.
func (c *Client) GetKeyStats() map[string]KeyStats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make(map[string]KeyStats)
	for key, stats := range c.keyStats {
		stats.mu.Lock()
		// Mask the key for security (show first 8 chars)
		maskedKey := key
		if len(key) > 8 {
			maskedKey = key[:8] + "..." + key[len(key)-4:]
		}
		result[maskedKey] = KeyStats{
			TotalRequests:  stats.TotalRequests,
			FailedRequests: stats.FailedRequests,
			LastUsed:       stats.LastUsed,
		}
		stats.mu.Unlock()
	}
	return result
}

// FetchCharityDetails fetches complete charity details by charity number.
func (c *Client) FetchCharityDetails(ctx context.Context, charityNum int) (map[string]any, error) {
	url := fmt.Sprintf("%s/allcharitydetailsV2/%d/0", baseURL, charityNum)

	var data map[string]any
	err := c.doRequest(ctx, url, &data)
	if err != nil {
		return nil, err
	}

	return data, nil
}

// SearchByName searches for charities by name.
func (c *Client) SearchByName(ctx context.Context, query string) ([]map[string]any, error) {
	encodedQuery := url.PathEscape(query)
	apiURL := fmt.Sprintf("%s/searchCharityName/%s", baseURL, encodedQuery)

	var results []map[string]any
	err := c.doRequest(ctx, apiURL, &results)
	if err != nil {
		return nil, err
	}

	return results, nil
}

// SearchByNumber searches for a charity by registration number.
func (c *Client) SearchByNumber(ctx context.Context, charityNum string) ([]map[string]any, error) {
	apiURL := fmt.Sprintf("%s/charityRegNumber/%s/0", baseURL, charityNum)

	var result map[string]any
	err := c.doRequest(ctx, apiURL, &result)
	if err != nil {
		return nil, err
	}

	// Wrap in array for consistency with other search functions
	return []map[string]any{result}, nil
}

// FetchFinancialHistory fetches detailed financial history for a charity.
func (c *Client) FetchFinancialHistory(ctx context.Context, charityNum int) ([]map[string]any, error) {
	apiURL := fmt.Sprintf("%s/charityfinancialhistory/%d/0", baseURL, charityNum)

	var history []map[string]any
	err := c.doRequest(ctx, apiURL, &history)
	if err != nil {
		return nil, err
	}

	return history, nil
}

// doRequest executes an HTTP request with retry logic and rate limiting.
func (c *Client) doRequest(ctx context.Context, url string, result any) error {
	var lastErr error
	var currentKey string

	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		// Get API key for this attempt (might rotate on retry)
		currentKey = c.getNextAPIKey()

		// Wait for rate limiter
		if c.rateLimiter != nil {
			if err := c.rateLimiter.Wait(ctx); err != nil {
				return err
			}
		}

		// Exponential backoff delay before retry (skip on first attempt)
		if attempt > 0 {
			backoffDuration := time.Duration(math.Pow(2, float64(attempt-1))) * time.Second
			if c.verbose {
				log.Printf("Retry %d/%d after %v (using key ...%s)",
					attempt, c.maxRetries, backoffDuration, currentKey[len(currentKey)-4:])
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoffDuration):
			}
		}

		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return err
		}

		req.Header.Set("Ocp-Apim-Subscription-Key", currentKey)
		req.Header.Set("User-Agent", c.userAgent)

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = err
			c.recordFailure(currentKey)
			continue
		}

		// Handle response
		if resp.StatusCode == 200 {
			defer resp.Body.Close()
			if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
				return fmt.Errorf("failed to decode response: %w", err)
			}
			return nil
		}

		// Handle 404 - resource not found
		if resp.StatusCode == 404 {
			resp.Body.Close()
			return fmt.Errorf("not found (404)")
		}

		// Handle 429 - rate limited (try next key if available)
		if resp.StatusCode == 429 {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()

			retryAfter := resp.Header.Get("Retry-After")
			waitTime := time.Duration(math.Pow(2, float64(attempt))) * time.Second

			if retryAfter != "" {
				if seconds, err := strconv.Atoi(retryAfter); err == nil {
					waitTime = time.Duration(seconds) * time.Second
				}
			}

			if c.verbose {
				log.Printf("Rate limited (429) on key ...%s, waiting %v before retry",
					currentKey[len(currentKey)-4:], waitTime)
			}

			c.recordFailure(currentKey)

			// If we have multiple keys, try the next one immediately
			if len(c.apiKeys) > 1 && attempt < c.maxRetries {
				if c.verbose {
					log.Printf("Rotating to next API key")
				}
				continue
			}

			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(waitTime):
			}

			lastErr = fmt.Errorf("rate limited: %s", string(body))
			continue
		}

		// Handle 5xx - server errors
		if resp.StatusCode >= 500 {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()

			waitTime := time.Duration(math.Pow(2, float64(attempt))) * time.Second
			if c.verbose {
				log.Printf("Server error (%d), waiting %v before retry", resp.StatusCode, waitTime)
			}

			c.recordFailure(currentKey)

			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(waitTime):
			}

			lastErr = fmt.Errorf("server error %d: %s", resp.StatusCode, string(body))
			continue
		}

		// Other errors (4xx except 429) - don't retry
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		c.recordFailure(currentKey)
		return fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	return fmt.Errorf("max retries exceeded: %w", lastErr)
}

// recordFailure increments the failure count for a key.
func (c *Client) recordFailure(apiKey string) {
	c.mu.RLock()
	stats := c.keyStats[apiKey]
	c.mu.RUnlock()

	if stats != nil {
		stats.mu.Lock()
		stats.FailedRequests++
		stats.mu.Unlock()
	}
}
