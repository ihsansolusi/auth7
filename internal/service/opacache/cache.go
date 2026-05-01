package opacache

import (
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"
)

type Cache struct {
	mu      sync.RWMutex
	entries map[string]*cacheEntry
	ttl     time.Duration
	logger  zerolog.Logger
}

type cacheEntry struct {
	value     interface{}
	expiresAt time.Time
}

func NewCache(ttl time.Duration, logger zerolog.Logger) *Cache {
	c := &Cache{
		entries: make(map[string]*cacheEntry),
		ttl:     ttl,
		logger:  logger,
	}

	go c.cleanupLoop()

	return c
}

func (c *Cache) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, ok := c.entries[key]
	if !ok {
		return nil, false
	}

	if time.Now().After(entry.expiresAt) {
		return nil, false
	}

	return entry.value, true
}

func (c *Cache) Set(key string, value interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries[key] = &cacheEntry{
		value:     value,
		expiresAt: time.Now().Add(c.ttl),
	}
}

func (c *Cache) Invalidate(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.entries, key)

	c.logger.Debug().
		Str("key", key).
		Msg("OPA cache invalidated")
}

func (c *Cache) InvalidateByPrefix(prefix string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	count := 0
	for key := range c.entries {
		if strings.HasPrefix(key, prefix) {
			delete(c.entries, key)
			count++
		}
	}

	c.logger.Debug().
		Str("prefix", prefix).
		Int("count", count).
		Msg("OPA cache invalidated by prefix")
}

func (c *Cache) cleanupLoop() {
	ticker := time.NewTicker(c.ttl / 2)
	defer ticker.Stop()

	for range ticker.C {
		c.cleanup()
	}
}

func (c *Cache) cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	for key, entry := range c.entries {
		if now.After(entry.expiresAt) {
			delete(c.entries, key)
		}
	}
}

func (c *Cache) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.entries)
}
