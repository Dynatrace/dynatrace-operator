package nodes

import (
	"encoding/json"
	"time"

	err "github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
)

const (
	cacheName = "dynatrace-node-cache"
)

var cacheLifetime = 10 * time.Minute
var lastUpdatedCacheTime = "DTOperatorLastUpdated"

// ErrNotFound is returned when entry hasn't been found on the cache.Ø
var ErrNotFound = err.New("not found")

// CacheEntry constains information about a Node.
type CacheEntry struct {
	Instance                 string    `json:"instance"`
	IPAddress                string    `json:"ip"`
	LastSeen                 time.Time `json:"seen"`
	LastMarkedForTermination time.Time `json:"marked"`
}

// Cache manages information about Nodes.
type Cache struct {
	Obj    *corev1.ConfigMap
	Create bool
	upd    bool
}

// Get returns the information about node, or error if not found or failed to unmarshall the data.
func (nodeCache *Cache) Get(node string) (CacheEntry, error) {
	if nodeCache.Obj.Data == nil {
		return CacheEntry{}, ErrNotFound
	}

	raw, ok := nodeCache.Obj.Data[node]
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
func (nodeCache *Cache) Set(node string, entry CacheEntry) error {
	raw, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	if nodeCache.Obj.Data == nil {
		nodeCache.Obj.Data = map[string]string{}
	}
	nodeCache.Obj.Data[node] = string(raw)
	nodeCache.upd = true
	return nil
}

// Delete removes the node from the cache.
func (nodeCache *Cache) Delete(node string) {
	if nodeCache.Obj.Data != nil {
		delete(nodeCache.Obj.Data, node)
		nodeCache.upd = true
	}
}

// Keys returns a list of node names on the cache.
func (nodeCache *Cache) Keys() []string {
	if nodeCache.Obj.Data == nil {
		return []string{}
	}

	out := make([]string, 0, len(nodeCache.Obj.Data))
	for k := range nodeCache.Obj.Data {
		out = append(out, k)
	}
	return out
}

// Changed returns true if changes have been made to the cache instance.
func (nodeCache *Cache) Changed() bool {
	return nodeCache.Create || nodeCache.upd
}

func (nodeCache *Cache) ContainsKey(key string) bool {
	for _, e := range nodeCache.Keys() {
		if e == key {
			return true
		}
	}
	return false
}

func (nodeCache *Cache) IsCacheOutdated() bool {
	if lastUpdated, ok := nodeCache.Obj.Annotations[lastUpdatedCacheTime]; ok {
		if lastUpdatedTime, err := time.Parse(time.RFC3339, lastUpdated); err == nil {
			return lastUpdatedTime.Add(cacheLifetime).Before(time.Now())
		} else {
			return false
		}
	}
	return true // Cache is not annotated -> outdated
}

func (nodeCache *Cache) UpdateTimestamp() {
	if nodeCache.Obj.Annotations == nil {
		nodeCache.Obj.Annotations = make(map[string]string)
	}
	nodeCache.Obj.Annotations[lastUpdatedCacheTime] = time.Now().Format(time.RFC3339)
	nodeCache.upd = true
}

func (nodeCache *Cache) updateLastMarkedForTerminationTimestamp(nodeInfo CacheEntry, nodeName string) error {
	nodeInfo.LastMarkedForTermination = time.Now().UTC()
	return nodeCache.Set(nodeName, nodeInfo)
}
