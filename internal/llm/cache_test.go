package llm

import (
	"testing"
	"time"
)

func TestErrorCache_BasicOperations(t *testing.T) {
	cache := NewErrorCache(10)

	if cache.Size() != 0 {
		t.Errorf("expected size 0, got %d", cache.Size())
	}

	analysis := &CachedAnalysis{
		RootCauseNodes:   []string{"missing dependency"},
		RemediationSteps: []string{"npm install"},
		Explanation:      "Package not found",
		Fix:              "npm install express",
		Confidence:       0.9,
	}
	cache.Put("sig-001", analysis)

	if cache.Size() != 1 {
		t.Errorf("expected size 1, got %d", cache.Size())
	}

	retrieved := cache.Get("sig-001")
	if retrieved == nil {
		t.Fatal("expected to find cached entry")
	}
	if retrieved.Fix != "npm install express" {
		t.Errorf("expected fix 'npm install express', got '%s'", retrieved.Fix)
	}
	if retrieved.HitCount != 2 {
		t.Errorf("expected hit count 2, got %d", retrieved.HitCount)
	}

	miss := cache.Get("sig-nonexistent")
	if miss != nil {
		t.Error("expected nil for cache miss")
	}

	if !cache.Contains("sig-001") {
		t.Error("contains should return true for existing key")
	}
	if cache.Contains("sig-nonexistent") {
		t.Error("contains should return false for non-existing key")
	}
}

func TestErrorCache_LRUEviction(t *testing.T) {
	cache := NewErrorCache(3)

	cache.Put("sig-1", &CachedAnalysis{Explanation: "first"})
	cache.Put("sig-2", &CachedAnalysis{Explanation: "second"})
	cache.Put("sig-3", &CachedAnalysis{Explanation: "third"})

	if cache.Size() != 3 {
		t.Errorf("expected size 3, got %d", cache.Size())
	}

	cache.Get("sig-1")

	cache.Put("sig-4", &CachedAnalysis{Explanation: "fourth"})

	if cache.Size() != 3 {
		t.Errorf("expected size still 3 after eviction, got %d", cache.Size())
	}

	if !cache.Contains("sig-1") {
		t.Error("sig-1 should not be evicted (recently used)")
	}

	if !cache.Contains("sig-4") {
		t.Error("sig-4 should exist")
	}
}

func TestErrorCache_Stats(t *testing.T) {
	cache := NewErrorCache(10)

	stats := cache.Stats()
	if stats.Hits != 0 || stats.Misses != 0 {
		t.Error("initial stats should be zero")
	}

	cache.Put("sig-1", &CachedAnalysis{Explanation: "test"})
	cache.Get("sig-1")
	cache.Get("sig-1")
	cache.Get("sig-2")

	stats = cache.Stats()
	if stats.Hits != 2 {
		t.Errorf("expected 2 hits, got %d", stats.Hits)
	}
	if stats.Misses != 1 {
		t.Errorf("expected 1 miss, got %d", stats.Misses)
	}
	if stats.HitRate < 0.66 || stats.HitRate > 0.67 {
		t.Errorf("expected hit rate ~0.67, got %f", stats.HitRate)
	}
}

func TestErrorCache_Clear(t *testing.T) {
	cache := NewErrorCache(10)

	cache.Put("sig-1", &CachedAnalysis{Explanation: "test1"})
	cache.Put("sig-2", &CachedAnalysis{Explanation: "test2"})
	cache.Get("sig-1")

	cache.Clear()

	if cache.Size() != 0 {
		t.Errorf("expected size 0 after clear, got %d", cache.Size())
	}
	stats := cache.Stats()
	if stats.Hits != 0 || stats.Misses != 0 {
		t.Error("stats should be reset after clear")
	}
}

func TestErrorCache_Delete(t *testing.T) {
	cache := NewErrorCache(10)

	cache.Put("sig-1", &CachedAnalysis{Explanation: "test"})
	cache.Delete("sig-1")

	if cache.Contains("sig-1") {
		t.Error("sig-1 should be deleted")
	}
	if cache.Size() != 0 {
		t.Errorf("expected size 0, got %d", cache.Size())
	}
}

func TestErrorCache_GetTopHits(t *testing.T) {
	cache := NewErrorCache(10)

	cache.Put("sig-1", &CachedAnalysis{Explanation: "low hits"})
	cache.Put("sig-2", &CachedAnalysis{Explanation: "high hits"})
	cache.Put("sig-3", &CachedAnalysis{Explanation: "medium hits"})

	cache.Get("sig-2")
	cache.Get("sig-2")
	cache.Get("sig-2")

	cache.Get("sig-3")

	topHits := cache.GetTopHits(2)
	if len(topHits) != 2 {
		t.Fatalf("expected 2 top hits, got %d", len(topHits))
	}
	if topHits[0].Signature != "sig-2" {
		t.Errorf("expected sig-2 to be top hit, got %s", topHits[0].Signature)
	}
}

func TestErrorCache_Timestamps(t *testing.T) {
	cache := NewErrorCache(10)

	before := time.Now()
	cache.Put("sig-1", &CachedAnalysis{Explanation: "test"})
	after := time.Now()

	entry := cache.Get("sig-1")
	if entry.CreatedAt.Before(before) || entry.CreatedAt.After(after) {
		t.Error("CreatedAt should be within test bounds")
	}
	if entry.LastHit.Before(entry.CreatedAt) {
		t.Error("LastHit should be at or after CreatedAt")
	}
}
