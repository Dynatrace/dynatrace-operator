package dynakube

import (
	"context"
	"errors"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/exp"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/activegate"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/apimonitoring"
	controllermock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/controllers"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestReconcileActiveGate(t *testing.T) {
	ctx := context.Background()
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
		mockActiveGateReconciler.On("Reconcile", mock.Anything, mock.Anything).Return(nil)

		controller := &Controller{
			client:                      fakeClient,
			apiReader:                   fakeClient,
			activeGateReconcilerBuilder: createActivegateReconcilerBuilder(mockActiveGateReconciler),
		}

		err := controller.reconcileActiveGate(ctx, dk, nil, nil)
		require.NoError(t, err)
	})
	t.Run("no active-gate configured => active-gate reconcile returns error => returns error", func(t *testing.T) {
		dk := dkBase.DeepCopy()
		dk.Spec.ActiveGate = activegate.Spec{}

		fakeClient := fake.NewClientWithIndex(dk)

		mockActiveGateReconciler := controllermock.NewReconciler(t)
		mockActiveGateReconciler.On("Reconcile", mock.Anything, mock.Anything).Return(errors.New("BOOM"))

		controller := &Controller{
			client:                      fakeClient,
			apiReader:                   fakeClient,
			activeGateReconcilerBuilder: createActivegateReconcilerBuilder(mockActiveGateReconciler),
		}

		err := controller.reconcileActiveGate(ctx, dk, nil, nil)
		require.Error(t, err)
		require.Equal(t, "failed to reconcile ActiveGate: BOOM", err.Error())
	})
	t.Run(`reconcile disabled automatic kubernetes api monitoring`, func(t *testing.T) {
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
				KubeSystemUUID:        testUID,
				KubernetesClusterMEID: testMEID,
			},
		}

		fakeClient := fake.NewClientWithIndex(dk)
		mockActiveGateReconciler := controllermock.NewReconciler(t)
		mockActiveGateReconciler.On("Reconcile", mock.Anything, mock.Anything).Return(nil)

		mockAPIMonitoringReconciler := controllermock.NewReconciler(t)
		mockAPIMonitoringReconciler.On("Reconcile", mock.Anything, mock.Anything).Return(nil).Maybe()

		controller := &Controller{
			client:                         fakeClient,
			apiReader:                      fakeClient,
			activeGateReconcilerBuilder:    createActivegateReconcilerBuilder(mockActiveGateReconciler),
			apiMonitoringReconcilerBuilder: createAPIMonitoringReconcilerBuilder(mockAPIMonitoringReconciler),
		}

		mockClient := createDTMockClient(t, dtclient.TokenScopes{}, dtclient.TokenScopes{})
		err := controller.reconcileActiveGate(ctx, dk, mockClient, nil)
		require.NoError(t, err)

		mockAPIMonitoringReconciler.AssertNotCalled(t, "Reconcile", mock.Anything)
	})
	t.Run(`reconcile automatic kubernetes api monitoring`, func(t *testing.T) {
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
				KubeSystemUUID:        testUID,
				KubernetesClusterMEID: testMEID,
			},
		}
		fakeClient := fake.NewClientWithIndex(dk)

		mockClient := createDTMockClient(t, dtclient.TokenScopes{}, dtclient.TokenScopes{})

		mockActiveGateReconciler := controllermock.NewReconciler(t)
		mockActiveGateReconciler.On("Reconcile", mock.Anything, mock.Anything).Return(nil)

		mockAPIMonitoringReconciler := controllermock.NewReconciler(t)
		mockAPIMonitoringReconciler.On("Reconcile", mock.Anything, mock.Anything).Return(nil)
		controller := &Controller{
			client:                         fakeClient,
			apiReader:                      fakeClient,
			activeGateReconcilerBuilder:    createActivegateReconcilerBuilder(mockActiveGateReconciler),
			apiMonitoringReconcilerBuilder: createAPIMonitoringReconcilerBuilder(mockAPIMonitoringReconciler),
		}

		err := controller.reconcileActiveGate(ctx, dk, mockClient, nil)
		require.NoError(t, err)

		mockAPIMonitoringReconciler.AssertCalled(t, "Reconcile", mock.Anything)
		require.NoError(t, err)
	})
	t.Run(`reconcile automatic kubernetes api monitoring with custom cluster name`, func(t *testing.T) {
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
				Conditions: []metav1.Condition{
					{
						Type:   dtclient.ConditionTypeAPITokenSettingsRead,
						Status: metav1.ConditionTrue,
					},
				},
			},
		}

		fakeClient := fake.NewClientWithIndex(dk)

		mockClient := createDTMockClient(t, dtclient.TokenScopes{}, dtclient.TokenScopes{dtclient.ConditionTypeAPITokenSettingsRead})
		mockClient.On("CreateOrUpdateKubernetesSetting",
			mock.AnythingOfType("context.backgroundCtx"),
			mock.AnythingOfType("string"),
			mock.AnythingOfType("string"),
			mock.AnythingOfType("string")).Return(testUID, nil)

		mockActiveGateReconciler := controllermock.NewReconciler(t)
		mockActiveGateReconciler.On("Reconcile", mock.Anything, mock.Anything).Return(nil)

		controller := &Controller{
			client:                         fakeClient,
			apiReader:                      fakeClient,
			activeGateReconcilerBuilder:    createActivegateReconcilerBuilder(mockActiveGateReconciler),
			apiMonitoringReconcilerBuilder: apimonitoring.NewReconciler,
		}

		err := controller.reconcileActiveGate(ctx, dk, mockClient, nil)
		require.NoError(t, err)
		mockClient.AssertCalled(t, "CreateOrUpdateKubernetesSetting",
			mock.AnythingOfType("context.backgroundCtx"),
			clusterLabel,
			testUID,
			mock.AnythingOfType("string"))
		require.NoError(t, err)
	})
}
