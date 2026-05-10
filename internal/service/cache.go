package service

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"emptyclassroom/internal/model"
)

type summaryCache struct {
	mu      sync.RWMutex
	ttl     time.Duration
	entries map[string]summaryCacheEntry
}

type summaryCacheEntry struct {
	expires time.Time
	value   model.ClassroomSummary
}

func newSummaryCache(ttl time.Duration) *summaryCache {
	c := &summaryCache{
		ttl:     ttl,
		entries: make(map[string]summaryCacheEntry),
	}
	go c.evictLoop()
	return c
}

func (c *summaryCache) Get(campusID int, date time.Time, slots []int) (model.ClassroomSummary, bool) {
	c.mu.RLock()
	entry, ok := c.entries[cacheKey(campusID, date, slots)]
	c.mu.RUnlock()
	if !ok || time.Now().After(entry.expires) {
		return model.ClassroomSummary{}, false
	}
	return entry.value, true
}

func (c *summaryCache) Set(campusID int, date time.Time, slots []int, value model.ClassroomSummary) {
	c.mu.Lock()
	c.entries[cacheKey(campusID, date, slots)] = summaryCacheEntry{
		expires: time.Now().Add(c.ttl),
		value:   value,
	}
	c.mu.Unlock()
}

func (c *summaryCache) Delete(campusID int, date time.Time) {
	c.mu.Lock()
	prefix := fmt.Sprintf("%d:%s:", campusID, date.Format("2006-01-02"))
	for key := range c.entries {
		if strings.HasPrefix(key, prefix) {
			delete(c.entries, key)
		}
	}
	c.mu.Unlock()
}

func (c *summaryCache) evictLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		c.mu.Lock()
		now := time.Now()
		for key, entry := range c.entries {
			if now.After(entry.expires) {
				delete(c.entries, key)
			}
		}
		c.mu.Unlock()
	}
}

func cacheKey(campusID int, date time.Time, slots []int) string {
	sorted := make([]int, len(slots))
	copy(sorted, slots)
	sort.Ints(sorted)
	parts := make([]string, len(sorted))
	for i, s := range sorted {
		parts[i] = strconv.Itoa(s)
	}
	return fmt.Sprintf("%d:%s:%s", campusID, date.Format("2006-01-02"), strings.Join(parts, ","))
}
