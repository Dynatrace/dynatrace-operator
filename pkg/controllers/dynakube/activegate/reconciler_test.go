package activegate

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/capability"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/connectioninfo"
	mocks "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/clients/dynatrace"
	controllerMocks "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/controllers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
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
	dtc := mocks.NewClient(t)
	dtc.On("GetActiveGateAuthToken", mock.AnythingOfType("context.backgroundCtx"), testName).Return(&dtclient.ActiveGateAuthTokenInfo{}, nil)

	t.Run(`Create works with minimal setup`, func(t *testing.T) {
		instance := &dynatracev1beta1.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: testNamespace,
				Name:      testName,
			}}
		fakeClient := fake.NewClient()
		r := NewReconciler(fakeClient, fakeClient, scheme.Scheme, instance, dtc)
		err := r.Reconcile(context.Background())
		require.NoError(t, err)
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
		r := NewReconciler(fakeClient, fakeClient, scheme.Scheme, instance, dtc)
		err := r.Reconcile(context.Background())
		require.NoError(t, err)

		var service corev1.Service
		err = fakeClient.Get(context.Background(), types.NamespacedName{Name: testServiceName, Namespace: testNamespace}, &service)
		require.NoError(t, err)

		// remove AG from spec
		instance.Spec.ActiveGate = dynatracev1beta1.ActiveGateSpec{}
		err = r.Reconcile(context.Background())
		require.NoError(t, err)

		err = fakeClient.Get(context.Background(), types.NamespacedName{Name: testServiceName, Namespace: testNamespace}, &service)
		assert.True(t, k8serrors.IsNotFound(err))
	})
	t.Run("Reconcile DynaKube without Proxy after a DynaKube with proxy must not interfere with the second DKs Proxy Secret", func(t *testing.T) { // TODO: This is not a unit test, it tests the functionality of another package, it should use a mock for that
		dynaKubeWithProxy := &dynatracev1beta1.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: testNamespace,
				Name:      "proxyDk",
			},
			Spec: dynatracev1beta1.DynaKubeSpec{
				Proxy:      &dynatracev1beta1.DynaKubeProxy{Value: testProxyName},
				ActiveGate: dynatracev1beta1.ActiveGateSpec{Capabilities: []dynatracev1beta1.CapabilityDisplayName{dynatracev1beta1.KubeMonCapability.DisplayName}},
			},
		}
		dynaKubeNoProxy := &dynatracev1beta1.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: testNamespace,
				Name:      "noProxyDk",
			},
			Spec: dynatracev1beta1.DynaKubeSpec{
				ActiveGate: dynatracev1beta1.ActiveGateSpec{Capabilities: []dynatracev1beta1.CapabilityDisplayName{dynatracev1beta1.KubeMonCapability.DisplayName}},
			},
		}
		fakeClient := fake.NewClient()
		fakeReconciler := controllerMocks.NewReconciler(t)
		fakeReconciler.On("Reconcile", mock.Anything).Return(nil)
		proxyReconciler := Reconciler{
			client:              fakeClient,
			apiReader:           fakeClient,
			dynakube:            dynaKubeWithProxy,
			scheme:              scheme.Scheme,
			authTokenReconciler: fakeReconciler,
			newStatefulsetReconcilerFunc: func(_ client.Client, _ client.Reader, _ *runtime.Scheme, _ *dynatracev1beta1.DynaKube, _ capability.Capability) controllers.Reconciler {
				return fakeReconciler
			},
			newCapabilityReconcilerFunc: func(_ client.Client, _ capability.Capability, _ *dynatracev1beta1.DynaKube, _ controllers.Reconciler, _ controllers.Reconciler) controllers.Reconciler {
				return fakeReconciler
			},
			newCustomPropertiesReconcilerFunc: func(_ string, customPropertiesSource *dynatracev1beta1.DynaKubeValueSource) controllers.Reconciler {
				return fakeReconciler
			},
		}
		err := proxyReconciler.Reconcile(context.Background())
		require.NoError(t, err)

		noProxyReconciler := Reconciler{
			client:              fakeClient,
			apiReader:           fakeClient,
			dynakube:            dynaKubeNoProxy,
			scheme:              scheme.Scheme,
			authTokenReconciler: fakeReconciler,
			newStatefulsetReconcilerFunc: func(_ client.Client, _ client.Reader, _ *runtime.Scheme, _ *dynatracev1beta1.DynaKube, _ capability.Capability) controllers.Reconciler {
				return fakeReconciler
			},
			newCapabilityReconcilerFunc: func(_ client.Client, _ capability.Capability, _ *dynatracev1beta1.DynaKube, _ controllers.Reconciler, _ controllers.Reconciler) controllers.Reconciler {
				return fakeReconciler
			},
			newCustomPropertiesReconcilerFunc: func(_ string, customPropertiesSource *dynatracev1beta1.DynaKubeValueSource) controllers.Reconciler {
				return fakeReconciler
			},
		}
		err = noProxyReconciler.Reconcile(context.Background())
		require.NoError(t, err)
	})
	t.Run(`Reconciles Kubernetes Monitoring`, func(t *testing.T) {
		instance := &dynatracev1beta1.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testName,
				Namespace: testNamespace,
			},
			Spec: dynatracev1beta1.DynaKubeSpec{
				APIURL: "test-api-url",
				KubernetesMonitoring: dynatracev1beta1.KubernetesMonitoringSpec{
					Enabled: true,
				}},
		}
		fakeClient := fake.NewClient(testKubeSystemNamespace)
		r := NewReconciler(fakeClient, fakeClient, scheme.Scheme, instance, dtc)
		err := r.Reconcile(context.Background())
		require.NoError(t, err)

		require.NoError(t, err)

		var statefulSet appsv1.StatefulSet

		kubeMonCapability := capability.NewKubeMonCapability(instance)
		name := capability.CalculateStatefulSetName(kubeMonCapability, instance.Name)
		err = fakeClient.Get(context.Background(), client.ObjectKey{Name: name, Namespace: testNamespace}, &statefulSet)

		require.NoError(t, err)
		assert.NotNil(t, statefulSet)
		assert.Equal(t, "test-name-kubemon", statefulSet.GetName())
	})
}

