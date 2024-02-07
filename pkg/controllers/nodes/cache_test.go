package nodes

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestCache(t *testing.T) {
	t.Run("get non-existing key", func(t *testing.T) {
		cm := corev1.ConfigMap{}
		nodesCache := &Cache{Obj: &cm}
		_, err := nodesCache.Get("node1")
		require.ErrorIs(t, err, ErrNotFound)
	})

	t.Run("get non json key", func(t *testing.T) {
		cm := corev1.ConfigMap{Data: map[string]string{"node1": "non-json-key"}}
		nodesCache := &Cache{Obj: &cm}
		_, err := nodesCache.Get("node1")
		require.EqualError(t, err, "invalid character 'o' in literal null (expecting 'u')")
	})

	t.Run("set cache key if configmap data nil", func(t *testing.T) {
		cm := corev1.ConfigMap{}
		nodesCache := &Cache{Obj: &cm}
		err := nodesCache.Set("node1", CacheEntry{
			Instance:  "dynakube",
			IPAddress: "10.128.0.48",
		})
		require.NoError(t, err)

		entry, err := nodesCache.Get("node1")
		require.NoError(t, err)

		assert.Equal(t, CacheEntry{
			Instance:  "dynakube",
			IPAddress: "10.128.0.48",
		}, entry)
	})

	t.Run("get all cache keys if configmap data nil", func(t *testing.T) {
		cm := corev1.ConfigMap{}
		nodesCache := &Cache{Obj: &cm}
		keys := nodesCache.Keys()
		assert.Equal(t, []string{}, keys)
	})

	t.Run("check if cache is not outdated", func(t *testing.T) {
		cm := corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{lastUpdatedCacheAnnotation: ""}}}
		nodesCache := &Cache{Obj: &cm}
		assert.False(t, nodesCache.IsCacheOutdated())
	})
}
