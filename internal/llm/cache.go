package llm

import (
	"sync"
	"time"
)

// CachedAnalysis represents a cached RCA result.
type CachedAnalysis struct {
	Signature        string
	RootCauseNodes   []string
	RemediationSteps []string
	Explanation      string
	Fix              string
	Confidence       float64
	HitCount         int
	LastHit          time.Time
	CreatedAt        time.Time
}

// ErrorCache provides fast lookup for previously analyzed errors.
// Uses LRU eviction to limit memory usage.
type ErrorCache struct {
	cache   map[string]*CachedAnalysis
	order   []string // For LRU ordering
	maxSize int
	mu      sync.RWMutex
	hits    int64
	misses  int64
}

// NewErrorCache creates a cache with configurable size.
func NewErrorCache(maxSize int) *ErrorCache {
	if maxSize <= 0 {
		maxSize = 100
	}
	return &ErrorCache{
		cache:   make(map[string]*CachedAnalysis),
		order:   make([]string, 0, maxSize),
		maxSize: maxSize,
	}
}

// Get retrieves cached analysis by error signature.
// Returns nil if not found.
func (c *ErrorCache) Get(signature string) *CachedAnalysis {
	c.mu.Lock()
	defer c.mu.Unlock()

	analysis, ok := c.cache[signature]
	if !ok {
		c.misses++
		return nil
	}

	c.hits++
	analysis.HitCount++
	analysis.LastHit = time.Now()

	c.moveToFront(signature)

	return analysis
}

// Put stores analysis result.
func (c *ErrorCache) Put(signature string, analysis *CachedAnalysis) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, exists := c.cache[signature]; exists {
		c.cache[signature] = analysis
		c.moveToFront(signature)
		return
	}

	if len(c.cache) >= c.maxSize {
		c.evictOldest()
	}

	analysis.CreatedAt = time.Now()
	analysis.LastHit = time.Now()
	analysis.HitCount = 1
	analysis.Signature = signature
	c.cache[signature] = analysis
	c.order = append([]string{signature}, c.order...)
}

// Contains checks if a signature is cached without updating LRU order.
func (c *ErrorCache) Contains(signature string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	_, ok := c.cache[signature]
	return ok
}

// Delete removes an entry from the cache.
func (c *ErrorCache) Delete(signature string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.cache, signature)
	c.removeFromOrder(signature)
}

// Clear removes all entries from the cache.
func (c *ErrorCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cache = make(map[string]*CachedAnalysis)
	c.order = make([]string, 0, c.maxSize)
	c.hits = 0
	c.misses = 0
}

// Size returns the current number of cached entries.
func (c *ErrorCache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.cache)
}

// Stats returns cache hit/miss statistics.
func (c *ErrorCache) Stats() CacheStats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	hitRate := float64(0)
	total := c.hits + c.misses
	if total > 0 {
		hitRate = float64(c.hits) / float64(total)
	}

	return CacheStats{
		Hits:    c.hits,
		Misses:  c.misses,
		Size:    len(c.cache),
		MaxSize: c.maxSize,
		HitRate: hitRate,
	}
}

// CacheStats contains cache performance metrics.
type CacheStats struct {
	Hits    int64
	Misses  int64
	Size    int
	MaxSize int
	HitRate float64
}

// GetTopHits returns the most frequently accessed cache entries.
func (c *ErrorCache) GetTopHits(limit int) []*CachedAnalysis {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entries := make([]*CachedAnalysis, 0, len(c.cache))
	for _, v := range c.cache {
		entries = append(entries, v)
	}

	for i := 0; i < len(entries)-1; i++ {
		for j := i + 1; j < len(entries); j++ {
			if entries[j].HitCount > entries[i].HitCount {
				entries[i], entries[j] = entries[j], entries[i]
			}
		}
	}

	if limit > len(entries) {
		limit = len(entries)
	}

	return entries[:limit]
}

// moveToFront moves a signature to the front of the LRU order.
func (c *ErrorCache) moveToFront(signature string) {
	c.removeFromOrder(signature)
	c.order = append([]string{signature}, c.order...)
}

// removeFromOrder removes a signature from the order slice.
func (c *ErrorCache) removeFromOrder(signature string) {
	for i, s := range c.order {
		if s == signature {
			c.order = append(c.order[:i], c.order[i+1:]...)
			break
		}
	}
}

// evictOldest removes the least recently used entry.
func (c *ErrorCache) evictOldest() {
	if len(c.order) == 0 {
		return
	}

	oldest := c.order[len(c.order)-1]
	delete(c.cache, oldest)
	c.order = c.order[:len(c.order)-1]
}
