package middleware

import (
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/utils/lru"
)

func freshLRUCache(t *testing.T) *lru.Cache {
	t.Helper()
	freshCache := lru.New(1)

	return freshCache
}

func TestLRUCache_SetAndGet(t *testing.T) {
	t.Run("miss on empty cache", func(t *testing.T) {
		c := freshLRUCache(t)
		entry, ok := c.Get("anykey")
		assert.False(t, ok)
		assert.Nil(t, entry)
	})

	t.Run("nil response is not stored", func(t *testing.T) { // fails, but not end of the world
		c := freshLRUCache(t)
		c.Add("key", nil)
		entry, ok := c.Get("key")
		assert.False(t, ok)
		assert.Nil(t, entry)
	})

	t.Run("body can be read multiple times from cached response", func(t *testing.T) { // fails, can't read body
		c := freshLRUCache(t)
		c.Add("key", fakeResponse("body-content"))

		entry, _ := c.Get("key")
		resp1 := entry.(*http.Response) // We would need to write a wrapper for this
		require.NotNil(t, resp1)
		body1, err := io.ReadAll(resp1.Body)
		require.NoError(t, err)

		entry, _ = c.Get("key")
		resp2 := entry.(*http.Response)
		require.NotNil(t, resp2)
		body2, _ := io.ReadAll(resp2.Body)

		assert.Equal(t, "body-content", string(body1))
		assert.Equal(t, "body-content", string(body2))
	})
}
