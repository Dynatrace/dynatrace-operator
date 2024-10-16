package activegate

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/communication"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/value"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube/activegate"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/capability"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/connectioninfo"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/dtpullsecret"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/istio"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/version"
	dtclientmock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/clients/dynatrace"
	controllermock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/controllers"
	istiomock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/controllers/dynakube/istio"
	versionmock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/controllers/dynakube/version"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
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
	t.Run(`Create works with minimal setup`, func(t *testing.T) {
		dk := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: testNamespace,
				Name:      testName,
			}}
		fakeClient := fake.NewClient()
		r := NewReconciler(fakeClient, fakeClient, dk, createMockDtClient(t, false), nil, nil)
		err := r.Reconcile(context.Background())
		require.NoError(t, err)
	})
	t.Run(`Pull secret reconciler is called even if ActiveGate disabled`, func(t *testing.T) {
		dk := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: testNamespace,
				Name:      testName,
			},
			Status: dynakube.DynaKubeStatus{
				Conditions: []metav1.Condition{
					{
						Type:   dtpullsecret.PullSecretConditionType,
						Status: metav1.ConditionTrue,
					},
				},
			},
		}

		fakeClient := fake.NewClientWithIndex(dk, &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: testNamespace,
				Name:      testName + "-pull-secret",
			},
		})
		r := NewReconciler(fakeClient, fakeClient, dk, createMockDtClient(t, false), nil, nil)
		err := r.Reconcile(context.Background())
		require.NoError(t, err)

		var secret corev1.Secret
		err = fakeClient.Get(context.Background(), types.NamespacedName{Name: testName + "-pull-secret", Namespace: testNamespace}, &secret)
		require.True(t, k8serrors.IsNotFound(err))
		require.Nil(t, meta.FindStatusCondition(dk.Status.Conditions, dtpullsecret.PullSecretConditionType))
	})
	t.Run(`Create AG capability (creation and deletion)`, func(t *testing.T) {
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
		r.connectionReconciler = createGenericReconcilerMock(t)
		r.versionReconciler = createVersionReconcilerMock(t)
		r.istioReconciler = createIstioReconcilerMock(t)
		r.pullSecretReconciler = createGenericReconcilerMock(t)

		err := r.Reconcile(context.Background())
		require.NoError(t, err)

		var service corev1.Service
		err = fakeClient.Get(context.Background(), types.NamespacedName{Name: testServiceName, Namespace: testNamespace}, &service)
		require.NoError(t, err)

		// remove AG from spec
		dk.Spec.ActiveGate = activegate.Spec{}
		r.connectionReconciler = createGenericReconcilerMock(t)
		r.versionReconciler = createVersionReconcilerMock(t)
		r.pullSecretReconciler = createGenericReconcilerMock(t)
		err = r.Reconcile(context.Background())
		require.NoError(t, err)

		err = fakeClient.Get(context.Background(), types.NamespacedName{Name: testServiceName, Namespace: testNamespace}, &service)
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
		fakeReconciler := controllermock.NewReconciler(t)
		fakeReconciler.On("Reconcile", mock.Anything).Return(nil)
		proxyReconciler := Reconciler{
			client:               fakeClient,
			apiReader:            fakeClient,
			dk:                   dkWithProxy,
			authTokenReconciler:  fakeReconciler,
			pullSecretReconciler: fakeReconciler,
			newStatefulsetReconcilerFunc: func(_ client.Client, _ client.Reader, _ *dynakube.DynaKube, _ capability.Capability) controllers.Reconciler {
				return fakeReconciler
			},
			newCapabilityReconcilerFunc: func(_ client.Client, _ capability.Capability, _ *dynakube.DynaKube, _ controllers.Reconciler, _ controllers.Reconciler) controllers.Reconciler {
				return fakeReconciler
			},
			newCustomPropertiesReconcilerFunc: func(_ string, customPropertiesSource *value.Source) controllers.Reconciler {
				return fakeReconciler
			},
			connectionReconciler: createGenericReconcilerMock(t),
			versionReconciler:    createVersionReconcilerMock(t),
		}
		err := proxyReconciler.Reconcile(context.Background())
		require.NoError(t, err)

		noProxyReconciler := Reconciler{
			client:               fakeClient,
			apiReader:            fakeClient,
			dk:                   dkNoProxy,
			authTokenReconciler:  fakeReconciler,
			pullSecretReconciler: fakeReconciler,
			newStatefulsetReconcilerFunc: func(_ client.Client, _ client.Reader, _ *dynakube.DynaKube, _ capability.Capability) controllers.Reconciler {
				return fakeReconciler
			},
			newCapabilityReconcilerFunc: func(_ client.Client, _ capability.Capability, _ *dynakube.DynaKube, _ controllers.Reconciler, _ controllers.Reconciler) controllers.Reconciler {
				return fakeReconciler
			},
			newCustomPropertiesReconcilerFunc: func(_ string, customPropertiesSource *value.Source) controllers.Reconciler {
				return fakeReconciler
			},
			connectionReconciler: createGenericReconcilerMock(t),
			versionReconciler:    createVersionReconcilerMock(t),
		}
		err = noProxyReconciler.Reconcile(context.Background())
		require.NoError(t, err)
	})
	t.Run(`Reconciles Kubernetes Monitoring`, func(t *testing.T) {
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
		r.connectionReconciler = createGenericReconcilerMock(t)
		r.versionReconciler = createVersionReconcilerMock(t)
		r.pullSecretReconciler = createGenericReconcilerMock(t)

		err := r.Reconcile(context.Background())
		require.NoError(t, err)

		require.NoError(t, err)

		var statefulSet appsv1.StatefulSet

		kubeMonCapability := capability.NewMultiCapability(dk)
		name := capability.CalculateStatefulSetName(kubeMonCapability, dk.Name)
		err = fakeClient.Get(context.Background(), client.ObjectKey{Name: name, Namespace: testNamespace}, &statefulSet)

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
				Extensions: dynakube.ExtensionsSpec{
					Enabled: false,
				},
			},
		}

		fakeClient := fake.NewClient(testKubeSystemNamespace)

		r := NewReconciler(fakeClient, fakeClient, instance, createMockDtClient(t, false), nil, nil).(*Reconciler)
		r.connectionReconciler = createGenericReconcilerMock(t)
		r.versionReconciler = createVersionReconcilerMock(t)
		r.pullSecretReconciler = createGenericReconcilerMock(t)

		err := r.Reconcile(context.Background())
		require.NoError(t, err)

		var service corev1.Service
		err = fakeClient.Get(context.Background(), types.NamespacedName{Name: testServiceName, Namespace: testNamespace}, &service)
		assert.True(t, k8serrors.IsNotFound(err))

		var statefulset appsv1.StatefulSet
		err = fakeClient.Get(context.Background(), types.NamespacedName{Name: instance.Name + "-activegate", Namespace: testNamespace}, &statefulset)
		assert.True(t, k8serrors.IsNotFound(err))
	})
	t.Run("activegate is created when extensions are enabled in dk, but no activegate is configured", func(t *testing.T) {
		instance := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: testNamespace,
				Name:      testName,
			},
			Spec: dynakube.DynaKubeSpec{
				Extensions: dynakube.ExtensionsSpec{
					Enabled: true,
				},
			},
		}

		fakeClient := fake.NewClient(testKubeSystemNamespace)

		r := NewReconciler(fakeClient, fakeClient, instance, createMockDtClient(t, true), nil, nil).(*Reconciler)
		r.connectionReconciler = createGenericReconcilerMock(t)
		r.versionReconciler = createVersionReconcilerMock(t)
		r.pullSecretReconciler = createGenericReconcilerMock(t)

		err := r.Reconcile(context.Background())
		require.NoError(t, err)

		var service corev1.Service
		err = fakeClient.Get(context.Background(), types.NamespacedName{Name: testServiceName, Namespace: testNamespace}, &service)
		require.NoError(t, err)
		require.Equal(t, testServiceName, service.Name)

		var statefulset appsv1.StatefulSet
		err = fakeClient.Get(context.Background(), types.NamespacedName{Name: instance.Name + "-activegate", Namespace: testNamespace}, &statefulset)
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
				Extensions: dynakube.ExtensionsSpec{
					Enabled: true,
				},
			},
		}

		fakeClient := fake.NewClient(testKubeSystemNamespace)

		r := NewReconciler(fakeClient, fakeClient, instance, createMockDtClient(t, true), nil, nil).(*Reconciler)
		r.connectionReconciler = createGenericReconcilerMock(t)
		r.versionReconciler = createVersionReconcilerMock(t)
		r.pullSecretReconciler = createGenericReconcilerMock(t)

		err := r.Reconcile(context.Background())
		require.NoError(t, err)

		var service corev1.Service
		err = fakeClient.Get(context.Background(), types.NamespacedName{Name: testServiceName, Namespace: testNamespace}, &service)
		require.NoError(t, err)
		require.Equal(t, testServiceName, service.Name)

		var statefulset appsv1.StatefulSet
		err = fakeClient.Get(context.Background(), types.NamespacedName{Name: instance.Name + "-activegate", Namespace: testNamespace}, &statefulset)
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
				Extensions: dynakube.ExtensionsSpec{
					Enabled: true,
				},
			},
		}

		fakeClient := fake.NewClient(testKubeSystemNamespace)

		r := NewReconciler(fakeClient, fakeClient, instance, createMockDtClient(t, true), nil, nil).(*Reconciler)
		r.connectionReconciler = createGenericReconcilerMock(t)
		r.versionReconciler = createVersionReconcilerMock(t)
		r.pullSecretReconciler = createGenericReconcilerMock(t)

		err := r.Reconcile(context.Background())
		require.NoError(t, err)

		var service corev1.Service
		err = fakeClient.Get(context.Background(), types.NamespacedName{Name: testServiceName, Namespace: testNamespace}, &service)
		require.NoError(t, err)
		require.Equal(t, testServiceName, service.Name)

		var statefulset appsv1.StatefulSet
		err = fakeClient.Get(context.Background(), types.NamespacedName{Name: instance.Name + "-activegate", Namespace: testNamespace}, &statefulset)
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
				Extensions: dynakube.ExtensionsSpec{
					Enabled: true,
				},
			},
		}

		fakeClient := fake.NewClient(testKubeSystemNamespace)

		r := NewReconciler(fakeClient, fakeClient, instance, createMockDtClient(t, true), nil, nil).(*Reconciler)
		r.connectionReconciler = createGenericReconcilerMock(t)
		r.versionReconciler = createVersionReconcilerMock(t)
		r.pullSecretReconciler = createGenericReconcilerMock(t)

		err := r.Reconcile(context.Background())
		require.NoError(t, err)

		var service corev1.Service
		err = fakeClient.Get(context.Background(), types.NamespacedName{Name: testServiceName, Namespace: testNamespace}, &service)
		require.NoError(t, err)
		require.Equal(t, testServiceName, service.Name)

		var statefulset appsv1.StatefulSet
		err = fakeClient.Get(context.Background(), types.NamespacedName{Name: instance.Name + "-activegate", Namespace: testNamespace}, &statefulset)
		require.NoError(t, err)

		// remove AG from spec
		r.dk.Spec.ActiveGate = activegate.Spec{}
		r.connectionReconciler = createGenericReconcilerMock(t)
		r.versionReconciler = createVersionReconcilerMock(t)
		r.pullSecretReconciler = createGenericReconcilerMock(t)
		err = r.Reconcile(context.Background())
		require.NoError(t, err)

		var service1 corev1.Service
		err = fakeClient.Get(context.Background(), types.NamespacedName{Name: testServiceName, Namespace: testNamespace}, &service1)
		require.NoError(t, err)
		require.Equal(t, testServiceName, service1.Name)

		err = fakeClient.Get(context.Background(), types.NamespacedName{Name: instance.Name + "-activegate", Namespace: testNamespace}, &statefulset)
		require.NoError(t, err)

		// disable extensions
		r.dk.Spec.Extensions.Enabled = false
		r.connectionReconciler = createGenericReconcilerMock(t)
		r.versionReconciler = createVersionReconcilerMock(t)
		r.pullSecretReconciler = createGenericReconcilerMock(t)
		err = r.Reconcile(context.Background())
		require.NoError(t, err)

		err = fakeClient.Get(context.Background(), types.NamespacedName{Name: testServiceName, Namespace: testNamespace}, &service)
		assert.True(t, k8serrors.IsNotFound(err))

		err = fakeClient.Get(context.Background(), types.NamespacedName{Name: instance.Name + "-activegate", Namespace: testNamespace}, &statefulset)
		assert.True(t, k8serrors.IsNotFound(err))
	})
}

