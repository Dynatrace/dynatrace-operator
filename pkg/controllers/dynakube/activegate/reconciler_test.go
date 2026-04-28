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
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	agclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/activegate"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/capability"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/internal/statefulset"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/connectioninfo"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/token"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/version"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/objects/k8sconfigmap"
	agclientmock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/clients/dynatrace/activegate"
	versionmock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/controllers/dynakube/version"
	"github.com/pkg/errors"
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
	anyCtx                       = mock.MatchedBy(func(context.Context) bool { return true })
	anyDynakube                  = mock.MatchedBy(func(*dynakube.DynaKube) bool { return true })
	anyAgClient                  = mock.MatchedBy(func(apiClient agclient.Client) bool { return true })
	anyTokens                    = mock.MatchedBy(func(token.Tokens) bool { return true })
	anyCapability                = mock.MatchedBy(func(capability.Capability) bool { return true })
	anyCustomPropertiesOwnerName = mock.MatchedBy(func(string) bool { return true })
	anyCustomPropertiesSource    = mock.MatchedBy(func(*value.Source) bool { return true })

	testKubeSystemNamespace = &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "kube-system",
			UID:  "01234-5678-9012-3456",
		},
	}
)

func TestReconciler_Reconcile_Error(t *testing.T) {
	buildDynakube := func() *dynakube.DynaKube {
		return &dynakube.DynaKube{
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
	}
	t.Run("customPropertiesReconciler errors", func(t *testing.T) {
		expectErr := errors.New("customproperties error")

		clt := fake.NewClient()

		mockCustomPropertiesReconciler := newMockCustomPropertiesReconciler(t)
		mockCustomPropertiesReconciler.EXPECT().Reconcile(anyCtx, anyDynakube, anyCustomPropertiesOwnerName, anyCustomPropertiesSource).Return(expectErr).Once()

		r := &Reconciler{
			client:                     clt,
			apiReader:                  clt,
			authTokenReconciler:        mockAuthTokenReconcileOnce(t),
			istioReconciler:            createIstioReconcilerMock(t),
			connectionReconciler:       mockConnectionReconcileOnce(t),
			versionReconciler:          mockVersionReconcileOnce(t),
			pullSecretReconciler:       mockPullSecretReconcileOnce(t),
			customPropertiesReconciler: mockCustomPropertiesReconciler,
			configMaps:                 k8sconfigmap.Query(clt, clt, log),
		}

		err := r.Reconcile(t.Context(), buildDynakube(), createMockDTClient(t, false), nil)
		require.ErrorIs(t, err, expectErr)
	})
	t.Run("tlsReconciler errors", func(t *testing.T) {
		expectErr := errors.New("tls error")

		dk := buildDynakube()
		clt := fake.NewClient()

		tlsSecretReconciler := newMockTlsReconciler(t)
		tlsSecretReconciler.EXPECT().Reconcile(anyCtx, anyDynakube).Return(expectErr).Once()

		r := &Reconciler{
			client:                     clt,
			apiReader:                  clt,
			authTokenReconciler:        mockAuthTokenReconcileOnce(t),
			istioReconciler:            createIstioReconcilerMock(t),
			connectionReconciler:       mockConnectionReconcileOnce(t),
			versionReconciler:          mockVersionReconcileOnce(t),
			pullSecretReconciler:       mockPullSecretReconcileOnce(t),
			customPropertiesReconciler: mockCustomPropertiesReconcileOnce(t),
			tlsSecretReconciler:        tlsSecretReconciler,
			configMaps:                 k8sconfigmap.Query(clt, clt, log),
		}

		err := r.Reconcile(t.Context(), dk, createMockDTClient(t, false), nil)
		require.ErrorIs(t, err, expectErr)
	})
	t.Run("tlsReconciler errors", func(t *testing.T) {
		expectErr := errors.New("tls error")

		dk := buildDynakube()
		clt := fake.NewClient()

		statefulsetReconciler := newMockStatefulsetReconciler(t)
		statefulsetReconciler.EXPECT().Reconcile(anyCtx, anyDynakube, anyCapability).Return(expectErr).Once()

		r := &Reconciler{
			client:                     clt,
			apiReader:                  clt,
			authTokenReconciler:        mockAuthTokenReconcileOnce(t),
			istioReconciler:            createIstioReconcilerMock(t),
			connectionReconciler:       mockConnectionReconcileOnce(t),
			versionReconciler:          mockVersionReconcileOnce(t),
			pullSecretReconciler:       mockPullSecretReconcileOnce(t),
			customPropertiesReconciler: mockCustomPropertiesReconcileOnce(t),
			tlsSecretReconciler:        mockTLSReconcileOnce(t),
			statefulsetReconciler:      statefulsetReconciler,
			configMaps:                 k8sconfigmap.Query(clt, clt, log),
		}

		err := r.Reconcile(t.Context(), dk, createMockDTClient(t, false), nil)
		require.ErrorIs(t, err, expectErr)
	})
}

func TestReconciler_Reconcile(t *testing.T) {
	t.Run("No sub reconciler runs if AG was not enabled", func(t *testing.T) {
		dk := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: testNamespace,
				Name:      testName,
			}}
		// don't initialize the other fields to cause a panic if anything is accessed
		r := Reconciler{}

		err := r.Reconcile(t.Context(), dk, nil, nil)
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

		clt := fake.NewClient()

		r := &Reconciler{
			client:                     clt,
			apiReader:                  clt,
			authTokenReconciler:        mockAuthTokenReconcileOnce(t),
			istioReconciler:            createIstioReconcilerMock(t),
			connectionReconciler:       mockConnectionReconcileOnce(t),
			versionReconciler:          mockVersionReconcileOnce(t),
			pullSecretReconciler:       mockPullSecretReconcileOnce(t),
			statefulsetReconciler:      mockStatefulsetReconcileOnce(t),
			customPropertiesReconciler: mockCustomPropertiesReconcileOnce(t),
			tlsSecretReconciler:        mockTLSReconcileOnce(t),
			configMaps:                 k8sconfigmap.Query(clt, clt, log),
		}

		err := r.Reconcile(t.Context(), dk, createMockDTClient(t, false), nil)
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

		clt := fake.NewClient()

		r := Reconciler{
			client:               clt,
			apiReader:            clt,
			authTokenReconciler:  mockAuthTokenReconcileOnce(t),
			istioReconciler:      createIstioReconcilerMock(t),
			connectionReconciler: mockConnectionReconcileOnce(t),
			versionReconciler:    mockVersionReconcileOnce(t),
			pullSecretReconciler: mockPullSecretReconcileOnce(t),
			// statefulsetReconciler: panic if called
			// customPropertiesReconciler: panic if called
			tlsSecretReconciler: mockTLSReconcileOnce(t),
			configMaps:          k8sconfigmap.Query(clt, clt, log),
		}

		err := r.Reconcile(t.Context(), dk, createMockDTClient(t, false), nil)
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

		r := NewReconciler(fakeClient, fakeClient)
		r.connectionReconciler = mockConnectionReconcileOnce(t)
		r.versionReconciler = mockVersionReconcileOnce(t)
		r.istioReconciler = createIstioReconcilerMock(t)
		r.pullSecretReconciler = mockPullSecretReconcileOnce(t)

		err := r.Reconcile(t.Context(), dk, createMockDTClient(t, true), nil)
		require.NoError(t, err)

		var service corev1.Service
		err = fakeClient.Get(t.Context(), types.NamespacedName{Name: testServiceName, Namespace: testNamespace}, &service)
		require.NoError(t, err)

		var secret corev1.Secret
		err = fakeClient.Get(t.Context(), types.NamespacedName{Name: dk.ActiveGate().GetTLSSecretName(), Namespace: testNamespace}, &secret)
		require.NoError(t, err)

		// remove AG from spec
		dk.Spec.ActiveGate = activegate.Spec{}
		r.connectionReconciler = mockConnectionReconcileOnce(t)
		r.versionReconciler = mockVersionReconcileOnce(t)
		r.istioReconciler = createIstioReconcilerMock(t)
		r.pullSecretReconciler = mockPullSecretReconcileOnce(t)
		err = r.Reconcile(t.Context(), dk, createMockDTClient(t, false), nil)
		require.NoError(t, err)

		err = fakeClient.Get(t.Context(), types.NamespacedName{Name: testServiceName, Namespace: testNamespace}, &service)
		assert.True(t, k8serrors.IsNotFound(err))

		err = fakeClient.Get(t.Context(), types.NamespacedName{Name: dk.ActiveGate().GetTLSSecretName(), Namespace: testNamespace}, &secret)
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
			client:                     fakeClient,
			apiReader:                  fakeClient,
			authTokenReconciler:        mockAuthTokenReconcileOnce(t),
			pullSecretReconciler:       mockPullSecretReconcileOnce(t),
			connectionReconciler:       mockConnectionReconcileOnce(t),
			versionReconciler:          mockVersionReconcileOnce(t),
			istioReconciler:            createIstioReconcilerMock(t),
			statefulsetReconciler:      mockStatefulsetReconcileOnce(t),
			customPropertiesReconciler: mockCustomPropertiesReconcileOnce(t),
			tlsSecretReconciler:        mockTLSReconcileOnce(t),
			configMaps:                 k8sconfigmap.Query(fakeClient, fakeClient, log),
		}
		err := proxyReconciler.Reconcile(t.Context(), dkWithProxy, createMockDTClient(t, false), nil)
		require.NoError(t, err)

		noProxyReconciler := &Reconciler{
			client:                     fakeClient,
			apiReader:                  fakeClient,
			authTokenReconciler:        mockAuthTokenReconcileOnce(t),
			pullSecretReconciler:       mockPullSecretReconcileOnce(t),
			connectionReconciler:       mockConnectionReconcileOnce(t),
			versionReconciler:          mockVersionReconcileOnce(t),
			istioReconciler:            createIstioReconcilerMock(t),
			statefulsetReconciler:      mockStatefulsetReconcileOnce(t),
			customPropertiesReconciler: mockCustomPropertiesReconcileOnce(t),
			tlsSecretReconciler:        mockTLSReconcileOnce(t),
			configMaps:                 k8sconfigmap.Query(fakeClient, fakeClient, log),
		}

		err = noProxyReconciler.Reconcile(t.Context(), dkNoProxy, createMockDTClient(t, false), nil)
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

		r := NewReconciler(fakeClient, fakeClient)
		r.connectionReconciler = mockConnectionReconcileOnce(t)
		r.versionReconciler = mockVersionReconcileOnce(t)
		r.pullSecretReconciler = mockPullSecretReconcileOnce(t)
		r.istioReconciler = createIstioReconcilerMock(t)

		err := r.Reconcile(t.Context(), dk, createMockDTClient(t, true), nil)
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

		r := NewReconciler(fakeClient, fakeClient)

		err := r.Reconcile(t.Context(), instance, nil, nil)
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

		r := NewReconciler(fakeClient, fakeClient)
		r.connectionReconciler = mockConnectionReconcileOnce(t)
		r.versionReconciler = mockVersionReconcileOnce(t)
		r.pullSecretReconciler = mockPullSecretReconcileOnce(t)
		r.istioReconciler = createIstioReconcilerMock(t)

		err := r.Reconcile(t.Context(), instance, createMockDTClient(t, true), nil)
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

		r := NewReconciler(fakeClient, fakeClient)
		r.connectionReconciler = mockConnectionReconcileOnce(t)
		r.versionReconciler = mockVersionReconcileOnce(t)
		r.pullSecretReconciler = mockPullSecretReconcileOnce(t)
		r.istioReconciler = createIstioReconcilerMock(t)

		err := r.Reconcile(t.Context(), instance, createMockDTClient(t, true), nil)
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

		r := NewReconciler(fakeClient, fakeClient)
		r.connectionReconciler = mockConnectionReconcileOnce(t)
		r.versionReconciler = mockVersionReconcileOnce(t)
		r.pullSecretReconciler = mockPullSecretReconcileOnce(t)
		r.istioReconciler = createIstioReconcilerMock(t)

		err := r.Reconcile(t.Context(), instance, createMockDTClient(t, true), nil)
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

		r := NewReconciler(fakeClient, fakeClient)
		r.connectionReconciler = mockConnectionReconcileOnce(t)
		r.versionReconciler = mockVersionReconcileOnce(t)
		r.pullSecretReconciler = mockPullSecretReconcileOnce(t)
		r.istioReconciler = createIstioReconcilerMock(t)

		err := r.Reconcile(t.Context(), instance, createMockDTClient(t, true), nil)
		require.NoError(t, err)

		var service corev1.Service
		err = fakeClient.Get(t.Context(), types.NamespacedName{Name: testServiceName, Namespace: testNamespace}, &service)
		require.NoError(t, err)
		require.Equal(t, testServiceName, service.Name)

		var statefulset appsv1.StatefulSet
		err = fakeClient.Get(t.Context(), types.NamespacedName{Name: instance.Name + "-activegate", Namespace: testNamespace}, &statefulset)
		require.NoError(t, err)

		// remove AG from spec
		instance.Spec.ActiveGate = activegate.Spec{}
		r.connectionReconciler = mockConnectionReconcileOnce(t)
		r.versionReconciler = mockVersionReconcileOnce(t)
		r.pullSecretReconciler = mockPullSecretReconcileOnce(t)
		r.istioReconciler = createIstioReconcilerMock(t)

		err = r.Reconcile(t.Context(), instance, createMockDTClient(t, true), nil)
		require.NoError(t, err)

		var service1 corev1.Service
		err = fakeClient.Get(t.Context(), types.NamespacedName{Name: testServiceName, Namespace: testNamespace}, &service1)
		require.NoError(t, err)
		require.Equal(t, testServiceName, service1.Name)

		err = fakeClient.Get(t.Context(), types.NamespacedName{Name: instance.Name + "-activegate", Namespace: testNamespace}, &statefulset)
		require.NoError(t, err)

		// disable extensions
		instance.Spec.Extensions = nil
		r.connectionReconciler = mockConnectionReconcileOnce(t)
		r.versionReconciler = mockVersionReconcileOnce(t)
		r.pullSecretReconciler = mockPullSecretReconcileOnce(t)
		r.istioReconciler = createIstioReconcilerMock(t)

		err = r.Reconcile(t.Context(), instance, createMockDTClient(t, true), nil)
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

	dtClient := createMockDTClient(t, true)

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

			reconciler := NewReconciler(fakeClient, fakeClient)
			reconciler.connectionReconciler = mockConnectionReconcileOnce(t)
			reconciler.versionReconciler = mockVersionReconcileOnce(t)
			reconciler.pullSecretReconciler = mockPullSecretReconcileOnce(t)
			reconciler.istioReconciler = createIstioReconcilerMock(t)

			dk.Spec.ActiveGate.Capabilities = []activegate.CapabilityDisplayName{
				capName,
			}

			err := reconciler.Reconcile(t.Context(), dk, dtClient, nil)
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
			client:                     fakeClient,
			apiReader:                  fakeClient,
			authTokenReconciler:        mockAuthTokenReconcileOnce(t),
			pullSecretReconciler:       mockPullSecretReconcileOnce(t),
			connectionReconciler:       mockConnectionReconcileOnce(t),
			versionReconciler:          mockVersionReconcileOnce(t),
			istioReconciler:            createIstioReconcilerMock(t),
			statefulsetReconciler:      mockStatefulsetReconcileOnce(t),
			customPropertiesReconciler: mockCustomPropertiesReconcileOnce(t),
			tlsSecretReconciler:        mockTLSReconcileOnce(t),
			configMaps:                 k8sconfigmap.Query(fakeClient, fakeClient, log),
		}

		err := r.Reconcile(t.Context(), dk, createMockDTClient(t, true), nil)
		require.NoError(t, err)

		var actual corev1.ConfigMap
		err = fakeClient.Get(t.Context(), client.ObjectKey{Name: dk.ActiveGate().GetConnectionInfoConfigMapName(), Namespace: testNamespace}, &actual)
		require.NoError(t, err)
		assert.Equal(t, testTenantUUID, actual.Data[connectioninfo.TenantUUIDKey])
		assert.Equal(t, testTenantEndpoints, actual.Data[connectioninfo.CommunicationEndpointsKey])
	})
}

