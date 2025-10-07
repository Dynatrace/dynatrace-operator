package secrets

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	testNamespace = "test-ns"
	sourceSecret  = "source-secret"
	targetSecret  = "target-secret"
)

func TestEnsureReplicated(t *testing.T) {
	logger := logd.Get().WithName("test")

	t.Run("target already exists -> no replication", func(t *testing.T) {
		clt := fake.NewClient(
			&corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: targetSecret, Namespace: testNamespace}, Data: map[string][]byte{"foo": []byte("bar")}},
		)

		req := newRequest(t)
		err := EnsureReplicated(req, clt, clt, sourceSecret, targetSecret, logger)
		require.NoError(t, err)

		// ensure secret unchanged
		var s corev1.Secret
		require.NoError(t, clt.Get(t.Context(), client.ObjectKey{Name: targetSecret, Namespace: testNamespace}, &s))
		assert.Equal(t, []byte("bar"), s.Data["foo"])
	})

	t.Run("target missing + source present -> replication creates target", func(t *testing.T) {
		clt := fake.NewClient(
			&corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: sourceSecret, Namespace: testNamespace}, Data: map[string][]byte{"foo": []byte("bar")}},
		)

		req := newRequest(t)
		err := EnsureReplicated(req, clt, clt, sourceSecret, targetSecret, logger)
		require.NoError(t, err)

		var s corev1.Secret
		require.NoError(t, clt.Get(t.Context(), client.ObjectKey{Name: targetSecret, Namespace: testNamespace}, &s))
		assert.Equal(t, []byte("bar"), s.Data["foo"])
	})

	t.Run("target + source both missing -> returns not found error", func(t *testing.T) {
		clt := fake.NewClient()
		req := newRequest(t)

		err := EnsureReplicated(req, clt, clt, sourceSecret, targetSecret, logger)
		require.Error(t, err)
		assert.True(t, k8serrors.IsNotFound(err), "expected not found error from missing source secret")

		var s corev1.Secret
		errGet := clt.Get(t.Context(), client.ObjectKey{Name: targetSecret, Namespace: testNamespace}, &s)
		assert.True(t, k8serrors.IsNotFound(errGet), "target secret should not be created on failure")
	})
}

func newRequest(t *testing.T) *mutator.MutationRequest {
	pod := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "pod", Namespace: testNamespace}}
	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: testNamespace}}
	dk := &dynakube.DynaKube{ObjectMeta: metav1.ObjectMeta{Name: "dk", Namespace: testNamespace}}
	return mutator.NewMutationRequest(t.Context(), *ns, nil, pod, *dk)
}
