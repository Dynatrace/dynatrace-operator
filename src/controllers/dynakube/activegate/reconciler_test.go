package activegate

import (
	"context"
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/activegate/consts"
	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
	"github.com/Dynatrace/dynatrace-operator/src/scheme"
	"github.com/Dynatrace/dynatrace-operator/src/scheme/fake"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	testName        = "test-name"
	testNamespace   = "test-namespace"
	testProxyName   = "test-proxy"
	testServiceName = testName + "-activegate"
)

var (
	testKubeSystemNamespace = &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "kube-system",
			UID:  "01234-5678-9012-3456",
		},
	}
)

func TestReconciler_Reconcile(t *testing.T) {
	dtc := &dtclient.MockDynatraceClient{}
	dtc.On("GetActiveGateAuthToken", testName).Return(&dtclient.ActiveGateAuthTokenInfo{}, nil)

	t.Run(`Create works with minimal setup`, func(t *testing.T) {
		instance := &dynatracev1beta1.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: testNamespace,
				Name:      testName,
			}}
		fakeClient := fake.NewClient()
		r := NewReconciler(context.TODO(), fakeClient, fakeClient, scheme.Scheme, instance, dtc)
		err := r.Reconcile()
		require.NoError(t, err)
	})
	t.Run(`Create AG proxy secret`, func(t *testing.T) {
		instance := &dynatracev1beta1.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: testNamespace,
				Name:      testName,
			},
			Spec: dynatracev1beta1.DynaKubeSpec{
				Proxy: &dynatracev1beta1.DynaKubeProxy{Value: testProxyName},
			},
		}
		fakeClient := fake.NewClient()
		r := NewReconciler(context.TODO(), fakeClient, fakeClient, scheme.Scheme, instance, dtc)
		err := r.Reconcile()
		require.NoError(t, err)

		var proxySecret corev1.Secret
		err = fakeClient.Get(context.TODO(), types.NamespacedName{Name: "dynatrace-activegate-internal-proxy", Namespace: testNamespace}, &proxySecret)
		assert.NoError(t, err)
	})
	t.Run(`Create AG capability (creation and deletion)`, func(t *testing.T) {
		instance := &dynatracev1beta1.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: testNamespace,
				Name:      testName,
			},
			Spec: dynatracev1beta1.DynaKubeSpec{
				ActiveGate: dynatracev1beta1.ActiveGateSpec{
					Capabilities: []dynatracev1beta1.CapabilityDisplayName{dynatracev1beta1.RoutingCapability.DisplayName},
				},
			},
		}
		fakeClient := fake.NewClient(testKubeSystemNamespace)
		r := NewReconciler(context.TODO(), fakeClient, fakeClient, scheme.Scheme, instance, dtc)
		err := r.Reconcile()
		require.NoError(t, err)

		var service corev1.Service
		err = fakeClient.Get(context.TODO(), types.NamespacedName{Name: testServiceName, Namespace: testNamespace}, &service)
		require.NoError(t, err)

		// remove AG from spec
		instance.Spec.ActiveGate = dynatracev1beta1.ActiveGateSpec{}
		err = r.Reconcile()
		require.NoError(t, err)

		err = fakeClient.Get(context.TODO(), types.NamespacedName{Name: testServiceName, Namespace: testNamespace}, &service)
		assert.True(t, k8serrors.IsNotFound(err))
	})
}

func TestServiceCreation(t *testing.T) {
	dynatraceClient := &dtclient.MockDynatraceClient{}
	dynakube := &dynatracev1beta1.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNamespace,
			Name:      testName,
			Annotations: map[string]string{
				dynatracev1beta1.AnnotationFeatureActiveGateAuthToken: "false",
			},
		},
		Spec: dynatracev1beta1.DynaKubeSpec{
			ActiveGate: dynatracev1beta1.ActiveGateSpec{},
		},
	}

	t.Run("service exposes correct ports for single capabilities", func(t *testing.T) {
		expectedCapabilityPorts := map[dynatracev1beta1.CapabilityDisplayName][]string{
			dynatracev1beta1.RoutingCapability.DisplayName: {
				consts.HttpsServicePortName,
				consts.HttpServicePortName,
			},
			dynatracev1beta1.MetricsIngestCapability.DisplayName: {
				consts.HttpsServicePortName,
				consts.HttpServicePortName,
			},
			dynatracev1beta1.DynatraceApiCapability.DisplayName: {
				consts.HttpsServicePortName,
				consts.HttpServicePortName,
			},
			dynatracev1beta1.StatsdIngestCapability.DisplayName: {
				consts.StatsdIngestPortName,
			},
			dynatracev1beta1.KubeMonCapability.DisplayName: {},
		}

		for capability, expectedPorts := range expectedCapabilityPorts {
			fakeClient := fake.NewClient(testKubeSystemNamespace)
			reconciler := NewReconciler(context.TODO(), fakeClient, fakeClient, scheme.Scheme, dynakube, dynatraceClient)
			dynakube.Spec.ActiveGate.Capabilities = []dynatracev1beta1.CapabilityDisplayName{
				capability,
			}

			err := reconciler.Reconcile()
			require.NoError(t, err)

			if len(expectedPorts) == 0 {
				err = fakeClient.Get(context.TODO(), client.ObjectKey{Name: testServiceName, Namespace: testNamespace}, &corev1.Service{})

				assert.True(t, k8serrors.IsNotFound(err))
				continue
			}

			activegateService := getTestActiveGateService(t, fakeClient)
			assertContainsAllPorts(t, expectedPorts, activegateService.Spec.Ports)
		}
	})

	t.Run("service exposes correct ports for multiple capabilities", func(t *testing.T) {
		fakeClient := fake.NewClient(testKubeSystemNamespace)
		reconciler := NewReconciler(context.TODO(), fakeClient, fakeClient, scheme.Scheme, dynakube, dynatraceClient)
		dynakube.Spec.ActiveGate.Capabilities = []dynatracev1beta1.CapabilityDisplayName{
			dynatracev1beta1.RoutingCapability.DisplayName,
			dynatracev1beta1.StatsdIngestCapability.DisplayName,
		}
		expectedPorts := []string{
			consts.HttpsServicePortName,
			consts.HttpServicePortName,
			consts.StatsdIngestPortName,
		}

		err := reconciler.Reconcile()
		require.NoError(t, err)

		activegateService := getTestActiveGateService(t, fakeClient)
		assertContainsAllPorts(t, expectedPorts, activegateService.Spec.Ports)
	})
}

func assertContainsAllPorts(t *testing.T, expectedPorts []string, servicePorts []corev1.ServicePort) {
	actualPorts := make([]string, 0, len(servicePorts))

	for _, servicePort := range servicePorts {
		actualPorts = append(actualPorts, servicePort.Name)
	}

	for _, expectedPort := range expectedPorts {
		assert.Contains(t, actualPorts, expectedPort)
	}
}

func getTestActiveGateService(t *testing.T, fakeClient client.Client) corev1.Service {
	var activegateService corev1.Service
	err := fakeClient.Get(context.TODO(), client.ObjectKey{Name: testServiceName, Namespace: testNamespace}, &activegateService)

	require.NoError(t, err)

	return activegateService
}
