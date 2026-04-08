package middleware

import (
	"bytes"
	"context"
	"io"
	"maps"
	"net/http"
	"sync"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
)

var (
	log = logd.Get().WithName("dynatraceapi-cache")

	cache = newCache()
)

type cacheEntry struct {
	response *http.Response
	body     []byte
	lastCall time.Time
	ttl      time.Duration
}

func (ce *cacheEntry) isOutdated() bool {
	return time.Since(ce.lastCall) > ce.ttl
}

type responseCache struct {
	entries map[string]*cacheEntry
	mu      *sync.Mutex
}

func (rc *responseCache) get(key string) *http.Response {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	entry, ok := rc.entries[key]
	if !ok {
		return nil
	}

	if entry.isOutdated() {
		log.Debug("outdated entry found, removing", "url", entry.response.Request.URL)

		delete(rc.entries, key)

		return nil
	}

	// Reconstruct a fresh body reader so every caller gets a full, open body.
	resp := *entry.response
	resp.Body = io.NopCloser(bytes.NewReader(entry.body))

	return &resp
}

func (rc *responseCache) store(key string, response *http.Response, ttl time.Duration) {
	if response == nil {
		return
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return
	}

	response.Body.Close()
	// Restore body for the current caller so they can still read it.
	response.Body = io.NopCloser(bytes.NewReader(body))

	rc.mu.Lock()
	defer rc.mu.Unlock()

	rc.entries[key] = &cacheEntry{
		response: response,
		body:     body,
		lastCall: time.Now(),
		ttl:      ttl,
	}
}

func (rc *responseCache) cleanup() {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	maps.DeleteFunc(rc.entries, func(key string, value *cacheEntry) bool {
		isOutdated := value.isOutdated()
		if isOutdated {
			log.Debug("outdated entry found, removing", "url", value.response.Request.URL)
		}

		return isOutdated
	})
}

func newCache() *responseCache {
	return &responseCache{
		entries: make(map[string]*cacheEntry),
		mu:      &sync.Mutex{},
	}
}

func RunPeriodicCacheCleanup(ctx context.Context, period time.Duration) {
	go func() {
		ticker := time.NewTicker(period)
		defer ticker.Stop()

		log.Debug("periodic cache cleanup routine setup", "period", period)

		for {
			select {
			case <-ticker.C:
				log.Debug("periodic cache cleanup started")
				cache.cleanup()
			case <-ctx.Done():
				return
			}
		}
	}()
}
