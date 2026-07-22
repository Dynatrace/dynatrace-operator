// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package kubemon

import (
	"errors"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	kubemonapi "github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/kubemon"
	kubemonconnectioninfo "github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/kubemon/connectioninfo"
	kubemonstatefulset "github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/kubemon/statefulset"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8senv"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/objects/k8sstatefulset"
	pkgerrors "github.com/pkg/errors"
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
		authTokenReconciler := newMockAuthTokenReconciler(t)
		statefulSetReconciler := newMockStatefulsetReconciler(t)
		reconciler := &Reconciler{
			connectionInfoReconciler: connInfoReconciler,
			authTokenReconciler:      authTokenReconciler,
			statefulsetReconciler:    statefulSetReconciler,
		}
		dk := newTestDynaKube(false)

		meta.SetStatusCondition(dk.Conditions(), metav1.Condition{Type: kubemonapi.KubeMonAvailableConditionType, Status: metav1.ConditionTrue, Reason: reasonAvailable})
		connInfoReconciler.EXPECT().Reconcile(mock.Anything, mock.Anything, dk).Return(nil).Once()
		authTokenReconciler.EXPECT().Reconcile(mock.Anything, mock.Anything, dk).Return(nil).Once()
		statefulSetReconciler.EXPECT().Reconcile(mock.Anything, dk).Return(nil).Once()

		err := reconciler.Reconcile(t.Context(), dk, nil, nil)
		require.NoError(t, err)
		assert.Nil(t, meta.FindStatusCondition(*dk.Conditions(), kubemonapi.KubeMonAvailableConditionType))
	})
}

// TestReconcileConditionMapping maps each sub-reconciler outcome to the resulting condition
// (status/reason/message) and asserts the error is propagated. Only the "coming up" sentinels
// (rollout, connection info) map to Reconciling; any other error surfaces as Error with the
// root-cause message.
func TestReconcileConditionMapping(t *testing.T) {
	t.Setenv(k8senv.KubemonEnableOperand, "true") // remove with gate
	tests := []struct {
		name           string
		connInfoErr    error
		authTokenErr   error
		statefulSetErr error
		wantStatus     metav1.ConditionStatus
		wantReason     string
		wantMessage    string
	}{
		{
			name:        "all succeed -> available",
			wantStatus:  metav1.ConditionTrue,
			wantReason:  reasonAvailable,
			wantMessage: messageAvailable,
		},
		{
			name:        "connection info not ready -> reconciling",
			connInfoErr: kubemonconnectioninfo.ErrConnectionInfoNotReady,
			wantStatus:  metav1.ConditionFalse,
			wantReason:  reasonReconciling,
			wantMessage: kubemonconnectioninfo.ErrConnectionInfoNotReady.Error(),
		},
		{
			name:         "auth token error -> error",
			authTokenErr: errors.New("api error"),
			wantStatus:   metav1.ConditionFalse,
			wantReason:   reasonError,
			wantMessage:  "api error",
		},
		{
			name:           "rollout in progress -> reconciling",
			statefulSetErr: k8sstatefulset.ErrRolloutInProgress,
			wantStatus:     metav1.ConditionFalse,
			wantReason:     reasonReconciling,
			wantMessage:    k8sstatefulset.ErrRolloutInProgress.Error(),
		},
		{
			name:           "unexpected error -> error",
			statefulSetErr: errors.New("boom"),
			wantStatus:     metav1.ConditionFalse,
			wantReason:     reasonError,
			wantMessage:    "boom",
		},
		{
			name:           "stack-wrapped error -> error without stack trace in message",
			statefulSetErr: pkgerrors.WithStack(kubemonstatefulset.ErrImageRequired),
			wantStatus:     metav1.ConditionFalse,
			wantReason:     reasonError,
			wantMessage:    kubemonstatefulset.ErrImageRequired.Error(),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			connInfoReconciler := newMockConnectionInfoReconciler(t)
			authTokenReconciler := newMockAuthTokenReconciler(t)
			statefulSetReconciler := newMockStatefulsetReconciler(t)
			reconciler := &Reconciler{
				connectionInfoReconciler: connInfoReconciler,
				authTokenReconciler:      authTokenReconciler,
				statefulsetReconciler:    statefulSetReconciler,
			}
			dk := newTestDynaKube(true)

			connInfoReconciler.EXPECT().Reconcile(mock.Anything, mock.Anything, dk).Return(test.connInfoErr).Once()
			if test.connInfoErr == nil {
				authTokenReconciler.EXPECT().Reconcile(mock.Anything, mock.Anything, dk).Return(test.authTokenErr).Once()
			}
			if test.connInfoErr == nil && test.authTokenErr == nil {
				statefulSetReconciler.EXPECT().Reconcile(mock.Anything, dk).Return(test.statefulSetErr).Once()
			}

			err := reconciler.Reconcile(t.Context(), dk, nil, nil)

			wantErr := test.connInfoErr
			if wantErr == nil {
				wantErr = test.authTokenErr
			}
			if wantErr == nil {
				wantErr = test.statefulSetErr
			}
			require.ErrorIs(t, err, wantErr)

			condition := meta.FindStatusCondition(*dk.Conditions(), kubemonapi.KubeMonAvailableConditionType)
			require.NotNil(t, condition)
			assert.Equal(t, test.wantStatus, condition.Status)
			assert.Equal(t, test.wantReason, condition.Reason)
			assert.Equal(t, test.wantMessage, condition.Message)
		})
	}
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
