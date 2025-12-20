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

type Step struct {
	Type    string `json:"type"`
	Content string `json:"content"`
	File    string `json:"file,omitempty"`
	Note    string `json:"note,omitempty"`
}

type Solution struct {
	ID          int    `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Steps       []Step `json:"steps"`
	Source      string `json:"source,omitempty"`
}

type ResearchResult struct {
	Query     string     `json:"query"`
	Solutions []Solution `json:"solutions"`
}

type LogAnalysisResult struct {
	Explanation string `json:"explanation"`
	Fix         string `json:"fix"`
}

type cacheEntry struct {
	result    *ResearchResult
	timestamp time.Time
}

type ResponseCache struct {
	mu      sync.RWMutex
	entries map[string]cacheEntry
	keys    []string // For LRU ordering
	maxSize int
	ttl     time.Duration
}

func NewResponseCache(maxSize int, ttl time.Duration) *ResponseCache {
	return &ResponseCache{
		entries: make(map[string]cacheEntry),
		keys:    make([]string, 0),
		maxSize: maxSize,
		ttl:     ttl,
	}
}

func hashQuery(query string) string {
	h := sha256.New()
	h.Write([]byte(strings.ToLower(strings.TrimSpace(query))))
	return hex.EncodeToString(h.Sum(nil))[:16]
}

func (c *ResponseCache) Get(query string) (*ResearchResult, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	key := hashQuery(query)
	entry, ok := c.entries[key]
	if !ok {
		return nil, false
	}

	if time.Since(entry.timestamp) > c.ttl {
		return nil, false
	}

	return entry.result, true
}

func (c *ResponseCache) Set(query string, result *ResearchResult) {
	c.mu.Lock()
	defer c.mu.Unlock()

	key := hashQuery(query)

	if _, exists := c.entries[key]; exists {
		c.entries[key] = cacheEntry{result: result, timestamp: time.Now()}
		c.moveToEnd(key)
		return
	}

	if len(c.keys) >= c.maxSize {
		oldest := c.keys[0]
		delete(c.entries, oldest)
		c.keys = c.keys[1:]
	}

	c.entries[key] = cacheEntry{result: result, timestamp: time.Now()}
	c.keys = append(c.keys, key)
}

func (c *ResponseCache) moveToEnd(key string) {
	for i, k := range c.keys {
		if k == key {
			c.keys = append(c.keys[:i], c.keys[i+1:]...)
			c.keys = append(c.keys, key)
			return
		}
	}
}

func (c *ResponseCache) Stats() (size int, capacity int) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.entries), c.maxSize
}

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

func (h *HybridClient) CacheStats() (size int, capacity int) {
	return h.cache.Stats()
}

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
