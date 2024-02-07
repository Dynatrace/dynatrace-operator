package dynakube

import (
	"context"
	"errors"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	mockcontroller "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/controllers"
	mockconnectioninfo "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/controllers/dynakube/connectioninfo"
	mockversion "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/controllers/dynakube/version"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestReconcileActiveGate(t *testing.T) {
	ctx := context.Background()
	dynakubeBase := &dynatracev1beta1.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "this-is-a-name",
			Namespace: "dynatrace",
		},
		Spec: dynatracev1beta1.DynaKubeSpec{
			ActiveGate: dynatracev1beta1.ActiveGateSpec{Capabilities: []dynatracev1beta1.CapabilityDisplayName{dynatracev1beta1.KubeMonCapability.DisplayName}},
		},
	}

	t.Run("no active-gate configured => nothing happens (only call active-gate reconciler)", func(t *testing.T) {
		dynakube := dynakubeBase.DeepCopy()
		dynakube.Spec.ActiveGate = dynatracev1beta1.ActiveGateSpec{}

		fakeClient := fake.NewClientWithIndex(dynakube)

		mockActiveGateReconciler := mockcontroller.NewReconciler(t)
		mockActiveGateReconciler.On("Reconcile", mock.Anything, mock.Anything).Return(nil)

		controller := &Controller{
			client:                      fakeClient,
			apiReader:                   fakeClient,
			activeGateReconcilerBuilder: createActivegateReconcilerBuilder(mockActiveGateReconciler),
		}

		err := controller.reconcileActiveGate(ctx, dynakube, nil, nil, nil, nil)
		require.NoError(t, err)
	})
	t.Run("no active-gate configured => active-gate reconcile returns error => returns error", func(t *testing.T) {
		dynakube := dynakubeBase.DeepCopy()
		dynakube.Spec.ActiveGate = dynatracev1beta1.ActiveGateSpec{}

		fakeClient := fake.NewClientWithIndex(dynakube)

		mockActiveGateReconciler := mockcontroller.NewReconciler(t)
		mockActiveGateReconciler.On("Reconcile", mock.Anything, mock.Anything).Return(errors.New("BOOM"))

		controller := &Controller{
			client:                      fakeClient,
			apiReader:                   fakeClient,
			activeGateReconcilerBuilder: createActivegateReconcilerBuilder(mockActiveGateReconciler),
		}

		err := controller.reconcileActiveGate(ctx, dynakube, nil, nil, nil, nil)
		require.Error(t, err)
		require.Equal(t, "failed to reconcile ActiveGate: BOOM", err.Error())
	})
	t.Run(`reconcile automatic kubernetes api monitoring`, func(t *testing.T) {
		instance := &dynatracev1beta1.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testName,
				Namespace: testNamespace,
				Annotations: map[string]string{
					dynatracev1beta1.AnnotationFeatureAutomaticK8sApiMonitoring: "true",
				},
			},
			Spec: dynatracev1beta1.DynaKubeSpec{
				APIURL: testApiUrl,
				ActiveGate: dynatracev1beta1.ActiveGateSpec{
					Capabilities: []dynatracev1beta1.CapabilityDisplayName{
						dynatracev1beta1.KubeMonCapability.DisplayName,
					},
				},
			},
			Status: dynatracev1beta1.DynaKubeStatus{
				KubeSystemUUID: testUID,
			},
		}
		fakeClient := fake.NewClientWithIndex(instance)

		mockClient := createDTMockClient(t, dtclient.TokenScopes{}, dtclient.TokenScopes{})

		mockConnectionInfoReconciler := mockconnectioninfo.NewReconciler(t)
		mockConnectionInfoReconciler.On("ReconcileActiveGate", mock.Anything, mock.Anything).Return(nil)

		mockVersionReconciler := mockversion.NewReconciler(t)
		mockVersionReconciler.On("ReconcileActiveGate", mock.Anything, mock.Anything).Return(nil)

		mockActiveGateReconciler := mockcontroller.NewReconciler(t)
		mockActiveGateReconciler.On("Reconcile", mock.Anything, mock.Anything).Return(nil)

		controller := &Controller{
			client:                      fakeClient,
			apiReader:                   fakeClient,
			activeGateReconcilerBuilder: createActivegateReconcilerBuilder(mockActiveGateReconciler),
		}

		err := controller.reconcileActiveGate(ctx, instance, mockClient, nil, mockConnectionInfoReconciler, mockVersionReconciler)
		require.NoError(t, err)
		mockClient.AssertCalled(t, "CreateOrUpdateKubernetesSetting",
			testName,
			testUID,
			mock.AnythingOfType("string"))
		require.NoError(t, err)
	})
	t.Run(`reconcile automatic kubernetes api monitoring with custom cluster name`, func(t *testing.T) {
		const clusterLabel = "..blabla..;.🙃"

		instance := &dynatracev1beta1.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testName,
				Namespace: testNamespace,
				Annotations: map[string]string{
					dynatracev1beta1.AnnotationFeatureAutomaticK8sApiMonitoring:            "true",
					dynatracev1beta1.AnnotationFeatureAutomaticK8sApiMonitoringClusterName: clusterLabel,
				},
			},
			Spec: dynatracev1beta1.DynaKubeSpec{
				APIURL: testApiUrl,
				ActiveGate: dynatracev1beta1.ActiveGateSpec{
					Capabilities: []dynatracev1beta1.CapabilityDisplayName{
						dynatracev1beta1.KubeMonCapability.DisplayName,
					},
				},
			},
			Status: dynatracev1beta1.DynaKubeStatus{
				KubeSystemUUID: testUID,
			},
		}
		fakeClient := fake.NewClientWithIndex(instance)

		mockClient := createDTMockClient(t, dtclient.TokenScopes{}, dtclient.TokenScopes{})
		mockClient.On("CreateOrUpdateKubernetesSetting",
			mock.AnythingOfType("string"),
			mock.AnythingOfType("string"),
			mock.AnythingOfType("string")).Return(testUID, nil)

		mockConnectionInfoReconciler := mockconnectioninfo.NewReconciler(t)
		mockConnectionInfoReconciler.On("ReconcileActiveGate", mock.Anything, mock.Anything).Return(nil)

		mockVersionReconciler := mockversion.NewReconciler(t)
		mockVersionReconciler.On("ReconcileActiveGate", mock.Anything, mock.Anything).Return(nil)

		mockActiveGateReconciler := mockcontroller.NewReconciler(t)
		mockActiveGateReconciler.On("Reconcile", mock.Anything, mock.Anything).Return(nil)

		controller := &Controller{
			client:                      fakeClient,
			apiReader:                   fakeClient,
			activeGateReconcilerBuilder: createActivegateReconcilerBuilder(mockActiveGateReconciler),
		}

		err := controller.reconcileActiveGate(ctx, instance, mockClient, nil, mockConnectionInfoReconciler, mockVersionReconciler)
		require.NoError(t, err)
		mockClient.AssertCalled(t, "CreateOrUpdateKubernetesSetting",
			clusterLabel,
			testUID,
			mock.AnythingOfType("string"))
		require.NoError(t, err)
	})
}
