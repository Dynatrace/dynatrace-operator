package service

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/telemetryingest"
	"github.com/Dynatrace/dynatrace-operator/pkg/otelcgen"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/conditions"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8slabel"
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

func getTestDynakube(telemetryIngestSpec *telemetryingest.Spec) *dynakube.DynaKube {
	return &dynakube.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testDynakubeName,
			Namespace: testNamespaceName,
		},
		Spec: dynakube.DynaKubeSpec{
			TelemetryIngest: telemetryIngestSpec,
		},
		Status: dynakube.DynaKubeStatus{},
	}
}

func TestService(t *testing.T) {
	t.Run("create service if it does not exist", func(t *testing.T) {
		mockK8sClient := fake.NewFakeClient()
		dk := getTestDynakube(&telemetryingest.Spec{})
		err := NewReconciler(mockK8sClient, mockK8sClient, dk).Reconcile(t.Context())
		require.NoError(t, err)

		service := &corev1.Service{}
		err = mockK8sClient.Get(t.Context(), client.ObjectKey{Name: dk.TelemetryIngest().GetDefaultServiceName(), Namespace: dk.Namespace}, service)
		require.NoError(t, err)

		require.Len(t, service.Spec.Ports, 8)
		assert.Equal(t, otlpGrpcPortName, service.Spec.Ports[0].Name)
		assert.Equal(t, int32(4317), service.Spec.Ports[0].Port)
		assert.Equal(t, int32(4317), service.Spec.Ports[0].TargetPort.IntVal)
		assert.Equal(t, corev1.ProtocolTCP, service.Spec.Ports[0].Protocol)

		assert.Equal(t, otlpHTTPPortName, service.Spec.Ports[1].Name)
		assert.Equal(t, int32(4318), service.Spec.Ports[1].Port)
		assert.Equal(t, int32(4318), service.Spec.Ports[1].TargetPort.IntVal)
		assert.Equal(t, corev1.ProtocolTCP, service.Spec.Ports[1].Protocol)

		assert.Equal(t, jaegerGrpcPortName, service.Spec.Ports[2].Name)
		assert.Equal(t, int32(14250), service.Spec.Ports[2].Port)
		assert.Equal(t, int32(14250), service.Spec.Ports[2].TargetPort.IntVal)
		assert.Equal(t, corev1.ProtocolTCP, service.Spec.Ports[2].Protocol)

		assert.Equal(t, jaegerThriftBinaryPortName, service.Spec.Ports[3].Name)
		assert.Equal(t, int32(6832), service.Spec.Ports[3].Port)
		assert.Equal(t, int32(6832), service.Spec.Ports[3].TargetPort.IntVal)
		assert.Equal(t, corev1.ProtocolUDP, service.Spec.Ports[3].Protocol)

		assert.Equal(t, jaegerThriftCompactPortName, service.Spec.Ports[4].Name)
		assert.Equal(t, int32(6831), service.Spec.Ports[4].Port)
		assert.Equal(t, int32(6831), service.Spec.Ports[4].TargetPort.IntVal)
		assert.Equal(t, corev1.ProtocolUDP, service.Spec.Ports[4].Protocol)

		assert.Equal(t, jaegerThriftHTTPPortName, service.Spec.Ports[5].Name)
		assert.Equal(t, int32(14268), service.Spec.Ports[5].Port)
		assert.Equal(t, int32(14268), service.Spec.Ports[5].TargetPort.IntVal)
		assert.Equal(t, corev1.ProtocolTCP, service.Spec.Ports[5].Protocol)

		assert.Equal(t, statsdPortName, service.Spec.Ports[6].Name)
		assert.Equal(t, int32(8125), service.Spec.Ports[6].Port)
		assert.Equal(t, int32(8125), service.Spec.Ports[6].TargetPort.IntVal)
		assert.Equal(t, corev1.ProtocolUDP, service.Spec.Ports[6].Protocol)

		assert.Equal(t, zipkinPortName, service.Spec.Ports[7].Name)
		assert.Equal(t, int32(9411), service.Spec.Ports[7].Port)
		assert.Equal(t, int32(9411), service.Spec.Ports[7].TargetPort.IntVal)
		assert.Equal(t, corev1.ProtocolTCP, service.Spec.Ports[7].Protocol)

		require.Len(t, dk.Status.Conditions, 1)
		assert.Equal(t, serviceConditionType, dk.Status.Conditions[0].Type)
		assert.Equal(t, conditions.ServiceCreatedReason, dk.Status.Conditions[0].Reason)
		assert.Equal(t, metav1.ConditionTrue, dk.Status.Conditions[0].Status)
	})
	t.Run("create service for specified protocols", func(t *testing.T) {
		mockK8sClient := fake.NewFakeClient()
		dk := getTestDynakube(&telemetryingest.Spec{
			Protocols: []string{
				string(otelcgen.ZipkinProtocol),
				string(otelcgen.StatsdProtocol),
			},
		})
		err := NewReconciler(mockK8sClient, mockK8sClient, dk).Reconcile(t.Context())
		require.NoError(t, err)

		service := &corev1.Service{}
		err = mockK8sClient.Get(t.Context(), client.ObjectKey{Name: dk.TelemetryIngest().GetDefaultServiceName(), Namespace: dk.Namespace}, service)
		require.NoError(t, err)

		require.Len(t, service.Spec.Ports, 2)
		assert.Equal(t, zipkinPortName, service.Spec.Ports[0].Name)
		assert.Equal(t, statsdPortName, service.Spec.Ports[1].Name)

		require.Len(t, dk.Status.Conditions, 1)
		assert.Equal(t, serviceConditionType, dk.Status.Conditions[0].Type)
		assert.Equal(t, conditions.ServiceCreatedReason, dk.Status.Conditions[0].Reason)
		assert.Equal(t, metav1.ConditionTrue, dk.Status.Conditions[0].Status)
	})
	t.Run("default service name, remove service if it is not needed", func(t *testing.T) {
		dk := getTestDynakube(nil)
		dk.Status.Conditions = []metav1.Condition{
			{
				Type: serviceConditionType,
			},
		}

		mockK8sClient := fake.NewFakeClient()
		err := mockK8sClient.Create(t.Context(), &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      dk.TelemetryIngest().GetDefaultServiceName(),
				Namespace: dk.Namespace,
				Labels: map[string]string{
					k8slabel.AppComponentLabel: k8slabel.OtelCComponentLabel,
					k8slabel.AppCreatedByLabel: dk.Name,
				},
			},
		})
		require.NoError(t, err)

		err = NewReconciler(mockK8sClient, mockK8sClient, dk).Reconcile(t.Context())
		require.NoError(t, err)

		service := &corev1.Service{}
		err = mockK8sClient.Get(t.Context(), client.ObjectKey{Name: dk.TelemetryIngest().GetDefaultServiceName(), Namespace: dk.Namespace}, service)
		require.Error(t, err)
		assert.True(t, k8serrors.IsNotFound(err))

		require.Empty(t, dk.Status.Conditions)
	})
	t.Run("custom service name, remove service if it is not needed", func(t *testing.T) {
		dk := getTestDynakube(nil)
		dk.Status.Conditions = []metav1.Condition{
			{
				Type: serviceConditionType,
			},
		}

		mockK8sClient := fake.NewFakeClient()
		err := mockK8sClient.Create(t.Context(), &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testServiceName,
				Namespace: dk.Namespace,
				Labels: map[string]string{
					k8slabel.AppComponentLabel: k8slabel.OtelCComponentLabel,
					k8slabel.AppCreatedByLabel: dk.Name,
				},
			},
		})
		require.NoError(t, err)

		err = NewReconciler(mockK8sClient, mockK8sClient, dk).Reconcile(t.Context())
		require.NoError(t, err)

		service := &corev1.Service{}
		err = mockK8sClient.Get(t.Context(), client.ObjectKey{Name: dk.TelemetryIngest().GetDefaultServiceName(), Namespace: dk.Namespace}, service)
		require.Error(t, err)
		assert.True(t, k8serrors.IsNotFound(err))

		require.Empty(t, dk.Status.Conditions)
	})
	t.Run("update from default service to custom service", func(t *testing.T) {
		mockK8sClient := fake.NewFakeClient()
		dk := getTestDynakube(&telemetryingest.Spec{})
		err := NewReconciler(mockK8sClient, mockK8sClient, dk).Reconcile(t.Context())
		require.NoError(t, err)

		service := &corev1.Service{}
		err = mockK8sClient.Get(t.Context(), client.ObjectKey{Name: dk.TelemetryIngest().GetDefaultServiceName(), Namespace: dk.Namespace}, service)
		require.NoError(t, err)

		require.Len(t, dk.Status.Conditions, 1)
		assert.Equal(t, serviceConditionType, dk.Status.Conditions[0].Type)
		assert.Equal(t, conditions.ServiceCreatedReason, dk.Status.Conditions[0].Reason)
		assert.Equal(t, metav1.ConditionTrue, dk.Status.Conditions[0].Status)

		// update
		dk.Spec.TelemetryIngest = &telemetryingest.Spec{
			ServiceName: testServiceName,
		}
		err = NewReconciler(mockK8sClient, mockK8sClient, dk).Reconcile(t.Context())
		require.NoError(t, err)

		service = &corev1.Service{}
		err = mockK8sClient.Get(t.Context(), client.ObjectKey{Name: dk.TelemetryIngest().GetDefaultServiceName(), Namespace: dk.Namespace}, service)
		require.Error(t, err)
		assert.True(t, k8serrors.IsNotFound(err))

		assert.NotEmpty(t, dk.Status.Conditions)
	})
	t.Run("update from custom service to default service", func(t *testing.T) {
		mockK8sClient := fake.NewFakeClient()
		dk := getTestDynakube(&telemetryingest.Spec{
			ServiceName: testServiceName,
		})
		err := NewReconciler(mockK8sClient, mockK8sClient, dk).Reconcile(t.Context())
		require.NoError(t, err)

		service := &corev1.Service{}
		err = mockK8sClient.Get(t.Context(), client.ObjectKey{Name: testServiceName, Namespace: dk.Namespace}, service)
		require.NoError(t, err)

		require.Len(t, dk.Status.Conditions, 1)
		assert.Equal(t, serviceConditionType, dk.Status.Conditions[0].Type)
		assert.Equal(t, conditions.ServiceCreatedReason, dk.Status.Conditions[0].Reason)
		assert.Equal(t, metav1.ConditionTrue, dk.Status.Conditions[0].Status)

		// update
		dk.Spec.TelemetryIngest = &telemetryingest.Spec{}
		err = NewReconciler(mockK8sClient, mockK8sClient, dk).Reconcile(t.Context())
		require.NoError(t, err)

		service = &corev1.Service{}
		err = mockK8sClient.Get(t.Context(), client.ObjectKey{Name: testServiceName, Namespace: dk.Namespace}, service)
		require.Error(t, err)
		assert.True(t, k8serrors.IsNotFound(err))

		assert.NotEmpty(t, dk.Status.Conditions)
	})
}
