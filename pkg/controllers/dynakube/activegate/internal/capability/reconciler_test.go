package capability

import (
	"context"
	"errors"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/activegate"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/capability"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/internal/authtoken"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/system"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	testToken    = "dt.testtoken.test"
	testUID      = "test-uid"
	testDynakube = "test-dynakube"
)

var (
	anyCtx      = mock.MatchedBy(func(context.Context) bool { return true })
	anyDynakube = mock.MatchedBy(func(*dynakube.DynaKube) bool { return true })
)

var capabilitiesWithService = []activegate.CapabilityDisplayName{
	activegate.RoutingCapability.DisplayName,
	activegate.KubeMonCapability.DisplayName,
	activegate.MetricsIngestCapability.DisplayName,
	activegate.DynatraceAPICapability.DisplayName,
}

var capabilitiesWithoutService = []activegate.CapabilityDisplayName{
	activegate.KubeMonCapability.DisplayName,
}

func createClient() client.WithWatch {
	return fake.NewClientBuilder().
		WithScheme(scheme.Scheme).
		WithObjects(&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: system.Namespace,
				UID:  testUID,
			},
		}).
		WithObjects(&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      activegate.AuthTokenSecretSuffix,
				Namespace: testNamespace,
			},
			Data: map[string][]byte{authtoken.ActiveGateAuthTokenName: []byte(testToken)},
		}).
		Build()
}

func buildDynakube(capabilities []activegate.CapabilityDisplayName) *dynakube.DynaKube {
	return &dynakube.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNamespace,
			Name:      testDynakube,
		},
		Spec: dynakube.DynaKubeSpec{
			APIURL: testAPIURL,
			ActiveGate: activegate.Spec{
				Capabilities: capabilities,
			},
		},
	}
}

func getMockReconciler(t *testing.T, returnValue error) *mockDynakubeReconciler {
	mockReconciler := newMockDynakubeReconciler(t)
	mockReconciler.EXPECT().Reconcile(anyCtx, anyDynakube).Return(returnValue).Once()

	return mockReconciler
}

func TestNewReconciler(t *testing.T) {
	r := NewReconciler(
		fake.NewClientBuilder().Build(),
		capability.NewMultiCapability(&dynakube.DynaKube{}),
		newMockDynakubeReconciler(t),
		newMockDynakubeReconciler(t),
		newMockDynakubeReconciler(t),
	)
	require.NotNil(t, r)
	require.NotNil(t, r.client)
	require.NotNil(t, r)
}

func TestReconcile(t *testing.T) {
	clt := createClient()

	t.Run("statefulSetReconciler errors", func(t *testing.T) {
		expectErr := errors.New("statefulset error")

		dk := buildDynakube(capabilitiesWithoutService)
		mockCustompropertiesReconciler := getMockReconciler(t, nil)
		mockStatefulSetReconciler := getMockReconciler(t, expectErr)
		mockTLSSecretReconciler := getMockReconciler(t, nil)

		r := NewReconciler(clt, capability.NewMultiCapability(dk), mockStatefulSetReconciler, mockCustompropertiesReconciler, mockTLSSecretReconciler)

		err := r.Reconcile(t.Context(), dk)
		require.ErrorIs(t, err, expectErr)
	})
	t.Run("customPropertiesReconciler errors", func(t *testing.T) {
		expectErr := errors.New("customproperties error")

		dk := buildDynakube(capabilitiesWithoutService)
		mockCustompropertiesReconciler := getMockReconciler(t, expectErr)
		mockStatefulSetReconciler := newMockDynakubeReconciler(t)
		mockTLSSecretReconciler := newMockDynakubeReconciler(t)

		r := NewReconciler(clt, capability.NewMultiCapability(dk), mockStatefulSetReconciler, mockCustompropertiesReconciler, mockTLSSecretReconciler)

		err := r.Reconcile(t.Context(), dk)
		require.ErrorIs(t, err, expectErr)
	})
	t.Run("service gets created", func(t *testing.T) {
		dk := buildDynakube(capabilitiesWithService)
		mockStatefulSetReconciler := getMockReconciler(t, nil)
		mockCustompropertiesReconciler := getMockReconciler(t, nil)
		mockTLSSecretReconciler := getMockReconciler(t, nil)

		r := NewReconciler(clt, capability.NewMultiCapability(dk), mockStatefulSetReconciler, mockCustompropertiesReconciler, mockTLSSecretReconciler)

		err := r.Reconcile(t.Context(), dk)
		require.NoError(t, err)

		service := corev1.Service{}
		err = clt.Get(t.Context(), client.ObjectKey{Name: capability.BuildServiceName(dk.Name), Namespace: dk.Namespace}, &service)

		require.NoError(t, err)
		assert.NotNil(t, service)
	})
	t.Run("service is created even though capability does not need it", func(t *testing.T) {
		clt := createClient()
		dk := buildDynakube(capabilitiesWithoutService)
		mockStatefulSetReconciler := getMockReconciler(t, nil)
		mockCustompropertiesReconciler := getMockReconciler(t, nil)
		mockTLSSecretReconciler := getMockReconciler(t, nil)

		r := NewReconciler(clt, capability.NewMultiCapability(dk), mockStatefulSetReconciler, mockCustompropertiesReconciler, mockTLSSecretReconciler)

		require.NoError(t, r.Reconcile(t.Context(), dk))

		service := corev1.Service{}
		err := clt.Get(t.Context(), client.ObjectKey{Name: capability.BuildServiceName(dk.Name), Namespace: dk.Namespace}, &service)

		require.NoError(t, err)

		assert.NotEmpty(t, service)
		assert.Len(t, service.Spec.Ports, 2)
	})
}

