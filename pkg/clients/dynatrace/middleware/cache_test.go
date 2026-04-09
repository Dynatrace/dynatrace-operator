package middleware

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/url"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// freshCache returns a new responseCache and replaces the package-level one,
// preventing state from leaking between tests.
func freshCache(t *testing.T) *responseCache {
	t.Helper()
	c := newCache()
	cache = c
	t.Cleanup(func() { cache = newCache() })

	return c
}

func fakeResponse(body string) *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(bytes.NewBufferString(body)),
		Request:    &http.Request{URL: &url.URL{Host: "example.com"}},
	}
}

func TestResponseCache_SetAndGet(t *testing.T) {
	t.Run("miss on empty cache", func(t *testing.T) {
		c := freshCache(t)
		assert.Nil(t, c.get("anykey"))
	})

	t.Run("hit after store within TTL", func(t *testing.T) {
		c := freshCache(t)
		c.set("key", fakeResponse("hello"), time.Minute)

		resp := c.get("key")
		require.NotNil(t, resp)
		body, _ := io.ReadAll(resp.Body)
		assert.Equal(t, "hello", string(body))
	})
	t.Run("expired entry is evicted on get and is a miss", func(t *testing.T) {
		c := freshCache(t)

		synctest.Test(t, func(t *testing.T) {
			c.set("key", fakeResponse("data"), time.Minute)
			time.Sleep(5 * time.Minute)

			entry := c.get("key") // triggers eviction
			assert.Nil(t, entry, "expired entry must be treated as a miss")
		})

		c.mu.Lock()
		_, exists := c.entries["key"]
		c.mu.Unlock()

		assert.False(t, exists, "expired entry must be deleted from the map")
	})

	t.Run("nil response is not stored", func(t *testing.T) {
		c := freshCache(t)
		c.set("key", nil, time.Minute)
		assert.Nil(t, c.get("key"))
	})

	t.Run("body can be read multiple times from cached response", func(t *testing.T) {
		c := freshCache(t)
		c.set("key", fakeResponse("body-content"), time.Minute)

		resp1 := c.get("key")
		require.NotNil(t, resp1)
		body1, _ := io.ReadAll(resp1.Body)

		resp2 := c.get("key")
		require.NotNil(t, resp2)
		body2, _ := io.ReadAll(resp2.Body)

		assert.Equal(t, "body-content", string(body1))
		assert.Equal(t, "body-content", string(body2))
	})

	t.Run("caller's original response body is restored after store", func(t *testing.T) {
		c := freshCache(t)
		resp := fakeResponse("original")
		c.set("key", resp, time.Minute)

		body, _ := io.ReadAll(resp.Body)
		assert.Equal(t, "original", string(body))
	})
}

func TestResponseCache_Cleanup(t *testing.T) {
	t.Run("removes expired entries, keeps fresh ones", func(t *testing.T) {
		c := freshCache(t)

		synctest.Test(t, func(t *testing.T) {
			c.set("expired", fakeResponse("old"), time.Minute)
			c.set("fresh", fakeResponse("new"), 3*time.Minute)
			time.Sleep(2 * time.Minute)

			c.cleanup()
		})

		c.mu.Lock()
		_, expiredExists := c.entries["expired"]
		_, freshExists := c.entries["fresh"]
		c.mu.Unlock()

		assert.False(t, expiredExists)
		assert.True(t, freshExists)
	})

	t.Run("does nothing on empty cache", func(t *testing.T) {
		c := freshCache(t)
		c.cleanup() // must not panic
		assert.Empty(t, c.entries)
	})
}

func TestRunPeriodicCacheCleanup(t *testing.T) {
	t.Run("evicts expired entries after a tick", func(t *testing.T) {
		c := freshCache(t)

		synctest.Test(t, func(t *testing.T) {
			c.set("key", fakeResponse("data"), time.Nanosecond)

			RunPeriodicCacheCleanup(t.Context(), time.Millisecond)

			time.Sleep(time.Hour)

			synctest.Wait()
		})

		c.mu.Lock()
		remaining := len(c.entries)
		c.mu.Unlock()

		assert.Equal(t, 0, remaining)
	})

	t.Run("goroutine exits when context is canceled", func(t *testing.T) {
		freshCache(t)
		ctx, cancel := context.WithCancel(context.Background())
		RunPeriodicCacheCleanup(ctx, time.Hour)
		cancel() // must not block or panic
	})
}
