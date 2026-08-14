// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package kubemon

import (
	"errors"
	"fmt"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/activegate"
	kubemonapi "github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/kubemon"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	kubemonconnectioninfo "github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/kubemon/connectioninfo"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/token"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8senv"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/objects/k8sstatefulset"
	agclientmock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/clients/dynatrace/activegate"
	imageclientmock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/clients/dynatrace/image"
	versionclientmock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/clients/dynatrace/version"
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
	t.Setenv(k8senv.ExperimentalEnableKubemonOperand, "true") // remove with gate
	t.Run("removes condition when disabled and cleanup succeeds", func(t *testing.T) {
		connInfoReconciler := newMockConnectionInfoReconciler(t)
		authTokenReconciler := newMockAuthTokenReconciler(t)
		statefulSetReconciler := newMockStatefulsetReconciler(t)
		pullSecretReconciler := newMockPullSecretReconciler(t)
		customPropertiesReconciler := newMockCustomPropertiesReconciler(t)
		istioRec := newMockIstioReconciler(t)
		reconciler := &Reconciler{
			connectionInfoReconciler:   connInfoReconciler,
			authTokenReconciler:        authTokenReconciler,
			statefulsetReconciler:      statefulSetReconciler,
			pullSecretReconciler:       pullSecretReconciler,
			customPropertiesReconciler: customPropertiesReconciler,
			istioReconciler:            istioRec,
		}
		dk := newTestDynaKube(false)

		meta.SetStatusCondition(dk.Conditions(), metav1.Condition{Type: kubemonapi.KubeMonAvailableConditionType, Status: metav1.ConditionTrue, Reason: reasonAvailable})
		connInfoReconciler.EXPECT().Reconcile(mock.Anything, mock.Anything, dk).Return(nil).Once()
		istioRec.EXPECT().ReconcileActiveGate(mock.Anything, dk).Return(nil).Once()
		authTokenReconciler.EXPECT().Reconcile(mock.Anything, mock.Anything, dk).Return(nil).Once()
		pullSecretReconciler.EXPECT().Reconcile(mock.Anything, dk, mock.Anything).Return(nil).Once()
		customPropertiesReconciler.EXPECT().Reconcile(mock.Anything, dk).Return(nil).Once()
		statefulSetReconciler.EXPECT().Reconcile(mock.Anything, dk, mock.Anything, mock.Anything).Return(nil).Once()

		err := reconciler.Reconcile(t.Context(), dk, newTestDTClient(t), token.Tokens(nil))
		require.NoError(t, err)
		assert.Nil(t, meta.FindStatusCondition(*dk.Conditions(), kubemonapi.KubeMonAvailableConditionType))
	})
}

