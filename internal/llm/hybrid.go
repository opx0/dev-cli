package llm

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"dev-cli/internal/config"
)

var webKeywords = []string{
	"install",
	"latest",
	"version",
	"how to",
	"compare",
	"why",
	"best",
	"setup",
	"configure",
	"deploy",
	"update",
	"upgrade",
}

// cacheEntry holds a cached LLM response with expiration
type cacheEntry struct {
	result    *ResearchResult
	timestamp time.Time
}

// ResponseCache provides LRU caching for LLM responses
type ResponseCache struct {
	mu      sync.RWMutex
	entries map[string]cacheEntry
	keys    []string // For LRU ordering
	maxSize int
	ttl     time.Duration
}

// NewResponseCache creates a new cache with the given size and TTL
func NewResponseCache(maxSize int, ttl time.Duration) *ResponseCache {
	return &ResponseCache{
		entries: make(map[string]cacheEntry),
		keys:    make([]string, 0),
		maxSize: maxSize,
		ttl:     ttl,
	}
}

// hashQuery creates a cache key from a query
func hashQuery(query string) string {
	h := sha256.New()
	h.Write([]byte(strings.ToLower(strings.TrimSpace(query))))
	return hex.EncodeToString(h.Sum(nil))[:16]
}

// Get retrieves a cached result if it exists and hasn't expired
func (c *ResponseCache) Get(query string) (*ResearchResult, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	key := hashQuery(query)
	entry, ok := c.entries[key]
	if !ok {
		return nil, false
	}

	// Check TTL
	if time.Since(entry.timestamp) > c.ttl {
		return nil, false
	}

	return entry.result, true
}

// Set stores a result in the cache
func (c *ResponseCache) Set(query string, result *ResearchResult) {
	c.mu.Lock()
	defer c.mu.Unlock()

	key := hashQuery(query)

	// If already exists, update and move to end of LRU
	if _, exists := c.entries[key]; exists {
		c.entries[key] = cacheEntry{result: result, timestamp: time.Now()}
		c.moveToEnd(key)
		return
	}

	// Evict oldest if at capacity
	if len(c.keys) >= c.maxSize {
		oldest := c.keys[0]
		delete(c.entries, oldest)
		c.keys = c.keys[1:]
	}

	c.entries[key] = cacheEntry{result: result, timestamp: time.Now()}
	c.keys = append(c.keys, key)
}

// moveToEnd moves a key to the end of the LRU list
func (c *ResponseCache) moveToEnd(key string) {
	for i, k := range c.keys {
		if k == key {
			c.keys = append(c.keys[:i], c.keys[i+1:]...)
			c.keys = append(c.keys, key)
			return
		}
	}
}

// Stats returns cache hit stats
func (c *ResponseCache) Stats() (size int, capacity int) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.entries), c.maxSize
}

// Clear empties the cache
func (c *ResponseCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries = make(map[string]cacheEntry)
	c.keys = make([]string, 0)
}

type HybridClient struct {
	perplexity *PerplexityClient
	ollama     *Client
	cache      *ResponseCache
}

// Default cache: 50 entries, 10 minute TTL
var defaultCache = NewResponseCache(50, 10*time.Minute)

func NewHybridClient() *HybridClient {
	cfg := config.Load()
	return &HybridClient{
		perplexity: NewPerplexityClient(cfg),
		ollama:     NewClient(cfg),
		cache:      defaultCache,
	}
}

func (h *HybridClient) Research(query string) (*ResearchResult, error) {
	// Check cache first
	if cached, ok := h.cache.Get(query); ok {
		return cached, nil
	}

	var result *ResearchResult
	var err error

	if h.perplexity != nil && needsWebSearch(query) {
		result, err = h.perplexity.Research(context.Background(), query)
		if err == nil {
			h.cache.Set(query, result)
			return result, nil
		}
	}

	result, err = h.ollama.Research(query)
	if err == nil {
		h.cache.Set(query, result)
	}
	return result, err
}

func (h *HybridClient) HasPerplexity() bool {
	return h.perplexity != nil
}

// CacheStats returns the current cache utilization
func (h *HybridClient) CacheStats() (size int, capacity int) {
	return h.cache.Stats()
}

// ClearCache empties the response cache
func (h *HybridClient) ClearCache() {
	h.cache.Clear()
}

func (h *HybridClient) AnalyzeLog(logLines string, aiMode string) (*LogAnalysisResult, error) {
	if os.Getenv("DEV_CLI_FORCE_LOCAL") != "" || aiMode == "local" {
		return h.ollama.AnalyzeLog(logLines)
	}

	if aiMode == "cloud" {
		if h.perplexity != nil {
			return h.perplexity.AnalyzeLog(context.Background(), logLines)
		}
		return nil, fmt.Errorf("cloud AI requested but PERPLEXITY_API_KEY is not set")
	}

	return h.ollama.AnalyzeLog(logLines)
}

func (h *HybridClient) Solve(goal string) (string, error) {
	return h.ollama.Solve(goal)
}

func needsWebSearch(query string) bool {
	if os.Getenv("DEV_CLI_FORCE_LOCAL") != "" {
		return false
	}

	lower := strings.ToLower(query)
	for _, kw := range webKeywords {
		if strings.Contains(lower, kw) {
			return true
		}
	}
	return false
}
