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

// ── Response Cache ───────────────────────────────────────────────────────────

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

// ── HybridClient ─────────────────────────────────────────────────────────────

// HybridClient routes requests between local (Ollama) and cloud (OpenAI-compatible,
// Perplexity) providers, with response caching and automatic web-search detection.
type HybridClient struct {
	perplexity *PerplexityProvider
	openai     *OpenAIProvider
	ollama     *OllamaProvider
	cache      *ResponseCache
	cfg        *config.Config
}

var defaultCache = NewResponseCache(50, 10*time.Minute)

func NewHybridClient() *HybridClient {
	cfg := config.Load()
	return &HybridClient{
		perplexity: NewPerplexityProvider(cfg),
		openai:     NewOpenAIProvider(cfg),
		ollama:     NewOllamaProvider(cfg),
		cache:      defaultCache,
		cfg:        cfg,
	}
}

// ChatCompletion routes a raw chat completion to the best provider:
//   - Tool-calling requests prefer OpenAI (cloud) when available and !ForceLocalLLM
//     — small local models frequently fail at function-call formatting.
//   - Requests explicitly targeting a Perplexity sonar model go to Perplexity.
//   - Everything else falls back to Ollama (local, no key required).
func (h *HybridClient) ChatCompletion(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	useCloud := h.cfg != nil && !h.cfg.ForceLocalLLM

	if useCloud && len(req.Tools) > 0 && h.openai != nil {
		// For tool calls, swap the model to the cloud provider's configured default
		// if the caller sent an Ollama-tag name (cloud APIs will reject it).
		if h.cfg.OpenAIModel != "" && (req.Model == "" || req.Model == h.cfg.OllamaModel) {
			req.Model = h.cfg.OpenAIModel
		}
		return h.openai.ChatCompletion(ctx, req)
	}

	if useCloud && strings.HasPrefix(strings.ToLower(req.Model), "sonar") && h.perplexity != nil {
		return h.perplexity.ChatCompletion(ctx, req)
	}

	return h.ollama.ChatCompletion(ctx, req)
}

// Name returns the provider identifier.
func (h *HybridClient) Name() string {
	return "hybrid"
}

// LocalProvider returns the Ollama provider for direct access.
func (h *HybridClient) LocalProvider() *OllamaProvider {
	return h.ollama
}

// CloudProvider returns the Perplexity provider (may be nil).
func (h *HybridClient) CloudProvider() *PerplexityProvider {
	return h.perplexity
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

// SelectAgentModel returns the (provider, model) pair that an agent loop
// should use given the current config and a force-local override. The cloud
// path is preferred for tool-using agents because small local models struggle
// with reliable function-call formatting.
func SelectAgentModel(cfg *config.Config, forceLocal bool) (provider, model string) {
	if cfg == nil {
		return "ollama", DefaultModel
	}
	if !forceLocal && !cfg.ForceLocalLLM && cfg.OpenAIKey != "" {
		m := cfg.OpenAIModel
		if m == "" {
			m = "gpt-4o-mini"
		}
		return "openai", m
	}
	m := cfg.OllamaModel
	if m == "" {
		m = DefaultModel
	}
	return "ollama", m
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
