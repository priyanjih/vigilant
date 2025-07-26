package llmcache

import (
	"fmt"
	"sync"
	"time"

	"vigilant/pkg/hashutil"
	"vigilant/pkg/summarizer"
)

type CachedSummary struct {
	Summary   map[string]summarizer.RootCauseSummary
	InputHash string
	Timestamp time.Time
	TTL       time.Duration
}

type LLMCache struct {
	cache map[string]*CachedSummary
	mu    sync.RWMutex
	defaultTTL time.Duration
}

func NewLLMCache(defaultTTL time.Duration) *LLMCache {
	return &LLMCache{
		cache:      make(map[string]*CachedSummary),
		defaultTTL: defaultTTL,
	}
}

// GetOrSummarize checks cache first, calls LLM only if needed
func (c *LLMCache) GetOrSummarize(correlations []summarizer.AlertCorrelation) (map[string]summarizer.RootCauseSummary, error) {
	// Early return for empty correlations - no LLM call needed
	if len(correlations) == 0 {
		fmt.Println("[LLM CACHE] No correlations - skipping LLM call")
		return make(map[string]summarizer.RootCauseSummary), nil
	}

	// Generate hash based on correlation content
	inputHash := hashutil.HashData(correlations)
	
	// Check cache first
	c.mu.RLock()
	if cached, exists := c.cache[inputHash]; exists {
		// Check if cache entry is still valid
		if time.Since(cached.Timestamp) < cached.TTL {
			c.mu.RUnlock()
			fmt.Printf("[LLM CACHE] Cache hit for hash %s - skipping LLM call\n", 
				hashutil.SafeHashDisplay(inputHash))
			return cached.Summary, nil
		}
		// Cache expired, will need to refresh
		fmt.Printf("[LLM CACHE] Cache expired for hash %s\n", 
			hashutil.SafeHashDisplay(inputHash))
	}
	c.mu.RUnlock()
	
	// Cache miss or expired - call LLM
	fmt.Printf("[LLM CACHE] Cache miss for hash %s - calling LLM\n", 
		hashutil.SafeHashDisplay(inputHash))
	
	summary, err := summarizer.SummarizeMany(correlations)
	if err != nil {
		return nil, fmt.Errorf("LLM call failed: %w", err)
	}
	
	// Store successful result in cache
	c.mu.Lock()
	c.cache[inputHash] = &CachedSummary{
		Summary:   summary,
		InputHash: inputHash,
		Timestamp: time.Now(),
		TTL:       c.defaultTTL,
	}
	c.mu.Unlock()
	
	fmt.Printf("[LLM CACHE] Cached new result for hash %s\n", 
		hashutil.SafeHashDisplay(inputHash))
	
	return summary, nil
}

// CleanupExpired removes expired cache entries
func (c *LLMCache) CleanupExpired() {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	now := time.Now()
	expired := 0
	
	for hash, cached := range c.cache {
		if now.Sub(cached.Timestamp) > cached.TTL {
			delete(c.cache, hash)
			expired++
		}
	}
	
	if expired > 0 {
		fmt.Printf("[LLM CACHE] Cleaned up %d expired entries\n", expired)
	}
}

// GetStats returns cache statistics
func (c *LLMCache) GetStats() (entries int, oldestAge time.Duration) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	entries = len(c.cache)
	oldest := time.Now()
	
	for _, cached := range c.cache {
		if cached.Timestamp.Before(oldest) {
			oldest = cached.Timestamp
		}
	}
	
	if entries > 0 {
		oldestAge = time.Since(oldest)
	}
	
	return entries, oldestAge
}

// Clear removes all cache entries (useful for testing)
func (c *LLMCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cache = make(map[string]*CachedSummary)
	fmt.Println("[LLM CACHE] Cache cleared")
}