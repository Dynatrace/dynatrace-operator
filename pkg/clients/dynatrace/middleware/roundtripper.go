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

type RoundTripperFunc func(*http.Request) (*http.Response, error)

func (rt RoundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	if rt == nil {
		return http.DefaultTransport.RoundTrip(r)
	}

	return rt(r)
}

func CacheRoundTripper(next http.RoundTripper, ttl time.Duration) http.RoundTripper {
	return RoundTripperFunc(func(r *http.Request) (*http.Response, error) {
		if r.Method != http.MethodGet ||
			ttl == 0 ||
			r.Header.Get("Accept") == "application/octet-stream" {
			return next.RoundTrip(r)
		}

		cacheKey := buildCacheKey(r)

		cachedResponse := cache.get(cacheKey)
		if cachedResponse != nil {
			log.Info("client using cached response", "endpoint", r.URL)

			return cachedResponse, nil
		}

		// send the actual request
		resp, err := next.RoundTrip(r)
		if err == nil {
			cache.store(cacheKey, resp, ttl)
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
