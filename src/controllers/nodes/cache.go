package nodes

import (
	"encoding/json"
	"errors"
	"time"

	corev1 "k8s.io/api/core/v1"
)

// ErrNotFound is returned when entry hasn't been found on the cache.
var ErrNotFound = errors.New("not found")

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
func (c *Cache) Get(node string) (CacheEntry, error) {
	if c.Obj.Data == nil {
		return CacheEntry{}, ErrNotFound
	}

	raw, ok := c.Obj.Data[node]
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
func (c *Cache) Set(node string, entry CacheEntry) error {
	raw, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	if c.Obj.Data == nil {
		c.Obj.Data = map[string]string{}
	}
	c.Obj.Data[node] = string(raw)
	c.upd = true
	return nil
}

// Delete removes the node from the cache.
func (c *Cache) Delete(node string) {
	if c.Obj.Data != nil {
		delete(c.Obj.Data, node)
		c.upd = true
	}
}

// Keys returns a list of node names on the cache.
func (c *Cache) Keys() []string {
	if c.Obj.Data == nil {
		return []string{}
	}

	out := make([]string, 0, len(c.Obj.Data))
	for k := range c.Obj.Data {
		out = append(out, k)
	}
	return out
}

// Changed returns true if changes have been made to the cache instance.
func (c *Cache) Changed() bool {
	return c.Create || c.upd
}
