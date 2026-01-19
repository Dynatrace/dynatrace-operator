package activegate

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/activegate"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/extensions"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/communication"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/value"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/capability"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/internal/customproperties"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/internal/statefulset"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/connectioninfo"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/istio"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/version"
	dtclientmock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/clients/dynatrace"
	controllermock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/controllers"
	versionmock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/controllers/dynakube/version"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
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
	anyCtx      = mock.MatchedBy(func(context.Context) bool { return true })
	anyDynakube = mock.MatchedBy(func(*dynakube.DynaKube) bool { return true })

	testKubeSystemNamespace = &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "kube-system",
			UID:  "01234-5678-9012-3456",
		},
	}
)

func TestReconciler_Reconcile(t *testing.T) {
	t.Run("No sub reconciler runs if AG was not enabled", func(t *testing.T) {
		dk := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: testNamespace,
				Name:      testName,
			}}
		// don't initialize the other fields to cause a panic if anything is accessed
		r := Reconciler{dk: dk}

		err := r.Reconcile(t.Context())
		require.NoError(t, err)
	})
	t.Run("ALL sub reconciler runs if AG is enabled", func(t *testing.T) {
		dk := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: testNamespace,
				Name:      testName,
			},
			Spec: dynakube.DynaKubeSpec{
				EnableIstio: true,
				ActiveGate: activegate.Spec{
					Capabilities: []activegate.CapabilityDisplayName{activegate.RoutingCapability.DisplayName},
				},
			}}
		fakeClient := fake.NewClient()

		r := &Reconciler{
			dk:                   dk,
			client:               fakeClient,
			apiReader:            fakeClient,
			authTokenReconciler:  mockGenericReconcileOnce(t),
			istioReconciler:      mockIstioReconcile(t),
			connectionReconciler: mockGenericReconcileOnce(t),
			versionReconciler:    mockVersionReconcileOnce(t),
			pullSecretReconciler: mockGenericReconcileOnce(t),
		}
		setupCapabilityReconcile(t, r)

		err := r.Reconcile(t.Context())
		require.NoError(t, err)
	})
	t.Run("ALL sub reconciler (except the capability ones) runs if AG is not enabled, but was enabled before, so to clean up", func(t *testing.T) {
		dk := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: testNamespace,
				Name:      testName,
			},
			Spec: dynakube.DynaKubeSpec{
				EnableIstio: true,
				ActiveGate:  activegate.Spec{},
			},
			Status: dynakube.DynaKubeStatus{
				Conditions: []metav1.Condition{
					{Type: statefulset.ActiveGateStatefulSetConditionType},
				},
			},
		}
		fakeClient := fake.NewClient()

		r := Reconciler{
			dk:                   dk,
			client:               fakeClient,
			apiReader:            fakeClient,
			authTokenReconciler:  mockGenericReconcileOnce(t),
			istioReconciler:      mockIstioReconcile(t),
			connectionReconciler: mockGenericReconcileOnce(t),
			versionReconciler:    mockVersionReconcileOnce(t),
			pullSecretReconciler: mockGenericReconcileOnce(t),
			// panic if omitted builders are called
		}

		err := r.Reconcile(t.Context())
		require.NoError(t, err)
	})
	t.Run("Create AG capability (creation and deletion)", func(t *testing.T) {
		dk := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: testNamespace,
				Name:      testName,
			},
			Spec: dynakube.DynaKubeSpec{
				EnableIstio: true,
				ActiveGate: activegate.Spec{
					Capabilities: []activegate.CapabilityDisplayName{activegate.RoutingCapability.DisplayName},
				},
			},
		}
		fakeClient := fake.NewClient(testKubeSystemNamespace)

		r := NewReconciler(fakeClient, fakeClient, dk, createMockDtClient(t, true), nil, nil).(*Reconciler)
		r.connectionReconciler = mockGenericReconcileOnce(t)
		r.versionReconciler = mockVersionReconcileOnce(t)
		r.istioReconciler = mockIstioReconcile(t)
		r.pullSecretReconciler = mockGenericReconcileOnce(t)

		err := r.Reconcile(t.Context())
		require.NoError(t, err)

		var service corev1.Service
		err = fakeClient.Get(t.Context(), types.NamespacedName{Name: testServiceName, Namespace: testNamespace}, &service)
		require.NoError(t, err)

		var secret corev1.Secret
		err = fakeClient.Get(t.Context(), types.NamespacedName{Name: r.dk.ActiveGate().GetTLSSecretName(), Namespace: testNamespace}, &secret)
		require.NoError(t, err)

		// remove AG from spec
		dk.Spec.ActiveGate = activegate.Spec{}
		r.connectionReconciler = mockGenericReconcileOnce(t)
		r.versionReconciler = mockVersionReconcileOnce(t)
		r.pullSecretReconciler = mockGenericReconcileOnce(t)
		err = r.Reconcile(t.Context())
		require.NoError(t, err)

		err = fakeClient.Get(t.Context(), types.NamespacedName{Name: testServiceName, Namespace: testNamespace}, &service)
		assert.True(t, k8serrors.IsNotFound(err))

		err = fakeClient.Get(t.Context(), types.NamespacedName{Name: r.dk.ActiveGate().GetTLSSecretName(), Namespace: testNamespace}, &secret)
		assert.True(t, k8serrors.IsNotFound(err))
	})
	t.Run("Reconcile DynaKube without Proxy after a DynaKube with proxy must not interfere with the second DKs Proxy Secret", func(t *testing.T) { // TODO: This is not a unit test, it tests the functionality of another package, it should use a mock for that
		dkWithProxy := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: testNamespace,
				Name:      "proxyDk",
			},
			Spec: dynakube.DynaKubeSpec{
				Proxy:      &value.Source{Value: testProxyName},
				ActiveGate: activegate.Spec{Capabilities: []activegate.CapabilityDisplayName{activegate.KubeMonCapability.DisplayName}},
			},
		}
		dkNoProxy := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: testNamespace,
				Name:      "noProxyDk",
			},
			Spec: dynakube.DynaKubeSpec{
				ActiveGate: activegate.Spec{Capabilities: []activegate.CapabilityDisplayName{activegate.KubeMonCapability.DisplayName}},
			},
		}
		fakeClient := fake.NewClient()
		proxyReconciler := &Reconciler{
			client:               fakeClient,
			apiReader:            fakeClient,
			dk:                   dkWithProxy,
			authTokenReconciler:  mockGenericReconcileOnce(t),
			pullSecretReconciler: mockGenericReconcileOnce(t),
			connectionReconciler: mockGenericReconcileOnce(t),
			versionReconciler:    mockVersionReconcileOnce(t),
		}
		setupCapabilityReconcile(t, proxyReconciler)
		err := proxyReconciler.Reconcile(t.Context())
		require.NoError(t, err)

		noProxyReconciler := &Reconciler{
			client:               fakeClient,
			apiReader:            fakeClient,
			dk:                   dkNoProxy,
			authTokenReconciler:  mockGenericReconcileOnce(t),
			pullSecretReconciler: mockGenericReconcileOnce(t),
			connectionReconciler: mockGenericReconcileOnce(t),
			versionReconciler:    mockVersionReconcileOnce(t),
		}
		setupCapabilityReconcile(t, noProxyReconciler)
		err = noProxyReconciler.Reconcile(t.Context())
		require.NoError(t, err)
	})
	t.Run("Reconciles Kubernetes Monitoring", func(t *testing.T) {
		dk := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testName,
				Namespace: testNamespace,
			},
			Spec: dynakube.DynaKubeSpec{
				APIURL: "test-api-url",
				ActiveGate: activegate.Spec{
					Capabilities: []activegate.CapabilityDisplayName{
						activegate.KubeMonCapability.DisplayName,
					},
				},
			},
		}
		fakeClient := fake.NewClient(testKubeSystemNamespace)

		r := NewReconciler(fakeClient, fakeClient, dk, createMockDtClient(t, true), nil, nil).(*Reconciler)
		r.connectionReconciler = mockGenericReconcileOnce(t)
		r.versionReconciler = mockVersionReconcileOnce(t)
		r.pullSecretReconciler = mockGenericReconcileOnce(t)

		err := r.Reconcile(t.Context())
		require.NoError(t, err)

		require.NoError(t, err)

		var statefulSet appsv1.StatefulSet

		name := capability.CalculateStatefulSetName(dk.Name)
		err = fakeClient.Get(t.Context(), client.ObjectKey{Name: name, Namespace: testNamespace}, &statefulSet)

		require.NoError(t, err)
		assert.NotNil(t, statefulSet)
		assert.Equal(t, "test-name-activegate", statefulSet.GetName())
	})
}

