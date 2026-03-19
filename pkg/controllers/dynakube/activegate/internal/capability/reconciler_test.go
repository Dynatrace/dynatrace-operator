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
	reconcilermock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/controllers/dynakube/activegate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	testToken    = "dt.testtoken.test"
	testUID      = "test-uid"
	testDynakube = "test-dynakube"
)

var anyCtx = mock.MatchedBy(func(context.Context) bool { return true })

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

func getMockReconciler(t *testing.T, returnValue error) *reconcilermock.CapabilityReconciler {
	mockReconciler := reconcilermock.NewCapabilityReconciler(t)
	mockReconciler.EXPECT().Reconcile(anyCtx).Return(returnValue).Once()

	return mockReconciler
}

func TestNewReconciler(t *testing.T) {
	r := NewReconciler(
		fake.NewClientBuilder().Build(),
		capability.NewMultiCapability(&dynakube.DynaKube{}),
		&dynakube.DynaKube{},
		reconcilermock.NewCapabilityReconciler(t),
		reconcilermock.NewCapabilityReconciler(t),
		reconcilermock.NewCapabilityReconciler(t),
	).(*Reconciler)
	require.NotNil(t, r)
	require.NotNil(t, r.client)
	require.NotNil(t, r)
	require.NotNil(t, r.dk)
}

func TestReconcile(t *testing.T) {
	clt := createClient()

	t.Run("statefulSetReconciler errors", func(t *testing.T) {
		expectErr := errors.New("statefulset error")

		dk := buildDynakube(capabilitiesWithoutService)
		mockCustompropertiesReconciler := getMockReconciler(t, nil)
		mockStatefulSetReconciler := getMockReconciler(t, expectErr)
		mockTLSSecretReconciler := getMockReconciler(t, nil)

		r := NewReconciler(clt, capability.NewMultiCapability(dk), dk, mockStatefulSetReconciler, mockCustompropertiesReconciler, mockTLSSecretReconciler)

		err := r.Reconcile(t.Context())
		require.ErrorIs(t, err, expectErr)
	})
	t.Run("customPropertiesReconciler errors", func(t *testing.T) {
		expectErr := errors.New("customproperties error")

		dk := buildDynakube(capabilitiesWithoutService)
		mockCustompropertiesReconciler := getMockReconciler(t, expectErr)
		mockStatefulSetReconciler := reconcilermock.NewCapabilityReconciler(t)
		mockTLSSecretReconciler := reconcilermock.NewCapabilityReconciler(t)

		r := NewReconciler(clt, capability.NewMultiCapability(dk), dk, mockStatefulSetReconciler, mockCustompropertiesReconciler, mockTLSSecretReconciler)

		err := r.Reconcile(t.Context())
		require.ErrorIs(t, err, expectErr)
	})
	t.Run("service gets created", func(t *testing.T) {
		dk := buildDynakube(capabilitiesWithService)
		mockStatefulSetReconciler := getMockReconciler(t, nil)
		mockCustompropertiesReconciler := getMockReconciler(t, nil)
		mockTLSSecretReconciler := getMockReconciler(t, nil)

		r := NewReconciler(clt, capability.NewMultiCapability(dk), dk, mockStatefulSetReconciler, mockCustompropertiesReconciler, mockTLSSecretReconciler)

		err := r.Reconcile(t.Context())
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

		r := NewReconciler(clt, capability.NewMultiCapability(dk), dk, mockStatefulSetReconciler, mockCustompropertiesReconciler, mockTLSSecretReconciler)

		require.NoError(t, r.Reconcile(t.Context()))

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
		r := &Reconciler{client: clt, capability: capability.NewMultiCapability(dk), dk: dk}

		err := r.createOrUpdateService(t.Context())
		require.NoError(t, err)

		service := &corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: capability.BuildServiceName(dk.Name), Namespace: dk.Namespace}}
		result, err := controllerutil.CreateOrUpdate(t.Context(), clt, service, func() error {
			test.mutate(service)

			return nil
		})
		require.NoError(t, err)
		require.Equal(t, controllerutil.OperationResultUpdated, result)

		err = r.createOrUpdateService(t.Context())
		require.NoError(t, err)

		actualService := getService(t, clt)
		desiredService := CreateService(dk)
		assert.Equal(t, desiredService.Labels, actualService.Labels)
		assert.Equal(t, desiredService.Spec, actualService.Spec)
		assert.NotEqual(t, actualService, service)
	}
}