func TestCreateOrUpdateService(t *testing.T) {
	dk := buildDynakube(capabilitiesWithService)

	getService := func(t *testing.T, clt client.Client) *corev1.Service {
		t.Helper()
		service := &corev1.Service{}
		err := clt.Get(t.Context(), client.ObjectKey{Name: capability.BuildServiceName(dk.Name), Namespace: dk.Namespace}, service)
		require.NoError(t, err)

		return service
	}

	tests := []struct {
		name   string
		mutate func(*corev1.Service)
	}{
		{
			"ports get updated",
			func(svc *corev1.Service) {
				svc.Spec.Ports = []corev1.ServicePort{}
			},
		},
		{
			"labels get updated",
			func(svc *corev1.Service) {
				svc.Labels = map[string]string{}
			},
		},
		{
			"selector gets updated",
			func(svc *corev1.Service) {
				svc.Spec.Selector = map[string]string{}
			},
		},
	}

	for _, test := range tests {
		clt := createClient()
		r := &Reconciler{client: clt, capability: capability.NewMultiCapability(dk)}

		err := r.createOrUpdateService(t.Context(), dk)
		require.NoError(t, err)

		service := &corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: capability.BuildServiceName(dk.Name), Namespace: dk.Namespace}}
		result, err := controllerutil.CreateOrUpdate(t.Context(), clt, service, func() error {
			test.mutate(service)

			return nil
		})
		require.NoError(t, err)
		require.Equal(t, controllerutil.OperationResultUpdated, result)

		err = r.createOrUpdateService(t.Context(), dk)
		require.NoError(t, err)

		actualService := getService(t, clt)
		desiredService := CreateService(dk)
		assert.Equal(t, desiredService.Labels, actualService.Labels)
		assert.Equal(t, desiredService.Spec, actualService.Spec)
		assert.NotEqual(t, actualService, service)
	}
}

func TestSetAGServiceIPs(t *testing.T) {
	t.Run("sets ServiceIPs from existing service ClusterIPs", func(t *testing.T) {
		dk := buildDynakube(capabilitiesWithService)
		expectedIPs := []string{"10.0.0.1", "fd00::1"}

		svc := CreateService(dk)
		svc.Spec.ClusterIPs = expectedIPs

		clt := fake.NewClientBuilder().
			WithScheme(scheme.Scheme).
			WithObjects(svc).
			Build()

		r := &Reconciler{client: clt, dk: dk}

		err := r.setAGServiceIPs(t.Context())
		require.NoError(t, err)
		assert.Equal(t, expectedIPs, dk.Status.ActiveGate.ServiceIPs)
	})

	t.Run("returns error when service does not exist", func(t *testing.T) {
		dk := buildDynakube(capabilitiesWithService)

		clt := fake.NewClientBuilder().
			WithScheme(scheme.Scheme).
			Build()

		r := &Reconciler{client: clt, dk: dk}

		err := r.setAGServiceIPs(t.Context())
		require.Error(t, err)
		assert.True(t, k8serrors.IsNotFound(err))
	})

	t.Run("retry if not there", func(t *testing.T) {
		dk := buildDynakube(capabilitiesWithService)
		expectedIPs := []string{"10.0.0.1", "fd00::1"}

		svc := CreateService(dk)
		svc.Spec.ClusterIPs = expectedIPs
		expectedAttempts := 2
		attemptCounter := 0

		clt := fake.NewClientBuilder().
			WithScheme(scheme.Scheme).
			WithInterceptorFuncs(interceptor.Funcs{
				Get: func(ctx context.Context, client client.WithWatch, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
					if attemptCounter < expectedAttempts {
						attemptCounter++

						return k8serrors.NewNotFound(schema.GroupResource{}, "test")
					}
					svc.DeepCopyInto(obj.(*corev1.Service))

					return nil
				},
				List: func(ctx context.Context, client client.WithWatch, list client.ObjectList, opts ...client.ListOption) error {
					return errors.New("UNEXPECTED")
				},
				Create: func(ctx context.Context, client client.WithWatch, obj client.Object, opts ...client.CreateOption) error {
					return errors.New("UNEXPECTED")
				},
				Delete: func(ctx context.Context, client client.WithWatch, obj client.Object, opts ...client.DeleteOption) error {
					return errors.New("UNEXPECTED")
				},
				Update: func(ctx context.Context, client client.WithWatch, obj client.Object, opts ...client.UpdateOption) error {
					return errors.New("UNEXPECTED")
				},
			}).
			Build()

		r := &Reconciler{client: clt, dk: dk}

		err := r.setAGServiceIPs(t.Context())
		require.NoError(t, err)
		assert.Equal(t, expectedIPs, dk.Status.ActiveGate.ServiceIPs)
		assert.Equal(t, expectedAttempts, attemptCounter)
	})

	t.Run("clears ServiceIPs when service has no ClusterIPs", func(t *testing.T) {
		dk := buildDynakube(capabilitiesWithService)
		dk.Status.ActiveGate.ServiceIPs = []string{"10.0.0.1"}

		svc := CreateService(dk)
		// ClusterIPs intentionally left empty

		clt := fake.NewClientBuilder().
			WithScheme(scheme.Scheme).
			WithObjects(svc).
			Build()

		r := &Reconciler{client: clt, dk: dk}

		err := r.setAGServiceIPs(t.Context())
		require.NoError(t, err)
		assert.Empty(t, dk.Status.ActiveGate.ServiceIPs)
	})
}