func mockAuthTokenReconcileOnce(t *testing.T) authTokenReconciler {
	t.Helper()

	reconciler := newMockAuthTokenReconciler(t)
	reconciler.EXPECT().Reconcile(anyCtx, anyAgClient, anyDynakube).Return(nil).Once()

	return reconciler
}

func mockConnectionReconcileOnce(t *testing.T) connectionReconciler {
	t.Helper()

	reconciler := newMockConnectionReconciler(t)
	reconciler.EXPECT().Reconcile(anyCtx, anyAgClient, anyDynakube).Return(nil).Once()

	return reconciler
}

func mockPullSecretReconcileOnce(t *testing.T) pullSecretReconciler {
	t.Helper()

	reconciler := newMockPullSecretReconciler(t)
	reconciler.EXPECT().Reconcile(anyCtx, anyDynakube, anyTokens).Return(nil).Once()

	return reconciler
}

func mockStatefulsetReconcileOnce(t *testing.T) statefulsetReconciler {
	t.Helper()

	reconciler := newMockStatefulsetReconciler(t)
	reconciler.EXPECT().Reconcile(anyCtx, anyDynakube, anyCapability).Return(nil).Once()

	return reconciler
}

func mockCustomPropertiesReconcileOnce(t *testing.T) customPropertiesReconciler {
	t.Helper()

	reconciler := newMockCustomPropertiesReconciler(t)
	reconciler.EXPECT().Reconcile(anyCtx, anyDynakube, anyCustomPropertiesOwnerName, anyCustomPropertiesSource).Return(nil).Once()

	return reconciler
}