func TestExtensionControllerRequiresActiveGate(t *testing.T) {
	t.Run("no activegate is created when extensions are disabled in dk, and no capability is configured", func(t *testing.T) {
		instance := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: testNamespace,
				Name:      testName,
			},
			Spec: dynakube.DynaKubeSpec{
				ActiveGate: activegate.Spec{Capabilities: []activegate.CapabilityDisplayName{}},
				Extensions: nil,
			},
		}

		fakeClient := fake.NewClient(testKubeSystemNamespace)

		r := NewReconciler(fakeClient, fakeClient, instance, createMockDtClient(t, false), nil, nil).(*Reconciler)
		r.connectionReconciler = controllermock.NewReconciler(t)
		r.versionReconciler = versionmock.NewReconciler(t)
		r.pullSecretReconciler = controllermock.NewReconciler(t)

		err := r.Reconcile(t.Context())
		require.NoError(t, err)

		var service corev1.Service
		err = fakeClient.Get(t.Context(), types.NamespacedName{Name: testServiceName, Namespace: testNamespace}, &service)
		assert.True(t, k8serrors.IsNotFound(err))

		var statefulset appsv1.StatefulSet
		err = fakeClient.Get(t.Context(), types.NamespacedName{Name: instance.Name + "-activegate", Namespace: testNamespace}, &statefulset)
		assert.True(t, k8serrors.IsNotFound(err))
	})
	t.Run("activegate is created when extensions are enabled in dk, but no activegate is configured", func(t *testing.T) {
		instance := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: testNamespace,
				Name:      testName,
			},
			Spec: dynakube.DynaKubeSpec{
				Extensions: &extensions.Spec{Prometheus: &extensions.PrometheusSpec{}},
			},
		}

		fakeClient := fake.NewClient(testKubeSystemNamespace)

		r := NewReconciler(fakeClient, fakeClient, instance, createMockDtClient(t, true), nil, nil).(*Reconciler)
		r.connectionReconciler = mockGenericReconcileOnce(t)
		r.versionReconciler = mockVersionReconcileOnce(t)
		r.pullSecretReconciler = mockGenericReconcileOnce(t)

		err := r.Reconcile(t.Context())
		require.NoError(t, err)

		var service corev1.Service
		err = fakeClient.Get(t.Context(), types.NamespacedName{Name: testServiceName, Namespace: testNamespace}, &service)
		require.NoError(t, err)
		require.Equal(t, testServiceName, service.Name)

		var statefulset appsv1.StatefulSet
		err = fakeClient.Get(t.Context(), types.NamespacedName{Name: instance.Name + "-activegate", Namespace: testNamespace}, &statefulset)
		require.NoError(t, err)
	})
	t.Run("activegate is created when extensions are enabled in dk, but no activegate capability is configured", func(t *testing.T) {
		instance := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: testNamespace,
				Name:      testName,
			},
			Spec: dynakube.DynaKubeSpec{
				ActiveGate: activegate.Spec{Capabilities: []activegate.CapabilityDisplayName{}},
				Extensions: &extensions.Spec{Prometheus: &extensions.PrometheusSpec{}},
			},
		}

		fakeClient := fake.NewClient(testKubeSystemNamespace)

		r := NewReconciler(fakeClient, fakeClient, instance, createMockDtClient(t, true), nil, nil).(*Reconciler)
		r.connectionReconciler = mockGenericReconcileOnce(t)
		r.versionReconciler = mockVersionReconcileOnce(t)
		r.pullSecretReconciler = mockGenericReconcileOnce(t)

		err := r.Reconcile(t.Context())
		require.NoError(t, err)

		var service corev1.Service
		err = fakeClient.Get(t.Context(), types.NamespacedName{Name: testServiceName, Namespace: testNamespace}, &service)
		require.NoError(t, err)
		require.Equal(t, testServiceName, service.Name)

		var statefulset appsv1.StatefulSet
		err = fakeClient.Get(t.Context(), types.NamespacedName{Name: instance.Name + "-activegate", Namespace: testNamespace}, &statefulset)
		require.NoError(t, err)
	})
	t.Run("activegate is created when extensions are enabled in dk, and activegate kubernetes is configured", func(t *testing.T) {
		instance := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: testNamespace,
				Name:      testName,
			},
			Spec: dynakube.DynaKubeSpec{
				ActiveGate: activegate.Spec{Capabilities: []activegate.CapabilityDisplayName{activegate.KubeMonCapability.DisplayName}},
				Extensions: &extensions.Spec{Prometheus: &extensions.PrometheusSpec{}},
			},
		}

		fakeClient := fake.NewClient(testKubeSystemNamespace)

		r := NewReconciler(fakeClient, fakeClient, instance, createMockDtClient(t, true), nil, nil).(*Reconciler)
		r.connectionReconciler = mockGenericReconcileOnce(t)
		r.versionReconciler = mockVersionReconcileOnce(t)
		r.pullSecretReconciler = mockGenericReconcileOnce(t)

		err := r.Reconcile(t.Context())
		require.NoError(t, err)

		var service corev1.Service
		err = fakeClient.Get(t.Context(), types.NamespacedName{Name: testServiceName, Namespace: testNamespace}, &service)
		require.NoError(t, err)
		require.Equal(t, testServiceName, service.Name)

		var statefulset appsv1.StatefulSet
		err = fakeClient.Get(t.Context(), types.NamespacedName{Name: instance.Name + "-activegate", Namespace: testNamespace}, &statefulset)
		require.NoError(t, err)
	})
	t.Run("activegate is created when extensions are enabled in dk, but activegate capabilities are removed", func(t *testing.T) {
		instance := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: testNamespace,
				Name:      testName,
			},
			Spec: dynakube.DynaKubeSpec{
				ActiveGate: activegate.Spec{Capabilities: []activegate.CapabilityDisplayName{activegate.KubeMonCapability.DisplayName}},
				Extensions: &extensions.Spec{Prometheus: &extensions.PrometheusSpec{}},
			},
		}

		fakeClient := fake.NewClient(testKubeSystemNamespace)

		r := NewReconciler(fakeClient, fakeClient, instance, createMockDtClient(t, true), nil, nil).(*Reconciler)
		r.connectionReconciler = mockGenericReconcileOnce(t)
		r.versionReconciler = mockVersionReconcileOnce(t)
		r.pullSecretReconciler = mockGenericReconcileOnce(t)

		err := r.Reconcile(t.Context())
		require.NoError(t, err)

		var service corev1.Service
		err = fakeClient.Get(t.Context(), types.NamespacedName{Name: testServiceName, Namespace: testNamespace}, &service)
		require.NoError(t, err)
		require.Equal(t, testServiceName, service.Name)

		var statefulset appsv1.StatefulSet
		err = fakeClient.Get(t.Context(), types.NamespacedName{Name: instance.Name + "-activegate", Namespace: testNamespace}, &statefulset)
		require.NoError(t, err)

		// remove AG from spec
		r.dk.Spec.ActiveGate = activegate.Spec{}
		r.connectionReconciler = mockGenericReconcileOnce(t)
		r.versionReconciler = mockVersionReconcileOnce(t)
		r.pullSecretReconciler = mockGenericReconcileOnce(t)
		err = r.Reconcile(t.Context())
		require.NoError(t, err)

		var service1 corev1.Service
		err = fakeClient.Get(t.Context(), types.NamespacedName{Name: testServiceName, Namespace: testNamespace}, &service1)
		require.NoError(t, err)
		require.Equal(t, testServiceName, service1.Name)

		err = fakeClient.Get(t.Context(), types.NamespacedName{Name: instance.Name + "-activegate", Namespace: testNamespace}, &statefulset)
		require.NoError(t, err)

		// disable extensions
		r.dk.Spec.Extensions = nil
		r.connectionReconciler = mockGenericReconcileOnce(t)
		r.versionReconciler = mockVersionReconcileOnce(t)
		r.pullSecretReconciler = mockGenericReconcileOnce(t)
		err = r.Reconcile(t.Context())
		require.NoError(t, err)

		err = fakeClient.Get(t.Context(), types.NamespacedName{Name: testServiceName, Namespace: testNamespace}, &service)
		assert.True(t, k8serrors.IsNotFound(err))

		err = fakeClient.Get(t.Context(), types.NamespacedName{Name: instance.Name + "-activegate", Namespace: testNamespace}, &statefulset)
		assert.True(t, k8serrors.IsNotFound(err))
	})
}

