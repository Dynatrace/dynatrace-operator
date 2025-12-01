package dynakube

import (
	"errors"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/exp"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/activegate"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	controllermock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/controllers"
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

		mockActiveGateReconciler := controllermock.NewReconciler(t)
		mockActiveGateReconciler.EXPECT().Reconcile(mockCtx).Return(nil).Once()

		controller := &Controller{
			client:                      fakeClient,
			apiReader:                   fakeClient,
			activeGateReconcilerBuilder: createActivegateReconcilerBuilder(mockActiveGateReconciler),
		}

		err := controller.reconcileActiveGate(t.Context(), dk, nil, nil)
		require.NoError(t, err)
	})
	t.Run("no active-gate configured => active-gate reconcile returns error => returns error", func(t *testing.T) {
		dk := dkBase.DeepCopy()
		dk.Spec.ActiveGate = activegate.Spec{}

		fakeClient := fake.NewClientWithIndex(dk)

		mockActiveGateReconciler := controllermock.NewReconciler(t)
		mockActiveGateReconciler.EXPECT().Reconcile(mockCtx).Return(errors.New("BOOM")).Once()

		controller := &Controller{
			client:                      fakeClient,
			apiReader:                   fakeClient,
			activeGateReconcilerBuilder: createActivegateReconcilerBuilder(mockActiveGateReconciler),
		}

		err := controller.reconcileActiveGate(t.Context(), dk, nil, nil)
		require.Error(t, err)
		require.Equal(t, "failed to reconcile ActiveGate: BOOM", err.Error())
	})
	t.Run("reconcile disabled automatic kubernetes api monitoring", func(t *testing.T) {
		dk := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testName,
				Namespace: testNamespace,
				Annotations: map[string]string{
					exp.AGAutomaticK8sAPIMonitoringKey: "false",
				},
			},
			Spec: dynakube.DynaKubeSpec{
				APIURL: testAPIURL,
				ActiveGate: activegate.Spec{
					Capabilities: []activegate.CapabilityDisplayName{
						activegate.KubeMonCapability.DisplayName,
					},
				},
			},
			Status: dynakube.DynaKubeStatus{
				KubeSystemUUID: testUID,
			},
		}

		mockActiveGateReconciler := controllermock.NewReconciler(t)
		mockActiveGateReconciler.EXPECT().Reconcile(mockCtx).Return(nil).Once()

		mockAPIMonitoringReconciler := controllermock.NewReconciler(t)

		controller := &Controller{
			activeGateReconcilerBuilder:    createActivegateReconcilerBuilder(mockActiveGateReconciler),
			apiMonitoringReconcilerBuilder: createAPIMonitoringReconcilerBuilder(mockAPIMonitoringReconciler),
		}

		err := controller.reconcileActiveGate(t.Context(), dk, nil, nil)
		require.NoError(t, err)
	})
	t.Run("reconcile automatic kubernetes api monitoring", func(t *testing.T) {
		dk := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testName,
				Namespace: testNamespace,
				Annotations: map[string]string{
					exp.AGAutomaticK8sAPIMonitoringKey: "true",
				},
			},
			Spec: dynakube.DynaKubeSpec{
				APIURL: testAPIURL,
				ActiveGate: activegate.Spec{
					Capabilities: []activegate.CapabilityDisplayName{
						activegate.KubeMonCapability.DisplayName,
					},
				},
			},
			Status: dynakube.DynaKubeStatus{
				KubeSystemUUID: testUID,
			},
		}

		mockActiveGateReconciler := controllermock.NewReconciler(t)
		mockActiveGateReconciler.EXPECT().Reconcile(mockCtx).Return(nil).Once()

		mockAPIMonitoringReconciler := controllermock.NewReconciler(t)
		mockAPIMonitoringReconciler.EXPECT().Reconcile(mockCtx).Return(nil).Once()

		controller := &Controller{
			activeGateReconcilerBuilder:    createActivegateReconcilerBuilder(mockActiveGateReconciler),
			apiMonitoringReconcilerBuilder: createAPIMonitoringReconcilerBuilder(mockAPIMonitoringReconciler),
		}

		err := controller.reconcileActiveGate(t.Context(), dk, nil, nil)
		require.NoError(t, err)
	})
	t.Run("reconcile automatic kubernetes api monitoring with custom cluster name", func(t *testing.T) {
		const clusterLabel = "..blabla..;.🙃"

		dk := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testName,
				Namespace: testNamespace,
				Annotations: map[string]string{
					exp.AGAutomaticK8sAPIMonitoringKey:            "true",
					exp.AGAutomaticK8sAPIMonitoringClusterNameKey: clusterLabel,
				},
			},
			Spec: dynakube.DynaKubeSpec{
				APIURL: testAPIURL,
				ActiveGate: activegate.Spec{
					Capabilities: []activegate.CapabilityDisplayName{
						activegate.KubeMonCapability.DisplayName,
					},
				},
			},
			Status: dynakube.DynaKubeStatus{
				KubeSystemUUID: testUID,
			},
		}

		mockActiveGateReconciler := controllermock.NewReconciler(t)
		mockActiveGateReconciler.EXPECT().Reconcile(mockCtx).Return(nil).Once()

		mockAPIMonitoringReconciler := controllermock.NewReconciler(t)
		mockAPIMonitoringReconciler.EXPECT().Reconcile(mockCtx).Return(nil).Once()

		controller := &Controller{
			activeGateReconcilerBuilder:    createActivegateReconcilerBuilder(mockActiveGateReconciler),
			apiMonitoringReconcilerBuilder: createAPIMonitoringReconcilerBuilder(mockAPIMonitoringReconciler),
		}

		err := controller.reconcileActiveGate(t.Context(), dk, nil, nil)
		require.NoError(t, err)
	})
}