func TestServiceCreation(t *testing.T) {
	dynatraceClient := dtclientmock.NewClient(t)
	dynatraceClient.On("GetActiveGateAuthToken", mock.AnythingOfType("context.backgroundCtx"), testName).Return(&dtclient.ActiveGateAuthTokenInfo{TokenId: "test", Token: "dt.some.valuegoeshere"}, nil)

	dk := &dynakube.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNamespace,
			Name:      testName,
		},
		Spec: dynakube.DynaKubeSpec{
			ActiveGate: activegate.Spec{},
		},
	}

	t.Run("service exposes correct ports for single capabilities", func(t *testing.T) {
		expectedCapabilityPorts := map[activegate.CapabilityDisplayName][]string{
			activegate.RoutingCapability.DisplayName: {
				consts.HttpsServicePortName,
			},
			activegate.MetricsIngestCapability.DisplayName: {
				consts.HttpsServicePortName,
				consts.HttpServicePortName,
			},
			activegate.DynatraceApiCapability.DisplayName: {
				consts.HttpsServicePortName,
			},
			activegate.KubeMonCapability.DisplayName: {},
		}

		for capName, expectedPorts := range expectedCapabilityPorts {
			fakeClient := fake.NewClient(testKubeSystemNamespace)

			reconciler := NewReconciler(fakeClient, fakeClient, dk, dynatraceClient, nil, nil).(*Reconciler)
			reconciler.connectionReconciler = createGenericReconcilerMock(t)
			reconciler.versionReconciler = createVersionReconcilerMock(t)
			reconciler.pullSecretReconciler = createGenericReconcilerMock(t)

			dk.Spec.ActiveGate.Capabilities = []activegate.CapabilityDisplayName{
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

		reconciler := NewReconciler(fakeClient, fakeClient, dk, dynatraceClient, nil, nil).(*Reconciler)
		reconciler.connectionReconciler = createGenericReconcilerMock(t)
		reconciler.versionReconciler = createVersionReconcilerMock(t)
		reconciler.pullSecretReconciler = createGenericReconcilerMock(t)

		dk.Spec.ActiveGate.Capabilities = []activegate.CapabilityDisplayName{
			activegate.RoutingCapability.DisplayName,
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

	t.Run(`create activegate ConfigMap`, func(t *testing.T) {
		fakeReconciler := controllermock.NewReconciler(t)
		fakeReconciler.On("Reconcile", mock.Anything).Return(nil)

		fakeClient := fake.NewClient(testKubeSystemNamespace)
		r := Reconciler{
			client:               fakeClient,
			apiReader:            fakeClient,
			dk:                   dk,
			authTokenReconciler:  fakeReconciler,
			pullSecretReconciler: fakeReconciler,
			newStatefulsetReconcilerFunc: func(_ client.Client, _ client.Reader, _ *dynakube.DynaKube, _ capability.Capability) controllers.Reconciler {
				return fakeReconciler
			},
			newCapabilityReconcilerFunc: func(_ client.Client, _ capability.Capability, _ *dynakube.DynaKube, _ controllers.Reconciler, _ controllers.Reconciler) controllers.Reconciler {
				return fakeReconciler
			},
			newCustomPropertiesReconcilerFunc: func(_ string, _ *value.Source) controllers.Reconciler {
				return fakeReconciler
			},
			connectionReconciler: createGenericReconcilerMock(t),
			versionReconciler:    createVersionReconcilerMock(t),
		}
		err := r.Reconcile(context.Background())
		require.NoError(t, err)

		var actual corev1.ConfigMap
		err = fakeClient.Get(context.TODO(), client.ObjectKey{Name: dk.ActiveGate().GetConnectionInfoConfigMapName(), Namespace: testNamespace}, &actual)
		require.NoError(t, err)
		assert.Equal(t, testTenantUUID, actual.Data[connectioninfo.TenantUUIDKey])
		assert.Equal(t, testTenantEndpoints, actual.Data[connectioninfo.CommunicationEndpointsKey])
	})
}

func createGenericReconcilerMock(t *testing.T) controllers.Reconciler {
	connectionInfoReconciler := controllermock.NewReconciler(t)
	connectionInfoReconciler.On("Reconcile",
		mock.AnythingOfType("context.backgroundCtx")).Return(nil).Once()

	return connectionInfoReconciler
}

func createVersionReconcilerMock(t *testing.T) version.Reconciler {
	versionReconciler := versionmock.NewReconciler(t)
	versionReconciler.On("ReconcileActiveGate",
		mock.AnythingOfType("context.backgroundCtx"),
		mock.AnythingOfType("*dynakube.DynaKube")).Return(nil).Once()

	return versionReconciler
}

func createIstioReconcilerMock(t *testing.T) istio.Reconciler {
	reconciler := istiomock.NewReconciler(t)
	reconciler.On("ReconcileActiveGateCommunicationHosts",
		mock.AnythingOfType("context.backgroundCtx"),
		mock.AnythingOfType("*dynakube.DynaKube")).Return(nil).Twice()

	return reconciler
}

func createMockDtClient(t *testing.T, authTokenRouteRequired bool) *dtclientmock.Client {
	dtc := dtclientmock.NewClient(t)
	if authTokenRouteRequired {
		dtc.On("GetActiveGateAuthToken", mock.AnythingOfType("context.backgroundCtx"), testName).Return(&dtclient.ActiveGateAuthTokenInfo{TokenId: "test", Token: "dt.some.valuegoeshere"}, nil)
	}

	return dtc
}
