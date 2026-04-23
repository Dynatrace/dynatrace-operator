package middleware

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/core"
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
		if r.Method != http.MethodGet ||
			r.Header.Get("Accept") == "application/octet-stream" {
			return next.RoundTrip(r)
		}

		cacheKey := buildCacheKey(r)

		if ttl == 0 || r.Header.Get(core.CacheSkipHeader) != "" {
			// if the caching was turned off intermittently, those entries should be removed
			cache.remove(cacheKey)

			return next.RoundTrip(r)
		}

		cachedResponse := cache.get(cacheKey)
		if cachedResponse != nil {
			cachedResponse.Header.Set(core.CacheHitHeader, "true")
			cachedResponse.Request = r

			return cachedResponse, nil
		}

		// send the actual request
		resp, err := next.RoundTrip(r)
		if err == nil && core.IsSuccessResponse(resp) {
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
