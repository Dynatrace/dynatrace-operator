package middleware

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"time"
)

const ( // CacheHitHeader is set on responses served from the in-memory cache so that
	// the core client can include a "cached" field in its log output.
	CacheHitHeader = "X-DT-Cache"

	// CacheRequestHeader must be set on a request to opt in to in-memory caching.
	// Only GET requests with this header set will be cached or served from cache.
	CacheRequestHeader = "X-DT-Cache-Request"

	// CacheKeyHeader is set on responses (both fresh and cached) by the cache middleware
	// to inform the core client of the cache key used for the request.
	CacheKeyHeader = "X-DT-Cache-Key"
)

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (rt roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	if rt == nil {
		panic("empty roundTripperFunc used")
	}

	return rt(r)
}

func NewCacheRoundTripper(next http.RoundTripper, ttl time.Duration) http.RoundTripper {
	return roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		if r.Header.Get("Accept") == "application/octet-stream" {
			return next.RoundTrip(r)
		}

		cacheKey := buildCacheKey(r)

		if ttl == 0 || r.Header.Get(CacheRequestHeader) == "" {
			// if the caching was turned off intermittently, those entries should be removed
			cache.remove(cacheKey)

			return next.RoundTrip(r)
		}

		cachedResponse := cache.get(cacheKey)
		if cachedResponse != nil {
			cachedResponse.Header.Set(CacheHitHeader, "true")
			cachedResponse.Header.Set(CacheKeyHeader, cacheKey)
			cachedResponse.Request = r

			return cachedResponse, nil
		}

		// send the actual request
		resp, err := next.RoundTrip(r)
		if err == nil {
			resp.Header.Set(CacheKeyHeader, cacheKey)
			// err is ignored, as cache.set can only error due to failing to read the body, which will cause an error down the line anyway
			// adding extra logs will just repeat the same, adding no value
			_ = cache.set(cacheKey, resp, ttl)

			return resp, nil
		}

		return resp, err
	})
}

func buildCacheKey(r *http.Request) string {
	h := sha256.New()

	fmt.Fprint(h, r.URL.String())
	fmt.Fprint(h, r.Header.Get("Authorization"))

	if r.Body != nil {
		body, _ := io.ReadAll(r.Body)
		r.Body = io.NopCloser(bytes.NewReader(body))
		h.Write(body)
	}

	return hex.EncodeToString(h.Sum(nil))
}