// TestReconcileConditionMapping maps each sub-reconciler outcome to the resulting condition
// (status/reason/message) and asserts the error is propagated. Only the "coming up" sentinels
// (rollout, connection info) map to Reconciling; any other error surfaces as Error with the
// root-cause message.
func TestReconcileConditionMapping(t *testing.T) {
	t.Setenv(k8senv.ExperimentalEnableKubemonOperand, "true") // remove with gate

	type reconcilerMocks struct {
		reconciler       *Reconciler
		connInfo         *mockConnectionInfoReconciler
		authToken        *mockAuthTokenReconciler
		pullSecret       *mockPullSecretReconciler
		customProperties *mockCustomPropertiesReconciler
		statefulSet      *mockStatefulsetReconciler
		istio            *mockIstioReconciler
	}

	newMocks := func(t *testing.T) reconcilerMocks {
		t.Helper()
		m := reconcilerMocks{
			connInfo:         newMockConnectionInfoReconciler(t),
			authToken:        newMockAuthTokenReconciler(t),
			pullSecret:       newMockPullSecretReconciler(t),
			customProperties: newMockCustomPropertiesReconciler(t),
			statefulSet:      newMockStatefulsetReconciler(t),
			istio:            newMockIstioReconciler(t),
		}
		m.reconciler = &Reconciler{
			connectionInfoReconciler:   m.connInfo,
			authTokenReconciler:        m.authToken,
			pullSecretReconciler:       m.pullSecret,
			customPropertiesReconciler: m.customProperties,
			statefulsetReconciler:      m.statefulSet,
			istioReconciler:            m.istio,
		}

		return m
	}

	assertCondition := func(t *testing.T, dk *dynakube.DynaKube, wantStatus metav1.ConditionStatus, wantReason, wantMessage string) {
		t.Helper()
		condition := meta.FindStatusCondition(*dk.Conditions(), kubemonapi.KubeMonAvailableConditionType)
		require.NotNil(t, condition)
		assert.Equal(t, wantStatus, condition.Status)
		assert.Equal(t, wantReason, condition.Reason)
		assert.Equal(t, wantMessage, condition.Message)
	}

	t.Run("all reconcilers succeed -> available", func(t *testing.T) {
		mocks := newMocks(t)
		dk := newTestDynaKube(true)

		mocks.connInfo.EXPECT().Reconcile(mock.Anything, mock.Anything, dk).Return(nil).Once()
		mocks.istio.EXPECT().ReconcileActiveGate(mock.Anything, dk).Return(nil).Once()
		mocks.authToken.EXPECT().Reconcile(mock.Anything, mock.Anything, dk).Return(nil).Once()
		mocks.pullSecret.EXPECT().Reconcile(mock.Anything, dk, mock.Anything).Return(nil).Once()
		mocks.customProperties.EXPECT().Reconcile(mock.Anything, dk).Return(nil).Once()
		mocks.statefulSet.EXPECT().Reconcile(mock.Anything, dk, mock.Anything, mock.Anything).Return(nil).Once()

		err := mocks.reconciler.Reconcile(t.Context(), dk, newTestDTClient(t), token.Tokens(nil))

		require.NoError(t, err)
		assertCondition(t, dk, metav1.ConditionTrue, reasonAvailable, messageAvailable)
	})
	t.Run("AG enabled: istio reconciliation skipped (handled by AG reconciler)", func(t *testing.T) {
		mocks := newMocks(t)
		dk := newTestDynaKube(true)
		dk.Spec.ActiveGate.Capabilities = []activegate.CapabilityDisplayName{activegate.RoutingCapability.DisplayName}

		mocks.connInfo.EXPECT().Reconcile(mock.Anything, mock.Anything, dk).Return(nil).Once()
		mocks.authToken.EXPECT().Reconcile(mock.Anything, mock.Anything, dk).Return(nil).Once()
		mocks.pullSecret.EXPECT().Reconcile(mock.Anything, dk, mock.Anything).Return(nil).Once()
		mocks.customProperties.EXPECT().Reconcile(mock.Anything, dk).Return(nil).Once()
		mocks.statefulSet.EXPECT().Reconcile(mock.Anything, dk, mock.Anything, mock.Anything).Return(nil).Once()

		err := mocks.reconciler.Reconcile(t.Context(), dk, newTestDTClient(t), token.Tokens(nil))

		require.NoError(t, err)
		assertCondition(t, dk, metav1.ConditionTrue, reasonAvailable, messageAvailable)
	})

	t.Run("connection info not ready -> reconciling", func(t *testing.T) {
		mocks := newMocks(t)
		dk := newTestDynaKube(true)

		mocks.connInfo.EXPECT().Reconcile(mock.Anything, mock.Anything, dk).Return(kubemonconnectioninfo.ErrConnectionInfoNotReady).Once()

		err := mocks.reconciler.Reconcile(t.Context(), dk, newTestDTClient(t), token.Tokens(nil))

		require.ErrorIs(t, err, kubemonconnectioninfo.ErrConnectionInfoNotReady)
		assertCondition(t, dk, metav1.ConditionFalse, reasonReconciling, kubemonconnectioninfo.ErrConnectionInfoNotReady.Error())
	})

	t.Run("istio error -> error", func(t *testing.T) {
		mocks := newMocks(t)
		dk := newTestDynaKube(true)
		istioErr := errors.New("istio error")

		mocks.connInfo.EXPECT().Reconcile(mock.Anything, mock.Anything, dk).Return(nil).Once()
		mocks.istio.EXPECT().ReconcileActiveGate(mock.Anything, dk).Return(istioErr).Once()

		err := mocks.reconciler.Reconcile(t.Context(), dk, newTestDTClient(t), token.Tokens(nil))

		require.ErrorIs(t, err, istioErr)
		assertCondition(t, dk, metav1.ConditionFalse, reasonError, "istio error")
	})

	t.Run("auth token error -> error", func(t *testing.T) {
		mocks := newMocks(t)
		dk := newTestDynaKube(true)
		apiErr := errors.New("api error")

		mocks.connInfo.EXPECT().Reconcile(mock.Anything, mock.Anything, dk).Return(nil).Once()
		mocks.istio.EXPECT().ReconcileActiveGate(mock.Anything, dk).Return(nil).Once()
		mocks.authToken.EXPECT().Reconcile(mock.Anything, mock.Anything, dk).Return(apiErr).Once()

		err := mocks.reconciler.Reconcile(t.Context(), dk, newTestDTClient(t), token.Tokens(nil))

		require.ErrorIs(t, err, apiErr)
		assertCondition(t, dk, metav1.ConditionFalse, reasonError, "api error")
	})

	t.Run("custom properties error -> error", func(t *testing.T) {
		mocks := newMocks(t)
		dk := newTestDynaKube(true)
		cpErr := errors.New("custom properties api error")

		mocks.connInfo.EXPECT().Reconcile(mock.Anything, mock.Anything, dk).Return(nil).Once()
		mocks.istio.EXPECT().ReconcileActiveGate(mock.Anything, dk).Return(nil).Once()
		mocks.authToken.EXPECT().Reconcile(mock.Anything, mock.Anything, dk).Return(nil).Once()
		mocks.pullSecret.EXPECT().Reconcile(mock.Anything, dk, mock.Anything).Return(nil).Once()
		mocks.customProperties.EXPECT().Reconcile(mock.Anything, dk).Return(cpErr).Once()

		err := mocks.reconciler.Reconcile(t.Context(), dk, newTestDTClient(t), token.Tokens(nil))

		require.ErrorIs(t, err, cpErr)
		assertCondition(t, dk, metav1.ConditionFalse, reasonError, "custom properties api error")
	})

	t.Run("rollout in progress -> reconciling", func(t *testing.T) {
		mocks := newMocks(t)
		dk := newTestDynaKube(true)

		mocks.connInfo.EXPECT().Reconcile(mock.Anything, mock.Anything, dk).Return(nil).Once()
		mocks.istio.EXPECT().ReconcileActiveGate(mock.Anything, dk).Return(nil).Once()
		mocks.authToken.EXPECT().Reconcile(mock.Anything, mock.Anything, dk).Return(nil).Once()
		mocks.pullSecret.EXPECT().Reconcile(mock.Anything, dk, mock.Anything).Return(nil).Once()
		mocks.customProperties.EXPECT().Reconcile(mock.Anything, dk).Return(nil).Once()
		mocks.statefulSet.EXPECT().Reconcile(mock.Anything, dk, mock.Anything, mock.Anything).Return(k8sstatefulset.ErrRolloutInProgress).Once()

		err := mocks.reconciler.Reconcile(t.Context(), dk, newTestDTClient(t), token.Tokens(nil))

		require.ErrorIs(t, err, k8sstatefulset.ErrRolloutInProgress)
		assertCondition(t, dk, metav1.ConditionFalse, reasonReconciling, k8sstatefulset.ErrRolloutInProgress.Error())
	})

	t.Run("unexpected stateful set error -> error", func(t *testing.T) {
		mocks := newMocks(t)
		dk := newTestDynaKube(true)
		boomErr := errors.New("boom")

		mocks.connInfo.EXPECT().Reconcile(mock.Anything, mock.Anything, dk).Return(nil).Once()
		mocks.istio.EXPECT().ReconcileActiveGate(mock.Anything, dk).Return(nil).Once()
		mocks.authToken.EXPECT().Reconcile(mock.Anything, mock.Anything, dk).Return(nil).Once()
		mocks.pullSecret.EXPECT().Reconcile(mock.Anything, dk, mock.Anything).Return(nil).Once()
		mocks.customProperties.EXPECT().Reconcile(mock.Anything, dk).Return(nil).Once()
		mocks.statefulSet.EXPECT().Reconcile(mock.Anything, dk, mock.Anything, mock.Anything).Return(boomErr).Once()

		err := mocks.reconciler.Reconcile(t.Context(), dk, newTestDTClient(t), token.Tokens(nil))

		require.ErrorIs(t, err, boomErr)
		assertCondition(t, dk, metav1.ConditionFalse, reasonError, "boom")
	})
}

// TestIsTransientError covers the shared classifier used both by reconcileCondition and by the
// parent DynaKube controller to decide whether a kubemon error is a converging/transient state.
func TestIsTransientError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil error -> false", nil, false},
		{"rollout in progress -> true", k8sstatefulset.ErrRolloutInProgress, true},
		{"connection info not ready -> true", kubemonconnectioninfo.ErrConnectionInfoNotReady, true},
		{"wrapped rollout in progress -> true", fmt.Errorf("wrap: %w", k8sstatefulset.ErrRolloutInProgress), true},
		{"unrelated error -> false", errors.New("boom"), false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.want, IsTransientError(test.err))
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

func newTestDTClient(t *testing.T) *dynatrace.Client {
	t.Helper()

	return &dynatrace.Client{
		ActiveGate: agclientmock.NewClient(t),
		Images:     imageclientmock.NewClient(t),
		Version:    versionclientmock.NewClient(t),
	}
}
