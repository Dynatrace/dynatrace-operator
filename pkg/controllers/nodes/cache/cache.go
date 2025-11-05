package cache

import (
	"context"
	"encoding/json"
	"slices"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	ConfigMapName              = "dynatrace-node-cache"
	cacheLifetime              = 10 * time.Minute
	lastUpdatedCacheAnnotation = "DTOperatorLastUpdated"
)

// ErrEntryNotFound is returned when entry hasn't been found in the cache.
var ErrEntryNotFound = errors.New("node entry not found")

// Entry contains information about a Node where a Dynatrace OneAgent is/was.
type Entry struct {
	LastSeen                 time.Time `json:"seen"`
	LastMarkedForTermination time.Time `json:"marked"`
	IPAddress                string    `json:"ip"`

	// Only informational
	NodeName     string `json:"-"`        // only here to simplify the code
	DynaKubeName string `json:"instance"` // the mismatch is intentional, this has no real purpose, left it in for compatibility reasons
}

// IsMarkableForTermination checks if the timestamp from last mark is at least one hour old
func (entry *Entry) IsMarkableForTermination(now time.Time) bool {
	// If the last mark was an hour ago, mark again
	// Zero value for time.Time is 0001-01-01, so first mark is also executed
	lastMarked := entry.LastMarkedForTermination

	return lastMarked.UTC().Add(time.Hour).Before(now)
}

func (entry *Entry) SetLastMarkedForTerminationTimestamp(now time.Time) {
	entry.LastMarkedForTermination = now
}

// Cache manages information about Nodes where Dynatrace OneAgents are/were running.
// There is a reason behind not directly having a `map[string]Entry`: "lazy parsing".
// The `map` within the ConfigMap can have 100+ entries, if the k8s cluster was big enough.
// So the idea is that we only parse the Entry that we are actually working on, and leave the rest alone. (we still load it into memory, but that comes with having the cache in memory :P )
// The Reconcile loop will only work with 1 Node at a time, so it make sense to not always parse the other n-1 entries for no good reason.
//
// Every now and then, we will need to clean up the cache, that is when we will need to parse all of the Entries, but this only happens every 10m.
// Note: This logic is old, and was only cleaned up for (hopefully) better understandability.
type Cache struct {
	obj    *corev1.ConfigMap
	create bool
	upd    bool
}

func New(ctx context.Context, apiReader client.Reader, ns string, owner client.Object) (*Cache, error) {
	var cm corev1.ConfigMap

	err := apiReader.Get(ctx, client.ObjectKey{Name: ConfigMapName, Namespace: ns}, &cm)
	if err == nil {
		return newCache(&cm, false), nil
	}

	if k8serrors.IsNotFound(err) {
		cm := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      ConfigMapName,
				Namespace: ns,
			},
			Data: map[string]string{},
		}
		// If running locally, don't set the controller.
		if owner != nil {
			if err = controllerutil.SetControllerReference(owner, cm, scheme.Scheme); err != nil {
				return nil, err
			}
		}

		return newCache(cm, true), nil
	}

	return nil, err
}

func newCache(data *corev1.ConfigMap, create bool) *Cache {
	return &Cache{
		obj:    data,
		create: create,
	}
}

// GetEntry returns the information about node, or error if not found or failed to unmarshall the data.
func (cache *Cache) GetEntry(node string) (Entry, error) {
	if cache.obj.Data == nil {
		return Entry{}, ErrEntryNotFound
	}

	raw, ok := cache.obj.Data[node]
	if !ok {
		return Entry{}, ErrEntryNotFound
	}

	var out Entry
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return Entry{}, err
	}

	out.NodeName = node

	return out, nil
}

// SetEntry updates the information about node, or error if failed to marshall the data.
func (cache *Cache) SetEntry(node string, entry Entry) error {
	raw, err := json.Marshal(entry)
	if err != nil {
		return err
	}

	if cache.obj.Data == nil {
		cache.obj.Data = map[string]string{}
	}

	cache.obj.Data[node] = string(raw)
	cache.upd = true

	return nil
}

// DeleteEntry removes the node from the cache.
func (cache *Cache) DeleteEntry(node string) {
	if cache.obj.Data != nil {
		delete(cache.obj.Data, node)
		cache.upd = true
	}
}

// Keys returns a list of node names on the cache.
func (cache *Cache) Keys() []string {
	if cache.obj.Data == nil {
		return []string{}
	}

	out := make([]string, 0, len(cache.obj.Data))
	for k := range cache.obj.Data {
		out = append(out, k)
	}

	return out
}

// Changed returns true if changes have been made to the cache instance.
func (cache *Cache) Changed() bool {
	return cache.create || cache.upd
}

func (cache *Cache) Store(ctx context.Context, client client.Client) error {
	if !cache.Changed() {
		return nil
	}

	if cache.create {
		return client.Create(ctx, cache.obj)
	}

	if err := client.Update(ctx, cache.obj); err != nil {
		return err
	}

	return nil
}

func (cache *Cache) IsOutdated(now time.Time) bool {
	if lastUpdated, ok := cache.obj.Annotations[lastUpdatedCacheAnnotation]; ok {
		if lastUpdatedTime, err := time.Parse(time.RFC3339, lastUpdated); err == nil {
			return lastUpdatedTime.Add(cacheLifetime).Before(now)
		} else {
			return false
		}
	}

	return true // Cache is not annotated == outdated
}

func (cache *Cache) UpdateTimestamp(now time.Time) {
	if cache.obj.Annotations == nil {
		cache.obj.Annotations = make(map[string]string)
	}

	cache.obj.Annotations[lastUpdatedCacheAnnotation] = now.Format(time.RFC3339)
	cache.upd = true
}

// Prune will collect the nodeNames from the Cache that do not have a corresponding k8s Node in cluster.
// - We return these nodeNames to the Controller, to send a mark for termination if need, just in case.
// It will also remove the Entries that have a corresponding k8s Node in cluster, but have not had a OneAgent on them for a while.
func (cache *Cache) Prune(ctx context.Context, client client.Client, now time.Time) ([]string, error) {
	var nodeLst corev1.NodeList
	if err := client.List(ctx, &nodeLst); err != nil {
		return nil, err
	}

	toBePruned := []string{}

	for _, cachedNodeName := range cache.Keys() {
		if slices.ContainsFunc(nodeLst.Items, func(clusterNode corev1.Node) bool { return clusterNode.Name == cachedNodeName }) {
			_ = cache.removeStaleEntry(now, cachedNodeName)
		} else {
			toBePruned = append(toBePruned, cachedNodeName)
		}
	}

	return toBePruned, nil
}

// removeStaleEntry will remove the entries from the Cache, where we haven't seen a OneAgent fro more than an hour.
func (cache *Cache) removeStaleEntry(now time.Time, nodeName string) bool {
	entry, err := cache.GetEntry(nodeName)
	if err != nil {
		return false
	}

	isNodeDeletable := now.Sub(entry.LastSeen).Hours() > 1 || entry.IPAddress == ""

	if isNodeDeletable {
		cache.DeleteEntry(entry.NodeName)

		return true
	}

	return false
}