func mockTLSReconcileOnce(t *testing.T) tlsReconciler {
	t.Helper()

	reconciler := newMockTlsReconciler(t)
	reconciler.EXPECT().Reconcile(anyCtx, anyDynakube).Return(nil).Once()

	return reconciler
}

func mockVersionReconcileOnce(t *testing.T) version.Reconciler {
	t.Helper()

	versionReconciler := versionmock.NewReconciler(t)
	versionReconciler.EXPECT().ReconcileActiveGate(anyCtx, anyDynakube).Return(nil).Once()

	return versionReconciler
}

func createIstioReconcilerMock(t *testing.T) istioReconciler {
	rec := newMockIstioReconciler(t)

	rec.EXPECT().ReconcileActiveGate(t.Context(), anyDynakube).Return(nil).Once()

	return rec
}

func createMockDTClient(t *testing.T, authTokenRouteRequired bool) *dynatrace.Client {
	t.Helper()

	agClient := agclientmock.NewClient(t)

	if authTokenRouteRequired {
		agClient.EXPECT().GetAuthToken(anyCtx, testName).Return(&agclient.AuthTokenInfo{TokenID: "test", Token: "dt.some.valuegoeshere"}, nil).Maybe()
	}

	return &dynatrace.Client{ActiveGate: agClient}
}
