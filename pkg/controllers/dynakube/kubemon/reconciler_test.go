package kubemon

import (
	"errors"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	kubemonapi "github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/kubemon"
	kubemonstatefulset "github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/kubemon/statefulset"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8senv"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/objects/k8sstatefulset"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Unit tests for the kubemon orchestrator. All sub-reconcilers are mocked, so these tests own only
// the orchestration logic; sub-reconciler internals are covered in their own packages.

// TestReconcileDisabled covers removal of an existing condition once cleanup succeeds.
func TestReconcileDisabled(t *testing.T) {
	t.Setenv(k8senv.KubemonEnableOperand, "true") // remove with gate
	t.Run("removes condition when disabled and cleanup succeeds", func(t *testing.T) {
		connInfoReconciler := newMockConnectionInfoReconciler(t)
		statefulSetReconciler := newMockStatefulsetReconciler(t)
		reconciler := &Reconciler{
			connectionInfoReconciler: connInfoReconciler,
			statefulsetReconciler:    statefulSetReconciler,
		}
		dk := newTestDynaKube(false)

		meta.SetStatusCondition(dk.Conditions(), metav1.Condition{Type: kubemonapi.KubeMonAvailableConditionType, Status: metav1.ConditionTrue, Reason: reasonAvailable})
		connInfoReconciler.EXPECT().Reconcile(mock.Anything, mock.Anything, dk).Return(nil).Once()
		statefulSetReconciler.EXPECT().Reconcile(mock.Anything, dk).Return(nil).Once()

		err := reconciler.Reconcile(t.Context(), dk, nil, nil)
		require.NoError(t, err)
		assert.Nil(t, meta.FindStatusCondition(*dk.Conditions(), kubemonapi.KubeMonAvailableConditionType))
	})
}

// TestReconcileConditionMapping maps each sub-reconciler outcome to a condition: nil → Available,
// rollout sentinel → Reconciling, persistent sentinel → Error.
func TestReconcileConditionMapping(t *testing.T) {
	t.Setenv(k8senv.KubemonEnableOperand, "true") // remove with gate
	tests := map[string]struct {
		connInfoErr    error
		statefulSetErr error
		wantStatus     metav1.ConditionStatus
		wantReason     string
	}{
		"both succeed -> available": {
			wantStatus: metav1.ConditionTrue,
			wantReason: reasonAvailable,
		},
		"connection info error -> reconciling": {
			connInfoErr: errors.New("no endpoints yet"),
			wantStatus:  metav1.ConditionUnknown,
			wantReason:  reasonReconciling,
		},
		"rollout in progress -> reconciling": {
			statefulSetErr: k8sstatefulset.ErrRolloutInProgress,
			wantStatus:     metav1.ConditionUnknown,
			wantReason:     reasonReconciling,
		},
		"persistent error -> error": {
			statefulSetErr: kubemonstatefulset.ErrImageRequired,
			wantStatus:     metav1.ConditionFalse,
			wantReason:     reasonError,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			connInfoReconciler := newMockConnectionInfoReconciler(t)
			statefulSetReconciler := newMockStatefulsetReconciler(t)
			reconciler := &Reconciler{
				connectionInfoReconciler: connInfoReconciler,
				statefulsetReconciler:    statefulSetReconciler,
			}
			dk := newTestDynaKube(true)

			connInfoReconciler.EXPECT().Reconcile(mock.Anything, mock.Anything, dk).Return(test.connInfoErr).Once()
			statefulSetReconciler.EXPECT().Reconcile(mock.Anything, dk).Maybe().Return(test.statefulSetErr)

			_ = reconciler.Reconcile(t.Context(), dk, nil, nil)

			assertCondition(t, dk, test.wantStatus, test.wantReason)
		})
	}
}

func assertCondition(t *testing.T, dk *dynakube.DynaKube, wantStatus metav1.ConditionStatus, wantReason string) {
	t.Helper()

	condition := meta.FindStatusCondition(*dk.Conditions(), kubemonapi.KubeMonAvailableConditionType)
	require.NotNil(t, condition)
	assert.Equal(t, wantStatus, condition.Status)
	assert.Equal(t, wantReason, condition.Reason)
}

func newTestDynaKube(enabled bool) *dynakube.DynaKube {
	dk := &dynakube.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-dk",
			Namespace: "dynatrace",
		},
		Spec: dynakube.DynaKubeSpec{
			APIURL: "https://tenant.live.dynatrace.com/api",
		},
	}

	if enabled {
		dk.Spec.KubernetesMonitoring = &kubemonapi.Spec{StatefulSetProperties: kubemonapi.StatefulSetProperties{Image: "registry.example.com/linux/activegate:1.2.3"}}
	}

	return dk
}