func TestServiceCreation(t *testing.T) {
	assertContainsAllPorts := func(t *testing.T, expectedPorts []string, servicePorts []corev1.ServicePort) {
		actualPorts := make([]string, 0, len(servicePorts))

		for _, servicePort := range servicePorts {
			actualPorts = append(actualPorts, servicePort.Name)
		}

		for _, expectedPort := range expectedPorts {
			assert.Contains(t, actualPorts, expectedPort)
		}
	}

	getTestActiveGateService := func(t *testing.T, fakeClient client.Client) corev1.Service {
		var activegateService corev1.Service
		err := fakeClient.Get(t.Context(), client.ObjectKey{Name: testServiceName, Namespace: testNamespace}, &activegateService)

		require.NoError(t, err)

		return activegateService
	}

	dynatraceClient := dtclientmock.NewClient(t)
	dynatraceClient.EXPECT().GetActiveGateAuthToken(anyCtx, testName).Return(&dtclient.ActiveGateAuthTokenInfo{TokenID: "test", Token: "dt.some.valuegoeshere"}, nil)

	dk := &dynakube.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNamespace,
			Name:      testName,
		},
		Spec: dynakube.DynaKubeSpec{
			ActiveGate: activegate.Spec{},
		},
	}

	t.Run("service exposes all ports for every capabilities", func(t *testing.T) {
		expectedCapabilityPorts := map[activegate.CapabilityDisplayName][]string{
			activegate.RoutingCapability.DisplayName: {
				consts.HTTPServicePortName,
				consts.HTTPSServicePortName,
			},
			activegate.MetricsIngestCapability.DisplayName: {
				consts.HTTPServicePortName,
				consts.HTTPSServicePortName,
			},
			activegate.DynatraceAPICapability.DisplayName: {
				consts.HTTPServicePortName,
				consts.HTTPSServicePortName,
			},
			activegate.KubeMonCapability.DisplayName: {
				consts.HTTPServicePortName,
				consts.HTTPSServicePortName,
			},
		}

		for capName, expectedPorts := range expectedCapabilityPorts {
			fakeClient := fake.NewClient(testKubeSystemNamespace)

			reconciler := NewReconciler(fakeClient, fakeClient, dk, dynatraceClient, nil, nil).(*Reconciler)
			reconciler.connectionReconciler = mockGenericReconcileOnce(t)
			reconciler.versionReconciler = mockVersionReconcileOnce(t)
			reconciler.pullSecretReconciler = mockGenericReconcileOnce(t)

			dk.Spec.ActiveGate.Capabilities = []activegate.CapabilityDisplayName{
				capName,
			}

			err := reconciler.Reconcile(t.Context())
			require.NoError(t, err)

			if len(expectedPorts) == 0 {
				err = fakeClient.Get(t.Context(), client.ObjectKey{Name: testServiceName, Namespace: testNamespace}, &corev1.Service{})

				assert.True(t, k8serrors.IsNotFound(err))

				continue
			}

			activegateService := getTestActiveGateService(t, fakeClient)
			assertContainsAllPorts(t, expectedPorts, activegateService.Spec.Ports)
		}
	})
}

