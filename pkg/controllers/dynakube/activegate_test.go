package dynakube

import (
	"context"
	"errors"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/apimonitoring"
	controllermock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/controllers"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestReconcileActiveGate(t *testing.T) {
	ctx := context.Background()
	dynakubeBase := &dynakube.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "this-is-a-name",
			Namespace: "dynatrace",
		},
		Spec: dynakube.DynaKubeSpec{
			ActiveGate: dynakube.ActiveGateSpec{Capabilities: []dynakube.CapabilityDisplayName{dynakube.KubeMonCapability.DisplayName}},
		},
	}

	t.Run("no active-gate configured => nothing happens (only call active-gate reconciler)", func(t *testing.T) {
		dk := dynakubeBase.DeepCopy()
		dk.Spec.ActiveGate = dynakube.ActiveGateSpec{}

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
		dk := dynakubeBase.DeepCopy()
		dk.Spec.ActiveGate = dynakube.ActiveGateSpec{}

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
	t.Run(`reconcile automatic kubernetes api monitoring`, func(t *testing.T) {
		instance := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testName,
				Namespace: testNamespace,
				Annotations: map[string]string{
					dynakube.AnnotationFeatureAutomaticK8sApiMonitoring: "true",
				},
			},
			Spec: dynakube.DynaKubeSpec{
				APIURL: testApiUrl,
				ActiveGate: dynakube.ActiveGateSpec{
					Capabilities: []dynakube.CapabilityDisplayName{
						dynakube.KubeMonCapability.DisplayName,
					},
				},
			},
			Status: dynakube.DynaKubeStatus{
				KubeSystemUUID: testUID,
			},
		}
		fakeClient := fake.NewClientWithIndex(instance)

		mockClient := createDTMockClient(t, dtclient.TokenScopes{}, dtclient.TokenScopes{})

		mockActiveGateReconciler := controllermock.NewReconciler(t)
		mockActiveGateReconciler.On("Reconcile", mock.Anything, mock.Anything).Return(nil)

		controller := &Controller{
			client:                         fakeClient,
			apiReader:                      fakeClient,
			activeGateReconcilerBuilder:    createActivegateReconcilerBuilder(mockActiveGateReconciler),
			apiMonitoringReconcilerBuilder: apimonitoring.NewReconciler, // TODO: actually mock it
		}

		err := controller.reconcileActiveGate(ctx, instance, mockClient, nil)
		require.NoError(t, err)
		mockClient.AssertCalled(t, "CreateOrUpdateKubernetesSetting",
			mock.AnythingOfType("context.backgroundCtx"),
			testName,
			testUID,
			mock.AnythingOfType("string"))
		require.NoError(t, err)
	})
	t.Run(`reconcile automatic kubernetes api monitoring with custom cluster name`, func(t *testing.T) {
		const clusterLabel = "..blabla..;.🙃"

		instance := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testName,
				Namespace: testNamespace,
				Annotations: map[string]string{
					dynakube.AnnotationFeatureAutomaticK8sApiMonitoring:            "true",
					dynakube.AnnotationFeatureAutomaticK8sApiMonitoringClusterName: clusterLabel,
				},
			},
			Spec: dynakube.DynaKubeSpec{
				APIURL: testApiUrl,
				ActiveGate: dynakube.ActiveGateSpec{
					Capabilities: []dynakube.CapabilityDisplayName{
						dynakube.KubeMonCapability.DisplayName,
					},
				},
			},
			Status: dynakube.DynaKubeStatus{
				KubeSystemUUID: testUID,
			},
		}
		fakeClient := fake.NewClientWithIndex(instance)

		mockClient := createDTMockClient(t, dtclient.TokenScopes{}, dtclient.TokenScopes{})
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
			apiMonitoringReconcilerBuilder: apimonitoring.NewReconciler, // TODO: actually mock it
		}

		err := controller.reconcileActiveGate(ctx, instance, mockClient, nil)
		require.NoError(t, err)
		mockClient.AssertCalled(t, "CreateOrUpdateKubernetesSetting",
			mock.AnythingOfType("context.backgroundCtx"),
			clusterLabel,
			testUID,
			mock.AnythingOfType("string"))
		require.NoError(t, err)
	})
}
