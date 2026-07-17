package integrationtests

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Testing reconcilers that use a cached client.
//
// Reconcilers read through the manager's cached client, which keeps them simple and cuts apiserver
// calls but is eventually consistent: a write reaches the apiserver at once, while the informer
// cache that serves reads catches up a moment later. So a reconciler that writes an object and then
// reads it back — a read-after-write — can act on a stale copy, a real class of bug.
//
// The test must run on that same cached client to exercise this path; a direct apiserver client is
// read-after-write consistent, so it would never hit it and would hide those bugs.
//
// Approach: ensure the cache reflects a setup write before reconciling. This mirrors production,
// where an informer event carries the change into the cache before it triggers a reconcile. Per
// phase:
//
//   - Arrange: after a setup write, WaitForCachedMatch / WaitForCachedGone until the cache observes it.
//   - Act: exactly one reconcile, never wrapped in require.Eventually.
//   - Assert: through the direct apiReader.
//
// Rules:
//
//   - Check the result with apiReader, not the value Reconcile returns. A reconcile may write an
//     object and then read it back from the cache to build its return value; because the cache lags,
//     that read can catch the old or the new object, so the same call may return different things on
//     different runs. The object in the apiserver is stable, so assert on that instead.
//   - Before checking that a reconcile changes nothing (a no-op), first WaitForCached the previous
//     write, so the reconcile reads an up-to-date cache and does not rewrite the object.
//   - Keep exactly one reconcile per phase. Only wrap a reconcile in require.Eventually if it
//     genuinely needs several passes on its own to settle (rare).
//
// Reference examples: pkg/controllers/dynakube/kubemon/statefulset and .../kubemon/connectioninfo.

const (
	cacheSyncTimeout = 5 * time.Second
	cacheSyncTick    = 50 * time.Millisecond
)

// WaitForCachedMatch blocks until reader (a cached client) observes obj at key and match(obj)
// holds, modeling the informer delivering a watch event into the cache before a reconcile.
func WaitForCachedMatch[T client.Object](t *testing.T, reader client.Reader, key client.ObjectKey, obj T, match func(T) bool) {
	t.Helper()

	require.Eventually(t, func() bool {
		if err := reader.Get(t.Context(), key, obj); err != nil {
			return false
		}

		return match(obj)
	}, cacheSyncTimeout, cacheSyncTick)
}

// WaitForCachedGone blocks until reader (a cached client) no longer observes obj at key, modeling
// the informer delivering a delete watch event into the cache before a reconcile.
func WaitForCachedGone(t *testing.T, reader client.Reader, key client.ObjectKey, obj client.Object) {
	t.Helper()

	require.Eventually(t, func() bool {
		return k8serrors.IsNotFound(reader.Get(t.Context(), key, obj))
	}, cacheSyncTimeout, cacheSyncTick)
}
