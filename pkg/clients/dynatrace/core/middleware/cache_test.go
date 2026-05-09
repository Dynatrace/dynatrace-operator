package middleware

import (
	"bytes"
	"context"
	"errors"
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

	return c
}

func fakeResponse(body string) *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     make(http.Header),
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
		require.NoError(t, c.set("key", fakeResponse("hello"), time.Minute))

		resp := c.get("key")
		require.NotNil(t, resp)
		body, _ := io.ReadAll(resp.Body)
		require.NoError(t, resp.Body.Close())
		assert.Equal(t, "hello", string(body))
	})
	t.Run("expired entry is evicted on get and is a miss", func(t *testing.T) {
		c := freshCache(t)

		synctest.Test(t, func(t *testing.T) {
			require.NoError(t, c.set("key", fakeResponse("data"), time.Minute))
			time.Sleep(time.Minute + time.Second)

			entry := c.get("key") // triggers eviction
			assert.Nil(t, entry, "expired entry must be treated as a miss")
		})

		assert.NotContains(t, c.entries, "key")
	})

	t.Run("nil response is not stored", func(t *testing.T) {
		c := freshCache(t)
		require.NoError(t, c.set("key", nil, time.Minute))
		assert.Nil(t, c.get("key"))
	})

	t.Run("body can be read multiple times from cached response", func(t *testing.T) {
		c := freshCache(t)
		require.NoError(t, c.set("key", fakeResponse("body-content"), time.Minute))

		resp1 := c.get("key")
		require.NotNil(t, resp1)
		body1, _ := io.ReadAll(resp1.Body)
		require.NoError(t, resp1.Body.Close())

		resp2 := c.get("key")
		require.NotNil(t, resp2)
		body2, _ := io.ReadAll(resp2.Body)
		require.NoError(t, resp2.Body.Close())

		assert.Equal(t, "body-content", string(body1))
		assert.Equal(t, "body-content", string(body2))
	})

	t.Run("caller's original response body is restored after store", func(t *testing.T) {
		c := freshCache(t)
		resp := fakeResponse("original")
		require.NoError(t, c.set("key", resp, time.Minute))

		body, _ := io.ReadAll(resp.Body)
		assert.Equal(t, "original", string(body))
	})

	t.Run("returns error and does not cache when reading body fails", func(t *testing.T) {
		c := freshCache(t)
		readErr := errors.New("read error")
		body := &ioErrReader{err: readErr}
		resp := fakeResponse("")
		resp.Body = body

		err := c.set("key", resp, time.Minute)

		require.ErrorIs(t, err, readErr)
		assert.Nil(t, c.get("key"))
		assert.True(t, body.closed, "body must be closed when reading fails")
	})
}

func TestResponseCache_Cleanup(t *testing.T) {
	t.Run("removes expired entries, keeps fresh ones", func(t *testing.T) {
		c := freshCache(t)

		synctest.Test(t, func(t *testing.T) {
			require.NoError(t, c.set("expired", fakeResponse("old"), time.Minute))
			require.NoError(t, c.set("fresh", fakeResponse("new"), 3*time.Minute))
			time.Sleep(2 * time.Minute)

			c.cleanup()
		})

		assert.NotContains(t, c.entries, "expired")
		assert.Contains(t, c.entries, "fresh")
	})

	t.Run("does nothing on empty cache", func(t *testing.T) {
		c := freshCache(t)
		assert.NotPanics(t, cache.cleanup)
		assert.Empty(t, c.entries)
	})
}

func TestRunPeriodicCacheCleanup(t *testing.T) {
	t.Run("evicts expired entries after a tick", func(t *testing.T) {
		c := freshCache(t)

		synctest.Test(t, func(t *testing.T) {
			require.NoError(t, c.set("key", fakeResponse("data"), time.Minute))

			go RunPeriodicCacheCleanup(t.Context(), 2*time.Minute)

			time.Sleep(4 * time.Minute)

			synctest.Wait()
		})

		assert.Empty(t, c.entries)
	})

	t.Run("goroutine exits when context is canceled", func(t *testing.T) {
		freshCache(t)
		ctx, cancel := context.WithCancel(t.Context())
		assert.NotPanics(t, func() {
			go RunPeriodicCacheCleanup(ctx, time.Hour)
			cancel()
		})
	})
}

// ioErrReader is a ReadCloser that always returns the configured error on Read
// and tracks whether Close was called.
type ioErrReader struct {
	err    error
	closed bool
}

func (r *ioErrReader) Read(_ []byte) (int, error) {
	return 0, r.err
}

func (r *ioErrReader) Close() error {
	r.closed = true

	return nil
}
