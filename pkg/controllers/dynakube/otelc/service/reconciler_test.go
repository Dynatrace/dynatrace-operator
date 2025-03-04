package service

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube/telemetryservice"
	"github.com/Dynatrace/dynatrace-operator/pkg/otelcgen"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/conditions"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	testDynakubeName  = "dynakube"
	testNamespaceName = "dynatrace"
	testServiceName   = "test-service-name"
)

func getTestDynakube(telemetryServiceSpec *telemetryservice.Spec) *dynakube.DynaKube {
	return &dynakube.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testDynakubeName,
			Namespace: testNamespaceName,
		},
		Spec: dynakube.DynaKubeSpec{
			TelemetryService: telemetryServiceSpec,
		},
		Status: dynakube.DynaKubeStatus{},
	}
}

func TestService(t *testing.T) {
	t.Run("create service if it does not exist", func(t *testing.T) {
		mockK8sClient := fake.NewFakeClient()
		dk := getTestDynakube(&telemetryservice.Spec{})
		err := NewReconciler(mockK8sClient, mockK8sClient, dk).Reconcile(context.Background())
		require.NoError(t, err)

		service := &corev1.Service{}
		err = mockK8sClient.Get(context.Background(), client.ObjectKey{Name: dk.TelemetryService().GetName(), Namespace: dk.Namespace}, service)
		require.NoError(t, err)

		require.Len(t, service.Spec.Ports, 8)
		assert.Equal(t, portNameOtlpGrpc, service.Spec.Ports[0].Name)
		assert.Equal(t, portNameOtlpHttp, service.Spec.Ports[1].Name)
		assert.Equal(t, portNameZipkin, service.Spec.Ports[2].Name)
		assert.Equal(t, portNameJaegerGrpc, service.Spec.Ports[3].Name)
		assert.Equal(t, portNameJaegerThriftBinary, service.Spec.Ports[4].Name)
		assert.Equal(t, portNameJaegerThriftCompact, service.Spec.Ports[5].Name)
		assert.Equal(t, portNameJaegerThriftHttp, service.Spec.Ports[6].Name)
		assert.Equal(t, portNameStatsd, service.Spec.Ports[7].Name)

		require.Len(t, dk.Status.Conditions, 1)
		assert.Equal(t, serviceConditionType, dk.Status.Conditions[0].Type)
		assert.Equal(t, conditions.ServiceCreatedReason, dk.Status.Conditions[0].Reason)
		assert.Equal(t, metav1.ConditionTrue, dk.Status.Conditions[0].Status)
	})
	t.Run("remove service if it is not needed", func(t *testing.T) {
		dk := getTestDynakube(nil)
		dk.Status.Conditions = []metav1.Condition{
			{
				Type: serviceConditionType,
			},
		}

		mockK8sClient := fake.NewFakeClient()
		err := mockK8sClient.Create(context.Background(), &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      dk.TelemetryService().GetName(),
				Namespace: dk.Namespace,
			},
		})
		require.NoError(t, err)

		err = NewReconciler(mockK8sClient, mockK8sClient, dk).Reconcile(context.Background())
		require.NoError(t, err)

		service := &corev1.Service{}
		err = mockK8sClient.Get(context.Background(), client.ObjectKey{Name: dk.TelemetryService().GetName(), Namespace: dk.Namespace}, service)
		require.Error(t, err)
		assert.True(t, k8serrors.IsNotFound(err))

		require.Empty(t, dk.Status.Conditions)
	})
	t.Run("create service for specified protocols", func(t *testing.T) {
		mockK8sClient := fake.NewFakeClient()
		dk := getTestDynakube(&telemetryservice.Spec{
			Protocols: []string{
				string(otelcgen.ZipkinProtocol),
				string(otelcgen.StatsdProtocol),
			},
		})
		err := NewReconciler(mockK8sClient, mockK8sClient, dk).Reconcile(context.Background())
		require.NoError(t, err)

		service := &corev1.Service{}
		err = mockK8sClient.Get(context.Background(), client.ObjectKey{Name: dk.TelemetryService().GetName(), Namespace: dk.Namespace}, service)
		require.NoError(t, err)

		require.Len(t, service.Spec.Ports, 2)
		assert.Equal(t, portNameZipkin, service.Spec.Ports[0].Name)
		assert.Equal(t, portNameStatsd, service.Spec.Ports[1].Name)

		require.Len(t, dk.Status.Conditions, 1)
		assert.Equal(t, serviceConditionType, dk.Status.Conditions[0].Type)
		assert.Equal(t, conditions.ServiceCreatedReason, dk.Status.Conditions[0].Reason)
		assert.Equal(t, metav1.ConditionTrue, dk.Status.Conditions[0].Status)
	})
	t.Run("custom service", func(t *testing.T) {
		mockK8sClient := fake.NewFakeClient()
		dk := getTestDynakube(&telemetryservice.Spec{
			ServiceName: testServiceName,
		})
		err := NewReconciler(mockK8sClient, mockK8sClient, dk).Reconcile(context.Background())
		require.NoError(t, err)

		service := &corev1.Service{}
		err = mockK8sClient.Get(context.Background(), client.ObjectKey{Name: testServiceName, Namespace: dk.Namespace}, service)
		require.Error(t, err)
		assert.True(t, k8serrors.IsNotFound(err))

		assert.Empty(t, dk.Status.Conditions)
	})
	t.Run("update from default service to custom service", func(t *testing.T) {
		mockK8sClient := fake.NewFakeClient()
		dk := getTestDynakube(&telemetryservice.Spec{})
		err := NewReconciler(mockK8sClient, mockK8sClient, dk).Reconcile(context.Background())
		require.NoError(t, err)

		service := &corev1.Service{}
		err = mockK8sClient.Get(context.Background(), client.ObjectKey{Name: dk.TelemetryService().GetName(), Namespace: dk.Namespace}, service)
		require.NoError(t, err)

		require.Len(t, dk.Status.Conditions, 1)
		assert.Equal(t, serviceConditionType, dk.Status.Conditions[0].Type)
		assert.Equal(t, conditions.ServiceCreatedReason, dk.Status.Conditions[0].Reason)
		assert.Equal(t, metav1.ConditionTrue, dk.Status.Conditions[0].Status)

		// update
		dk.Spec.TelemetryService = &telemetryservice.Spec{
			ServiceName: testServiceName,
		}
		err = NewReconciler(mockK8sClient, mockK8sClient, dk).Reconcile(context.Background())
		require.NoError(t, err)

		service = &corev1.Service{}
		err = mockK8sClient.Get(context.Background(), client.ObjectKey{Name: dk.TelemetryService().GetName(), Namespace: dk.Namespace}, service)
		require.Error(t, err)
		assert.True(t, k8serrors.IsNotFound(err))

		assert.Empty(t, dk.Status.Conditions)
	})
}
