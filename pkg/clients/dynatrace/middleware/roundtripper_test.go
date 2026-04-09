package middleware

import (
	"bytes"
	"io"
	"net/http"
	"net/url"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newRequest(t *testing.T, method, rawURL string) *http.Request {
	t.Helper()
	u, err := url.Parse(rawURL)
	require.NoError(t, err)

	return &http.Request{Method: method, URL: u, Header: make(http.Header)}
}

func TestNewCacheRoundTripper(t *testing.T) {
	const endpoint = "http://api.example.com/v1/resource"

	makeCachedRT := func(t *testing.T, ttl time.Duration, responses ...*http.Response) (http.RoundTripper, *int) {
		t.Helper()
		_ = freshCache(t) // reset global cache, cleaned up after test

		calls := 0
		idx := 0

		fake := RoundTripperFunc(func(r *http.Request) (*http.Response, error) {
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

		resp1, err := rt.RoundTrip(newRequest(t, http.MethodGet, endpoint))
		require.NoError(t, err)
		body1, _ := io.ReadAll(resp1.Body)
		assert.Equal(t, "hello", string(body1))
		assert.Equal(t, 1, *calls)

		resp2, err := rt.RoundTrip(newRequest(t, http.MethodGet, endpoint))
		require.NoError(t, err)
		body2, _ := io.ReadAll(resp2.Body)
		assert.Equal(t, "hello", string(body2))
		assert.Equal(t, 1, *calls, "second call must hit cache, not backend")
	})

	t.Run("non-GET requests bypass cache", func(t *testing.T) {
		rt, calls := makeCachedRT(t, time.Minute,
			fakeResponse("r1"),
			fakeResponse("r2"),
		)

		_, err := rt.RoundTrip(newRequest(t, http.MethodPost, endpoint))
		require.NoError(t, err)
		_, err = rt.RoundTrip(newRequest(t, http.MethodPost, endpoint))
		require.NoError(t, err)

		assert.Equal(t, 2, *calls, "POST must always reach backend")
	})

	t.Run("zero TTL disables caching", func(t *testing.T) {
		rt, calls := makeCachedRT(t, 0,
			fakeResponse("r1"),
			fakeResponse("r2"),
		)

		_, err := rt.RoundTrip(newRequest(t, http.MethodGet, endpoint))
		require.NoError(t, err)
		_, err = rt.RoundTrip(newRequest(t, http.MethodGet, endpoint))
		require.NoError(t, err)

		assert.Equal(t, 2, *calls, "zero TTL must bypass cache")
	})

	t.Run("octet-stream Accept header bypasses cache", func(t *testing.T) {
		rt, calls := makeCachedRT(t, time.Minute,
			fakeResponse("bin1"),
			fakeResponse("bin2"),
		)

		req1 := newRequest(t, http.MethodGet, endpoint)
		req1.Header.Set("Accept", "application/octet-stream")
		_, err := rt.RoundTrip(req1)
		require.NoError(t, err)

		req2 := newRequest(t, http.MethodGet, endpoint)
		req2.Header.Set("Accept", "application/octet-stream")
		_, err = rt.RoundTrip(req2)
		require.NoError(t, err)

		assert.Equal(t, 2, *calls, "octet-stream requests must bypass cache")
	})

	t.Run("different URLs have separate cache entries", func(t *testing.T) {
		rt, calls := makeCachedRT(t, time.Minute,
			fakeResponse("url-a"),
			fakeResponse("url-b"),
		)

		resp1, err := rt.RoundTrip(newRequest(t, http.MethodGet, "http://api.example.com/v1/a"))
		require.NoError(t, err)
		body1, _ := io.ReadAll(resp1.Body)

		resp2, err := rt.RoundTrip(newRequest(t, http.MethodGet, "http://api.example.com/v1/b"))
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

		req1 := newRequest(t, http.MethodGet, endpoint)
		req1.Header.Set("Authorization", "Api-Token abc")
		resp1, err := rt.RoundTrip(req1)
		require.NoError(t, err)
		body1, _ := io.ReadAll(resp1.Body)

		req2 := newRequest(t, http.MethodGet, endpoint)
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

			_, err := rt.RoundTrip(newRequest(t, http.MethodGet, endpoint))
			require.NoError(t, err)

			time.Sleep(5 * time.Minute)

			resp2, err := rt.RoundTrip(newRequest(t, http.MethodGet, endpoint))
			require.NoError(t, err)
			body2, _ := io.ReadAll(resp2.Body)

			assert.Equal(t, "second", string(body2))
			assert.Equal(t, 2, *calls)
		})
	})

	t.Run("cached response body can be read independently on each call", func(t *testing.T) {
		rt, _ := makeCachedRT(t, time.Minute, fakeResponse("body-content"))

		resp1, err := rt.RoundTrip(newRequest(t, http.MethodGet, endpoint))
		require.NoError(t, err)
		body1, _ := io.ReadAll(resp1.Body)

		resp2, err := rt.RoundTrip(newRequest(t, http.MethodGet, endpoint))
		require.NoError(t, err)
		body2, _ := io.ReadAll(resp2.Body)

		assert.Equal(t, "body-content", string(body1))
		assert.Equal(t, "body-content", string(body2))
	})

	t.Run("error responses are not cached", func(t *testing.T) {
		_ = freshCache(t)

		calls := 0
		errorResp := &http.Response{
			StatusCode: http.StatusInternalServerError,
			Body:       io.NopCloser(bytes.NewBufferString("error")),
		}
		successResp := fakeResponse("ok")

		fake := RoundTripperFunc(func(r *http.Request) (*http.Response, error) {
			calls++
			errorResp.Request = r
			successResp.Request = r
			if calls == 1 {
				return errorResp, nil
			}

			return successResp, nil
		})

		rt := NewCacheRoundTripper(fake, time.Minute)

		resp1, err := rt.RoundTrip(newRequest(t, http.MethodGet, endpoint))
		require.NoError(t, err)
		assert.Equal(t, http.StatusInternalServerError, resp1.StatusCode)

		resp2, err := rt.RoundTrip(newRequest(t, http.MethodGet, endpoint))
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp2.StatusCode)

		assert.Equal(t, 2, calls, "error response must not be cached; second call must reach backend")
	})
}

func TestBuildCacheKey(t *testing.T) {
	makeReq := func(t *testing.T, rawURL, auth string, body []byte) *http.Request {
		t.Helper()
		u, err := url.Parse(rawURL)
		require.NoError(t, err)
		req := &http.Request{URL: u, Header: make(http.Header)}
		if auth != "" {
			req.Header.Set("Authorization", auth)
		}
		if body != nil {
			req.Body = io.NopCloser(bytes.NewReader(body))
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
