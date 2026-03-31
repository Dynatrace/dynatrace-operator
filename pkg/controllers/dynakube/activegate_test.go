package dynakube

import (
	"errors"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/activegate"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestReconcileActiveGate(t *testing.T) {
	dkBase := &dynakube.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "this-is-a-name",
			Namespace: "dynatrace",
		},
		Spec: dynakube.DynaKubeSpec{
			ActiveGate: activegate.Spec{Capabilities: []activegate.CapabilityDisplayName{activegate.KubeMonCapability.DisplayName}},
		},
	}

	t.Run("no active-gate configured => nothing happens (only call active-gate reconciler)", func(t *testing.T) {
		dk := dkBase.DeepCopy()
		dk.Spec.ActiveGate = activegate.Spec{}

		fakeClient := fake.NewClientWithIndex(dk)

		mockActiveGateReconciler := newMockActiveGateReconciler(t)
		mockActiveGateReconciler.EXPECT().Reconcile(anyCtx, anyDynaKube, mock.Anything, mock.Anything).Return(nil).Once()

		controller := &Controller{
			client:               fakeClient,
			apiReader:            fakeClient,
			activeGateReconciler: mockActiveGateReconciler,
		}

		err := controller.reconcileActiveGate(t.Context(), dk, nil)
		require.NoError(t, err)
	})
	t.Run("active-gate reconcile returns error => returns error", func(t *testing.T) {
		dk := dkBase.DeepCopy()
		dk.Spec.ActiveGate = activegate.Spec{}

		fakeClient := fake.NewClientWithIndex(dk)

		mockActiveGateReconciler := newMockActiveGateReconciler(t)
		mockActiveGateReconciler.EXPECT().Reconcile(anyCtx, anyDynaKube, mock.Anything, mock.Anything).Return(errors.New("BOOM")).Once()

		controller := &Controller{
			client:               fakeClient,
			apiReader:            fakeClient,
			activeGateReconciler: mockActiveGateReconciler,
		}

		err := controller.reconcileActiveGate(t.Context(), dk, nil)
		require.Error(t, err)
		require.Equal(t, "failed to reconcile ActiveGate: BOOM", err.Error())
	})
}