func TestReconcile_ActivegateConfigMap(t *testing.T) {
	const (
		testName            = "test-name"
		testNamespace       = "test-namespace"
		testTenantUUID      = "test-uuid"
		testTenantEndpoints = "test-endpoints"
	)

	dk := &dynakube.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNamespace,
			Name:      testName,
		},
		Spec: dynakube.DynaKubeSpec{
			ActiveGate: activegate.Spec{Capabilities: []activegate.CapabilityDisplayName{activegate.KubeMonCapability.DisplayName}},
		},
		Status: dynakube.DynaKubeStatus{
			ActiveGate: activegate.Status{
				ConnectionInfo: communication.ConnectionInfo{
					TenantUUID: testTenantUUID,
					Endpoints:  testTenantEndpoints,
				},
			},
		},
	}

	t.Run("create activegate ConfigMap", func(t *testing.T) {
		fakeClient := fake.NewClient(testKubeSystemNamespace)
		r := &Reconciler{
			client:               fakeClient,
			apiReader:            fakeClient,
			dk:                   dk,
			authTokenReconciler:  mockGenericReconcileOnce(t),
			pullSecretReconciler: mockGenericReconcileOnce(t),
			connectionReconciler: mockGenericReconcileOnce(t),
			versionReconciler:    mockVersionReconcileOnce(t),
		}
		setupCapabilityReconcile(t, r)
		err := r.Reconcile(t.Context())
		require.NoError(t, err)

		var actual corev1.ConfigMap
		err = fakeClient.Get(t.Context(), client.ObjectKey{Name: dk.ActiveGate().GetConnectionInfoConfigMapName(), Namespace: testNamespace}, &actual)
		require.NoError(t, err)
		assert.Equal(t, testTenantUUID, actual.Data[connectioninfo.TenantUUIDKey])
		assert.Equal(t, testTenantEndpoints, actual.Data[connectioninfo.CommunicationEndpointsKey])
	})
}