func TestServiceCreation(t *testing.T) {
	dynatraceClient := mocks.NewClient(t)
	dynatraceClient.On("GetActiveGateAuthToken", mock.AnythingOfType("context.backgroundCtx"), testName).Return(&dtclient.ActiveGateAuthTokenInfo{}, nil)

	dynakube := &dynatracev1beta1.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNamespace,
			Name:      testName,
		},
		Spec: dynatracev1beta1.DynaKubeSpec{
			ActiveGate: dynatracev1beta1.ActiveGateSpec{},
		},
	}

	t.Run("service exposes correct ports for single capabilities", func(t *testing.T) {
		expectedCapabilityPorts := map[dynatracev1beta1.CapabilityDisplayName][]string{
			dynatracev1beta1.RoutingCapability.DisplayName: {
				consts.HttpsServicePortName,
			},
			dynatracev1beta1.MetricsIngestCapability.DisplayName: {
				consts.HttpsServicePortName,
				consts.HttpServicePortName,
			},
			dynatracev1beta1.DynatraceApiCapability.DisplayName: {
				consts.HttpsServicePortName,
			},
			dynatracev1beta1.KubeMonCapability.DisplayName: {},
		}

		for capName, expectedPorts := range expectedCapabilityPorts {
			fakeClient := fake.NewClient(testKubeSystemNamespace)
			reconciler := NewReconciler(fakeClient, fakeClient, scheme.Scheme, dynakube, dynatraceClient)
			dynakube.Spec.ActiveGate.Capabilities = []dynatracev1beta1.CapabilityDisplayName{
				capName,
			}

			err := reconciler.Reconcile(context.Background())
			require.NoError(t, err)

			if len(expectedPorts) == 0 {
				err = fakeClient.Get(context.Background(), client.ObjectKey{Name: testServiceName, Namespace: testNamespace}, &corev1.Service{})

				assert.True(t, k8serrors.IsNotFound(err))

				continue
			}

			activegateService := getTestActiveGateService(t, fakeClient)
			assertContainsAllPorts(t, expectedPorts, activegateService.Spec.Ports)
		}
	})

	t.Run("service exposes correct ports for multiple capabilities", func(t *testing.T) {
		fakeClient := fake.NewClient(testKubeSystemNamespace)
		reconciler := NewReconciler(fakeClient, fakeClient, scheme.Scheme, dynakube, dynatraceClient)
		dynakube.Spec.ActiveGate.Capabilities = []dynatracev1beta1.CapabilityDisplayName{
			dynatracev1beta1.RoutingCapability.DisplayName,
		}
		expectedPorts := []string{
			consts.HttpsServicePortName,
		}

		err := reconciler.Reconcile(context.Background())
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
	err := fakeClient.Get(context.Background(), client.ObjectKey{Name: testServiceName, Namespace: testNamespace}, &activegateService)

	require.NoError(t, err)

	return activegateService
}

func TestReconcile_ActivegateConfigMap(t *testing.T) {
	const (
		testName            = "test-name"
		testNamespace       = "test-namespace"
		testTenantUUID      = "test-uuid"
		testTenantEndpoints = "test-endpoints"
	)

	dynakube := &dynatracev1beta1.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNamespace,
			Name:      testName,
		},
		Spec: dynatracev1beta1.DynaKubeSpec{
			ActiveGate: dynatracev1beta1.ActiveGateSpec{Capabilities: []dynatracev1beta1.CapabilityDisplayName{dynatracev1beta1.KubeMonCapability.DisplayName}},
		},
		Status: dynatracev1beta1.DynaKubeStatus{
			ActiveGate: dynatracev1beta1.ActiveGateStatus{
				ConnectionInfoStatus: dynatracev1beta1.ActiveGateConnectionInfoStatus{
					ConnectionInfoStatus: dynatracev1beta1.ConnectionInfoStatus{
						TenantUUID:  testTenantUUID,
						Endpoints:   testTenantEndpoints,
						LastRequest: metav1.Time{},
					},
				},
			},
		},
	}

	t.Run(`create activegate ConfigMap`, func(t *testing.T) {
		fakeReconciler := controllerMocks.NewReconciler(t)
		fakeReconciler.On("Reconcile", mock.Anything).Return(nil)

		fakeClient := fake.NewClient(testKubeSystemNamespace)
		r := Reconciler{
			client:              fakeClient,
			apiReader:           fakeClient,
			dynakube:            dynakube,
			scheme:              scheme.Scheme,
			authTokenReconciler: fakeReconciler,
			newStatefulsetReconcilerFunc: func(_ client.Client, _ client.Reader, _ *runtime.Scheme, _ *dynatracev1beta1.DynaKube, _ capability.Capability) controllers.Reconciler {
				return fakeReconciler
			},
			newCapabilityReconcilerFunc: func(_ client.Client, _ capability.Capability, _ *dynatracev1beta1.DynaKube, _ controllers.Reconciler, _ controllers.Reconciler) controllers.Reconciler {
				return fakeReconciler
			},
			newCustomPropertiesReconcilerFunc: func(_ string, _ *dynatracev1beta1.DynaKubeValueSource) controllers.Reconciler {
				return fakeReconciler
			},
		}
		err := r.Reconcile(context.Background())
		require.NoError(t, err)

		var actual corev1.ConfigMap
		err = fakeClient.Get(context.TODO(), client.ObjectKey{Name: dynakube.ActiveGateConnectionInfoConfigMapName(), Namespace: testNamespace}, &actual)
		require.NoError(t, err)
		assert.Equal(t, testTenantUUID, actual.Data[connectioninfo.TenantUUIDName])
		assert.Equal(t, testTenantEndpoints, actual.Data[connectioninfo.CommunicationEndpointsName])
	})
}
