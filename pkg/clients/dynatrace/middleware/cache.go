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

// cacheEntry holds all relevant info for a cached response.
// The only not obvious part is that the `body` has to be stored separate from the actual `response`.
// The reason for this is that the `Body` part of the `http.Response` is an `io.ReadCloser`.
// This means the you may only read the body once, which is not good if we would like to reuse the `http.Response`
// To combat this we read the `Body` while creating the `cacheEntry` and store it inside this new entry.
// We always "recreate"(put it into a new Reader) the `Body` in the affected `http.Response` or when we returned the cached `http.Response`.
type cacheEntry struct {
	response     *http.Response
	body         []byte
	creationTime time.Time
	ttl          time.Duration
}

func (ce *cacheEntry) isOutdated() bool {
	return time.Since(ce.creationTime) > ce.ttl
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

func (rc *responseCache) set(key string, response *http.Response, ttl time.Duration) {
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
		response:     response,
		body:         body,
		creationTime: time.Now(),
		ttl:          ttl,
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

// RunPeriodicCacheCleanup creates a small goroutine that will remove all expired entries from the dtclient cache.
// This is necessary because if the user changes tokens, disables features etc., some entries will never be hit again,
// so we can't clean them up during normal reconciles.
// This only makes sense to use in the Operator Pod.
// - The bootstrap command only runs once, and can only download a ~700MB archive. (can be less or more, depending on the technologies present in the archive)
// - The csidriver (mainly `provisioner` container) does not make "in-memory cacheable" requests, it only downloads +700MB archives, that we "cache" on the node.
// - The webhook (and any other not listed command) does not use the dtclient.
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
