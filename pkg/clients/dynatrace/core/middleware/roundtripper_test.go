package middleware

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCacheRoundTripper(t *testing.T) {
	const endpoint = "http://api.example.com/v1/resource"

	newRequest := func(t *testing.T, method, rawURL string) *http.Request {
		t.Helper()
		r, err := http.NewRequest(method, rawURL, nil)
		require.NoError(t, err)

		return r
	}

	newCachedRequest := func(t *testing.T, method, rawURL string) *http.Request {
		t.Helper()
		r := newRequest(t, method, rawURL)
		r.Header.Set(CacheRequestHeader, "true")

		return r
	}

	makeCachedRT := func(t *testing.T, ttl time.Duration, responses ...*http.Response) (http.RoundTripper, *int) {
		t.Helper()
		_ = freshCache(t) // reset global cache, cleaned up after test

		calls := 0
		idx := 0

		fake := roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			calls++
			resp := responses[idx]
			if idx < len(responses)-1 {
				idx++
			}
			resp.Request = r

			return resp, nil
		})

		return NewCacheRoundTripper(fake, ttl), &calls
	}

	t.Run("GET response is served from cache on second call", func(t *testing.T) {
		rt, calls := makeCachedRT(t, time.Minute,
			fakeResponse("hello"),
		)

		resp1, err := rt.RoundTrip(newCachedRequest(t, http.MethodGet, endpoint))
		require.NoError(t, err)
		body1, _ := io.ReadAll(resp1.Body)
		assert.Equal(t, "hello", string(body1))
		assert.Equal(t, 1, *calls)

		resp2, err := rt.RoundTrip(newCachedRequest(t, http.MethodGet, endpoint))
		require.NoError(t, err)
		body2, _ := io.ReadAll(resp2.Body)
		assert.Equal(t, "hello", string(body2))
		assert.Equal(t, 1, *calls, "second call must hit cache, not backend")
	})

	t.Run("requests without CacheRequestHeader bypass cache", func(t *testing.T) {
		rt, calls := makeCachedRT(t, time.Minute,
			fakeResponse("r1"),
			fakeResponse("r2"),
		)

		_, err := rt.RoundTrip(newRequest(t, http.MethodGet, endpoint))
		require.NoError(t, err)
		_, err = rt.RoundTrip(newRequest(t, http.MethodGet, endpoint))
		require.NoError(t, err)

		assert.Equal(t, 2, *calls, "requests without CacheRequestHeader must always reach backend")
	})

	t.Run("requests without CacheRequestHeader evict cached entries", func(t *testing.T) {
		rt, calls := makeCachedRT(t, time.Minute,
			fakeResponse("r1"),
			fakeResponse("r2"),
			fakeResponse("r3"),
		)

		// Populate cache
		_, err := rt.RoundTrip(newCachedRequest(t, http.MethodGet, endpoint))
		require.NoError(t, err)
		assert.Equal(t, 1, *calls)

		// Call without CacheRequestHeader — evicts the cached entry and calls backend
		_, err = rt.RoundTrip(newRequest(t, http.MethodGet, endpoint))
		require.NoError(t, err)
		assert.Equal(t, 2, *calls, "request without CacheRequestHeader must bypass cache")

		// Next call with CacheRequestHeader — cache was evicted, backend is called again
		_, err = rt.RoundTrip(newCachedRequest(t, http.MethodGet, endpoint))
		require.NoError(t, err)
		assert.Equal(t, 3, *calls, "evicted entry must not be served from cache")
	})

	t.Run("zero TTL disables caching", func(t *testing.T) {
		rt, calls := makeCachedRT(t, 0,
			fakeResponse("r1"),
			fakeResponse("r2"),
		)

		_, err := rt.RoundTrip(newCachedRequest(t, http.MethodGet, endpoint))
		require.NoError(t, err)
		_, err = rt.RoundTrip(newCachedRequest(t, http.MethodGet, endpoint))
		require.NoError(t, err)

		assert.Equal(t, 2, *calls, "zero TTL must bypass cache")
	})

	t.Run("different URLs have separate cache entries", func(t *testing.T) {
		rt, calls := makeCachedRT(t, time.Minute,
			fakeResponse("url-a"),
			fakeResponse("url-b"),
		)

		resp1, err := rt.RoundTrip(newCachedRequest(t, http.MethodGet, "http://api.example.com/v1/a"))
		require.NoError(t, err)
		body1, _ := io.ReadAll(resp1.Body)

		resp2, err := rt.RoundTrip(newCachedRequest(t, http.MethodGet, "http://api.example.com/v1/b"))
		require.NoError(t, err)
		body2, _ := io.ReadAll(resp2.Body)

		assert.Equal(t, "url-a", string(body1))
		assert.Equal(t, "url-b", string(body2))
		assert.Equal(t, 2, *calls)
	})

	t.Run("different Authorization headers produce separate cache entries", func(t *testing.T) {
		rt, calls := makeCachedRT(t, time.Minute,
			fakeResponse("token-abc"),
			fakeResponse("token-xyz"),
		)

		req1 := newCachedRequest(t, http.MethodGet, endpoint)
		req1.Header.Set("Authorization", "Api-Token abc")
		resp1, err := rt.RoundTrip(req1)
		require.NoError(t, err)
		body1, _ := io.ReadAll(resp1.Body)

		req2 := newCachedRequest(t, http.MethodGet, endpoint)
		req2.Header.Set("Authorization", "Api-Token xyz")
		resp2, err := rt.RoundTrip(req2)
		require.NoError(t, err)
		body2, _ := io.ReadAll(resp2.Body)

		assert.Equal(t, "token-abc", string(body1))
		assert.Equal(t, "token-xyz", string(body2))
		assert.Equal(t, 2, *calls)
	})

	t.Run("expired cache entry triggers a fresh backend call", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			rt, calls := makeCachedRT(t, time.Minute,
				fakeResponse("first"),
				fakeResponse("second"),
			)

			_, err := rt.RoundTrip(newCachedRequest(t, http.MethodGet, endpoint))
			require.NoError(t, err)

			time.Sleep(time.Minute + time.Second)

			resp2, err := rt.RoundTrip(newCachedRequest(t, http.MethodGet, endpoint))
			require.NoError(t, err)
			body2, _ := io.ReadAll(resp2.Body)

			assert.Equal(t, "second", string(body2))
			assert.Equal(t, 2, *calls)
		})
	})

	t.Run("cached response body can be read independently on each call", func(t *testing.T) {
		rt, _ := makeCachedRT(t, time.Minute, fakeResponse("body-content"))

		resp1, err := rt.RoundTrip(newCachedRequest(t, http.MethodGet, endpoint))
		require.NoError(t, err)
		body1, _ := io.ReadAll(resp1.Body)

		resp2, err := rt.RoundTrip(newCachedRequest(t, http.MethodGet, endpoint))
		require.NoError(t, err)
		body2, _ := io.ReadAll(resp2.Body)

		assert.Equal(t, "body-content", string(body1))
		assert.Equal(t, "body-content", string(body2))
	})

	t.Run("cache-hit response has CacheHitHeader set + request copied", func(t *testing.T) {
		rt, _ := makeCachedRT(t, time.Minute, fakeResponse("data"))

		// First call — backend response, header must NOT be set
		req1 := newCachedRequest(t, http.MethodGet, endpoint)
		resp1, err := rt.RoundTrip(req1)
		require.NoError(t, err)
		assert.Empty(t, resp1.Header.Get(CacheHitHeader), "first (uncached) response must not have cache-hit header")
		assert.Same(t, req1, resp1.Request)

		// Second call — served from cache, header must be set + its Request must point to req2, not req1
		req2 := newCachedRequest(t, http.MethodGet, endpoint)
		resp2, err := rt.RoundTrip(req2)
		require.NoError(t, err)
		assert.Equal(t, "true", resp2.Header.Get(CacheHitHeader), "cached response must have cache-hit header")
		assert.Same(t, req2, resp2.Request, "cached response must carry the request that triggered the cache hit")
		assert.NotSame(t, req1, resp2.Request, "cached response must not carry the original request that populated the cache")
	})

	t.Run("setting CacheHitHeader on cached response does not affect stored entry", func(t *testing.T) {
		rt, _ := makeCachedRT(t, time.Minute, fakeResponse("data"))

		// Populate cache
		_, err := rt.RoundTrip(newCachedRequest(t, http.MethodGet, endpoint))
		require.NoError(t, err)

		// Two independent cache-hit responses should each have the header
		resp2, err := rt.RoundTrip(newCachedRequest(t, http.MethodGet, endpoint))
		require.NoError(t, err)
		assert.Equal(t, "true", resp2.Header.Get(CacheHitHeader))

		resp3, err := rt.RoundTrip(newCachedRequest(t, http.MethodGet, endpoint))
		require.NoError(t, err)
		assert.Equal(t, "true", resp3.Header.Get(CacheHitHeader))
	})

	t.Run("response has CacheKeyHeader set for invalidation", func(t *testing.T) {
		rt, _ := makeCachedRT(t, time.Minute, fakeResponse("data"))

		// Fresh response must carry the cache key
		req1 := newCachedRequest(t, http.MethodGet, endpoint)
		resp1, err := rt.RoundTrip(req1)
		require.NoError(t, err)
		assert.NotEmpty(t, resp1.Header.Get(CacheKeyHeader), "fresh response must have CacheKeyHeader set")

		// Cached response must carry the same cache key
		req2 := newCachedRequest(t, http.MethodGet, endpoint)
		resp2, err := rt.RoundTrip(req2)
		require.NoError(t, err)
		assert.Equal(t, resp1.Header.Get(CacheKeyHeader), resp2.Header.Get(CacheKeyHeader), "cached response must have same CacheKeyHeader")
	})

	t.Run("zero TTL removes a previously cached entry", func(t *testing.T) {
		_ = freshCache(t)

		calls := 0
		fake := roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			calls++
			resp := fakeResponse("data")
			resp.Request = r

			return resp, nil
		})

		// First: cache the response with a non-zero TTL
		rtWithTTL := NewCacheRoundTripper(fake, time.Minute)
		_, err := rtWithTTL.RoundTrip(newCachedRequest(t, http.MethodGet, endpoint))
		require.NoError(t, err)
		assert.Equal(t, 1, calls)

		// Second call hits cache, not backend
		_, err = rtWithTTL.RoundTrip(newCachedRequest(t, http.MethodGet, endpoint))
		require.NoError(t, err)
		assert.Equal(t, 1, calls, "second call must hit cache")

		// Now use zero TTL — the cached entry must be removed and the backend must be called
		rtNoTTL := NewCacheRoundTripper(fake, 0)
		_, err = rtNoTTL.RoundTrip(newCachedRequest(t, http.MethodGet, endpoint))
		require.NoError(t, err)
		assert.Equal(t, 2, calls, "zero TTL must remove cached entry and call backend")

		// Restore a non-zero TTL; since the entry was removed, the backend is called again
		_, err = rtWithTTL.RoundTrip(newCachedRequest(t, http.MethodGet, endpoint))
		require.NoError(t, err)
		assert.Equal(t, 3, calls, "after removal, next non-zero TTL call must hit backend again")
	})

	t.Run("cache.set error does not prevent response from being returned", func(t *testing.T) {
		_ = freshCache(t)

		readErr := errors.New("read error")

		fake := roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			badResp := &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{},
				Body:       io.NopCloser(&ioErrReader{err: readErr}),
			}
			badResp.Request = r

			return badResp, nil
		})

		rt := NewCacheRoundTripper(fake, time.Minute)
		resp, err := rt.RoundTrip(newCachedRequest(t, http.MethodGet, endpoint))
		require.NoError(t, err, "round trip must not return an error even if caching fails")
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Entry must NOT have been cached; a second call must reach the backend
		calls := 0
		fake2 := roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			calls++
			resp := fakeResponse("ok")
			resp.Request = r

			return resp, nil
		})
		rt2 := NewCacheRoundTripper(fake2, time.Minute)
		// populate fresh cache for same key
		_, err = rt2.RoundTrip(newCachedRequest(t, http.MethodGet, endpoint))
		require.NoError(t, err)
		assert.Equal(t, 1, calls, "backend must be called because previous set failed")
	})

	t.Run("all successful transport responses are cached regardless of status code", func(t *testing.T) {
		_ = freshCache(t)

		calls := 0
		errorResp := &http.Response{
			StatusCode: http.StatusInternalServerError,
			Header:     make(http.Header),
			Body:       io.NopCloser(bytes.NewBufferString("error")),
		}

		fake := roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			calls++
			errorResp.Request = r

			return errorResp, nil
		})

		rt := NewCacheRoundTripper(fake, time.Minute)

		resp1, err := rt.RoundTrip(newCachedRequest(t, http.MethodGet, endpoint))
		require.NoError(t, err)
		assert.Equal(t, http.StatusInternalServerError, resp1.StatusCode)
		assert.Equal(t, 1, calls)

		// Second call — 500 is served from cache; caller is responsible for invalidation
		resp2, err := rt.RoundTrip(newCachedRequest(t, http.MethodGet, endpoint))
		require.NoError(t, err)
		assert.Equal(t, http.StatusInternalServerError, resp2.StatusCode)
		assert.Equal(t, 1, calls, "500 response must be served from cache; use InvalidateCacheEntry to evict")
	})
}
func TestBuildCacheKey(t *testing.T) {
	makeReq := func(t *testing.T, rawURL, auth string, body []byte) *http.Request {
		t.Helper()
		req, err := http.NewRequest(http.MethodGet, rawURL, bytes.NewReader(body))
		require.NoError(t, err)

		if auth != "" {
			req.Header.Set("Authorization", auth)
		}

		return req
	}

	const base = "http://api.example.com/v1/endpoint"

	t.Run("same request produces same key", func(t *testing.T) {
		req := makeReq(t, base, "token", nil)
		assert.Equal(t, buildCacheKey(req), buildCacheKey(makeReq(t, base, "token", nil)))
	})

	t.Run("different URL produces different key", func(t *testing.T) {
		assert.NotEqual(t,
			buildCacheKey(makeReq(t, base+"/a", "", nil)),
			buildCacheKey(makeReq(t, base+"/b", "", nil)),
		)
	})

	t.Run("different Authorization produces different key", func(t *testing.T) {
		assert.NotEqual(t,
			buildCacheKey(makeReq(t, base, "token-a", nil)),
			buildCacheKey(makeReq(t, base, "token-b", nil)),
		)
	})

	t.Run("body is included in key", func(t *testing.T) {
		assert.NotEqual(t,
			buildCacheKey(makeReq(t, base, "", []byte("payload-a"))),
			buildCacheKey(makeReq(t, base, "", []byte("payload-b"))),
		)
	})

	t.Run("body can still be read after key is computed", func(t *testing.T) {
		req := makeReq(t, base, "", []byte("my-body"))
		_ = buildCacheKey(req)
		body, err := io.ReadAll(req.Body)
		require.NoError(t, err)
		assert.Equal(t, "my-body", string(body))
	})
}

func TestRoundTripperFunc(t *testing.T) {
	t.Run("nil roundTripperFunc causes panic, as it should never happen", func(t *testing.T) {
		var rt roundTripperFunc // nil

		assert.Panics(t, func() {
			_, _ = rt.RoundTrip(nil)
		})
	})
}
