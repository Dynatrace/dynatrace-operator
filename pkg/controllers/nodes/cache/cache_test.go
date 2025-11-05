package cache

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestCache(t *testing.T) {
	t.Run("get non-existing key", func(t *testing.T) {
		cm := corev1.ConfigMap{}
		nodesCache := &Cache{obj: &cm}
		_, err := nodesCache.GetEntry("node1")
		require.ErrorIs(t, err, ErrEntryNotFound)
	})

	t.Run("get non json key", func(t *testing.T) {
		cm := corev1.ConfigMap{Data: map[string]string{"node1": "non-json-key"}}
		nodesCache := &Cache{obj: &cm}
		_, err := nodesCache.GetEntry("node1")
		require.EqualError(t, err, "invalid character 'o' in literal null (expecting 'u')")
	})

	t.Run("set cache key if configmap data nil", func(t *testing.T) {
		cm := corev1.ConfigMap{}
		nodesCache := &Cache{obj: &cm}
		err := nodesCache.SetEntry("node1", Entry{
			IPAddress: "10.128.0.48",
		})
		require.NoError(t, err)

		entry, err := nodesCache.GetEntry("node1")
		require.NoError(t, err)

		assert.Equal(t, Entry{
			IPAddress: "10.128.0.48",
		}, entry)
	})

	t.Run("get all cache keys if configmap data nil", func(t *testing.T) {
		cm := corev1.ConfigMap{}
		nodesCache := &Cache{obj: &cm}
		keys := nodesCache.Keys()
		assert.Equal(t, []string{}, keys)
	})

	t.Run("check if cache is not outdated", func(t *testing.T) {
		cm := corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{lastUpdatedCacheAnnotation: ""}}}
		nodesCache := &Cache{obj: &cm}
		assert.False(t, nodesCache.IsOutdated(time.Now()))
	})
}
