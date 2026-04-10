package middleware

import (
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/hashicorp/golang-lru/v2/expirable"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func freshHashicorpCache(t *testing.T) *expirable.LRU[string, http.Response] {
	t.Helper()
	freshCache := expirable.NewLRU[string, http.Response](0, nil, time.Hour)
	t.Cleanup(func() { cache = newCache() })

	return freshCache
}

func TestHashicorpCache_SetAndGet(t *testing.T) {
	t.Run("miss on empty cache", func(t *testing.T) {
		c := freshHashicorpCache(t)
		entry, ok := c.Get("anykey")
		assert.False(t, ok)
		assert.Nil(t, entry)
	})

	t.Run("nil response is not stored", func(t *testing.T) {
		c := freshHashicorpCache(t)
		c.Add("key", http.Response{})
		entry, ok := c.Get("key")
		assert.False(t, ok)
		assert.Nil(t, entry)
	})

	t.Run("body can be read multiple times from cached response", func(t *testing.T) { // fails, can't read body
		c := freshHashicorpCache(t)
		c.Add("key", *fakeResponse("body-content"))

		resp1, _ := c.Get("key")
		require.NotNil(t, resp1)
		body1, err := io.ReadAll(resp1.Body)
		require.NoError(t, err)

		resp2, _ := c.Get("key")
		require.NotNil(t, resp2)
		body2, _ := io.ReadAll(resp2.Body)

		assert.Equal(t, "body-content", string(body1))
		assert.Equal(t, "body-content", string(body2))
	})
}