func mockGenericReconcileOnce(t *testing.T) *controllermock.Reconciler {
	t.Helper()

	genericReconciler := controllermock.NewReconciler(t)
	genericReconciler.EXPECT().Reconcile(anyCtx).Return(nil).Once()

	return genericReconciler
}

func mockVersionReconcileOnce(t *testing.T) version.Reconciler {
	t.Helper()

	versionReconciler := versionmock.NewReconciler(t)
	versionReconciler.EXPECT().ReconcileActiveGate(anyCtx, anyDynakube).Return(nil).Once()

	return versionReconciler
}

func mockIstioReconcile(t *testing.T) istio.Reconciler {
	t.Helper()

	reconciler := NewMockReconciler(t)
	reconciler.EXPECT().ReconcileActiveGateCommunicationHosts(anyCtx, anyDynakube).Return(nil)

	return reconciler
}

func createMockDtClient(t *testing.T, authTokenRouteRequired bool) *dtclientmock.Client {
	t.Helper()

	dtc := dtclientmock.NewClient(t)
	if authTokenRouteRequired {
		dtc.EXPECT().GetActiveGateAuthToken(anyCtx, testName).Return(&dtclient.ActiveGateAuthTokenInfo{TokenID: "test", Token: "dt.some.valuegoeshere"}, nil)
	}

	return dtc
}

func setupCapabilityReconcile(t *testing.T, r *Reconciler) {
	t.Helper()

	// initialize this here to ensure that the test fails if Reconcile isn't called
	mock := mockGenericReconcileOnce(t)

	r.newStatefulsetReconcilerFunc = func(_ client.Client, _ client.Reader, _ *dynakube.DynaKube, _ capability.Capability) controllers.Reconciler {
		return &statefulset.Reconciler{}
	}
	r.newCustomPropertiesReconcilerFunc = func(_ string, _ *value.Source) controllers.Reconciler {
		return &customproperties.Reconciler{}
	}
	r.newCapabilityReconcilerFunc = func(
		_ client.Client,
		_ capability.Capability,
		_ *dynakube.DynaKube,
		statefulSetReconciler controllers.Reconciler,
		customPropertiesReconciler controllers.Reconciler,
		_ controllers.Reconciler,
	) controllers.Reconciler {
		require.Equal(t, &statefulset.Reconciler{}, statefulSetReconciler)
		require.Equal(t, &customproperties.Reconciler{}, customPropertiesReconciler)

		return mock
	}
}
