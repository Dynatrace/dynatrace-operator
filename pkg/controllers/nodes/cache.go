package nodes

import (
	"encoding/json"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/util/timeprovider"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
)

// ErrNotFound is returned when entry hasn't been found on the cache.
var ErrNotFound = errors.New("not found")

// CacheEntry contains information about a Node.
type CacheEntry struct {
	LastSeen                 time.Time `json:"seen"`
	LastMarkedForTermination time.Time `json:"marked"`
	Instance                 string    `json:"instance"`
	IPAddress                string    `json:"ip"`
}

// Cache manages information about Nodes.
type Cache struct {
	Obj          *corev1.ConfigMap
	timeProvider *timeprovider.Provider
	Create       bool
	upd          bool
}

// Get returns the information about node, or error if not found or failed to unmarshall the data.
func (cache *Cache) Get(node string) (CacheEntry, error) {
	if cache.Obj.Data == nil {
		return CacheEntry{}, ErrNotFound
	}

	raw, ok := cache.Obj.Data[node]
	if !ok {
		return CacheEntry{}, ErrNotFound
	}

	var out CacheEntry
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return CacheEntry{}, err
	}

	return out, nil
}

// Set updates the information about node, or error if failed to marshall the data.
func (cache *Cache) Set(node string, entry CacheEntry) error {
	raw, err := json.Marshal(entry)
	if err != nil {
		return err
	}

	if cache.Obj.Data == nil {
		cache.Obj.Data = map[string]string{}
	}

	cache.Obj.Data[node] = string(raw)
	cache.upd = true

	return nil
}

// Delete removes the node from the cache.
func (cache *Cache) Delete(node string) {
	if cache.Obj.Data != nil {
		delete(cache.Obj.Data, node)
		cache.upd = true
	}
}

// Keys returns a list of node names on the cache.
func (cache *Cache) Keys() []string {
	if cache.Obj.Data == nil {
		return []string{}
	}

	out := make([]string, 0, len(cache.Obj.Data))
	for k := range cache.Obj.Data {
		out = append(out, k)
	}

	return out
}

// Changed returns true if changes have been made to the cache instance.
func (cache *Cache) Changed() bool {
	return cache.Create || cache.upd
}

func (cache *Cache) IsCacheOutdated() bool {
	if lastUpdated, ok := cache.Obj.Annotations[lastUpdatedCacheAnnotation]; ok {
		if lastUpdatedTime, err := time.Parse(time.RFC3339, lastUpdated); err == nil {
			return lastUpdatedTime.Add(cacheLifetime).Before(cache.timeProvider.Now().UTC())
		} else {
			return false
		}
	}

	return true // Cache is not annotated -> outdated
}

func (cache *Cache) UpdateTimestamp() {
	if cache.Obj.Annotations == nil {
		cache.Obj.Annotations = make(map[string]string)
	}

	cache.Obj.Annotations[lastUpdatedCacheAnnotation] = cache.timeProvider.Now().Format(time.RFC3339)
	cache.upd = true
}

func (cache *Cache) updateLastMarkedForTerminationTimestamp(nodeInfo CacheEntry, nodeName string) error {
	nodeInfo.LastMarkedForTermination = cache.timeProvider.Now().UTC()

	return cache.Set(nodeName, nodeInfo)
}
