package capability

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme"
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/capability"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/internal/authtoken"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubesystem"
	reconcilermock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/controllers/dynakube/activegate"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	testToken    = "dt.testtoken.test"
	testUID      = "test-uid"
	testDynakube = "test-dynakube"
)

var capabilitiesWithService = []dynatracev1beta1.CapabilityDisplayName{
	dynatracev1beta1.RoutingCapability.DisplayName,
	dynatracev1beta1.KubeMonCapability.DisplayName,
	dynatracev1beta1.MetricsIngestCapability.DisplayName,
	dynatracev1beta1.DynatraceApiCapability.DisplayName,
}

var capabilitiesWithoutService = []dynatracev1beta1.CapabilityDisplayName{
	dynatracev1beta1.KubeMonCapability.DisplayName,
}

func createClient() client.WithWatch {
	return fake.NewClientBuilder().
		WithScheme(scheme.Scheme).
		WithObjects(&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: kubesystem.Namespace,
				UID:  testUID,
			},
		}).
		WithObjects(&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      dynatracev1beta1.AuthTokenSecretSuffix,
				Namespace: testNamespace,
			},
			Data: map[string][]byte{authtoken.ActiveGateAuthTokenName: []byte(testToken)},
		}).
		Build()
}

func buildDynakube(capabilities []dynatracev1beta1.CapabilityDisplayName) *dynatracev1beta1.DynaKube {
	return &dynatracev1beta1.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNamespace,
			Name:      testDynakube,
		},
		Spec: dynatracev1beta1.DynaKubeSpec{
			APIURL: testApiUrl,
			ActiveGate: dynatracev1beta1.ActiveGateSpec{
				Capabilities: capabilities,
			},
		},
	}
}

func getMockReconciler(t *testing.T, returnArguments ...any) *reconcilermock.CapabilityReconciler {
	mockReconciler := reconcilermock.NewCapabilityReconciler(t)
	mockReconciler.On("Reconcile", mock.Anything).Return(returnArguments...).Maybe()

	return mockReconciler
}

func verifyReconciler(t *testing.T, r *Reconciler) {
	require.NotNil(t, r)
	require.NotNil(t, r.client)
	require.NotNil(t, r)
	require.NotNil(t, r.dynakube)
}

func TestReconcile(t *testing.T) {
	clt := createClient()

	t.Run(`reconciler works with multiple capabilities`, func(t *testing.T) {
		dynakube := buildDynakube(capabilitiesWithService)
		mockStatefulSetReconciler := getMockReconciler(t, nil)
		mockCustompropertiesReconciler := getMockReconciler(t, nil)

		r := NewReconciler(clt, capability.NewMultiCapability(dynakube), dynakube, mockStatefulSetReconciler, mockCustompropertiesReconciler).(*Reconciler)
		verifyReconciler(t, r)

		err := r.Reconcile(context.Background())

		mockStatefulSetReconciler.AssertCalled(t, "Reconcile", mock.Anything)
		mockCustompropertiesReconciler.AssertCalled(t, "Reconcile", mock.Anything)
		require.NoError(t, err)
	})
	t.Run(`statefulSetReconciler errors`, func(t *testing.T) {
		dynakube := buildDynakube(capabilitiesWithoutService)
		mockStatefulSetReconciler := getMockReconciler(t, errors.New(""))
		mockCustompropertiesReconciler := getMockReconciler(t, nil)

		r := NewReconciler(clt, capability.NewMultiCapability(dynakube), dynakube, mockStatefulSetReconciler, mockCustompropertiesReconciler).(*Reconciler)
		verifyReconciler(t, r)

		err := r.Reconcile(context.Background())

		mockStatefulSetReconciler.AssertCalled(t, "Reconcile", mock.Anything)
		mockCustompropertiesReconciler.AssertCalled(t, "Reconcile", mock.Anything)
		require.Error(t, err)
	})
	t.Run(`customPropertiesReconciler errors`, func(t *testing.T) {
		dynakube := buildDynakube(capabilitiesWithoutService)
		mockStatefulSetReconciler := getMockReconciler(t)
		mockCustompropertiesReconciler := getMockReconciler(t, errors.New(""))

		r := NewReconciler(clt, capability.NewMultiCapability(dynakube), dynakube, mockStatefulSetReconciler, mockCustompropertiesReconciler).(*Reconciler)
		verifyReconciler(t, r)

		err := r.Reconcile(context.Background())

		mockCustompropertiesReconciler.AssertCalled(t, "Reconcile", mock.Anything)
		require.Error(t, err)
	})
	t.Run(`statefulSetReconciler and customPropertiesReconciler error`, func(t *testing.T) {
		dynakube := buildDynakube(capabilitiesWithoutService)
		mockStatefulSetReconciler := getMockReconciler(t, errors.New(""))
		mockCustompropertiesReconciler := getMockReconciler(t, errors.New(""))

		r := NewReconciler(clt, capability.NewMultiCapability(dynakube), dynakube, mockStatefulSetReconciler, mockCustompropertiesReconciler).(*Reconciler)
		verifyReconciler(t, r)

		err := r.Reconcile(context.Background())
		require.Error(t, err)
	})
	t.Run(`service gets created`, func(t *testing.T) {
		dynakube := buildDynakube(capabilitiesWithService)
		mockStatefulSetReconciler := getMockReconciler(t, nil)
		mockCustompropertiesReconciler := getMockReconciler(t, nil)

		r := NewReconciler(clt, capability.NewMultiCapability(dynakube), dynakube, mockStatefulSetReconciler, mockCustompropertiesReconciler).(*Reconciler)
		verifyReconciler(t, r)

		err := r.Reconcile(context.Background())

		mockStatefulSetReconciler.AssertCalled(t, "Reconcile", mock.Anything)
		mockCustompropertiesReconciler.AssertCalled(t, "Reconcile", mock.Anything)
		require.NoError(t, err)

		service := corev1.Service{}
		err = r.client.Get(context.Background(), client.ObjectKey{Name: r.dynakube.Name + "-" + r.capability.ShortName(), Namespace: r.dynakube.Namespace}, &service)

		assert.NotNil(t, service)
		require.NoError(t, err)
	})
	t.Run(`service does not get created when missing capabilities`, func(t *testing.T) {
		clt := createClient()
		dynakube := buildDynakube(capabilitiesWithoutService)
		mockStatefulSetReconciler := getMockReconciler(t, nil)
		mockCustompropertiesReconciler := getMockReconciler(t, nil)

		r := NewReconciler(clt, capability.NewMultiCapability(dynakube), dynakube, mockStatefulSetReconciler, mockCustompropertiesReconciler).(*Reconciler)
		verifyReconciler(t, r)

		err := r.Reconcile(context.Background())

		mockStatefulSetReconciler.AssertCalled(t, "Reconcile", mock.Anything)
		mockCustompropertiesReconciler.AssertCalled(t, "Reconcile", mock.Anything)
		require.NoError(t, err)

		service := corev1.Service{}
		err = r.client.Get(context.Background(), client.ObjectKey{Name: r.dynakube.Name + "-" + r.capability.ShortName(), Namespace: r.dynakube.Namespace}, &service)

		assert.Empty(t, service)
		require.Error(t, err)
	})
}

func TestCreateOrUpdateService(t *testing.T) {
	clt := createClient()
	dynakube := buildDynakube(capabilitiesWithService)
	mockStatefulSetReconciler := getMockReconciler(t, nil)
	mockCustompropertiesReconciler := getMockReconciler(t, nil)

	t.Run(`create service works`, func(t *testing.T) {
		r := NewReconciler(clt, capability.NewMultiCapability(dynakube), dynakube, mockStatefulSetReconciler, mockCustompropertiesReconciler).(*Reconciler)
		verifyReconciler(t, r)

		service := &corev1.Service{}
		err := r.client.Get(context.Background(), client.ObjectKey{Name: r.dynakube.Name + "-" + r.capability.ShortName(), Namespace: r.dynakube.Namespace}, service)
		require.Error(t, err)
		assert.NotNil(t, service)

		err = r.createOrUpdateService(context.Background())
		require.NoError(t, err)

		service = &corev1.Service{}
		err = r.client.Get(context.Background(), client.ObjectKey{Name: r.dynakube.Name + "-" + r.capability.ShortName(), Namespace: r.dynakube.Namespace}, service)
		require.NoError(t, err)
		assert.NotNil(t, service)
	})
	t.Run(`ports get updated`, func(t *testing.T) {
		r := NewReconciler(clt, capability.NewMultiCapability(dynakube), dynakube, mockStatefulSetReconciler, mockCustompropertiesReconciler).(*Reconciler)
		verifyReconciler(t, r)

		err := r.createOrUpdateService(context.Background())
		require.NoError(t, err)

		service := &corev1.Service{}
		err = r.client.Get(context.Background(), client.ObjectKey{Name: r.dynakube.Name + "-" + r.capability.ShortName(), Namespace: r.dynakube.Namespace}, service)
		require.NoError(t, err)
		assert.NotNil(t, service)

		service.Spec.Ports = []corev1.ServicePort{}

		err = r.createOrUpdateService(context.Background())
		require.NoError(t, err)

		actualService := &corev1.Service{}

		err = r.client.Get(context.Background(), client.ObjectKey{Name: r.dynakube.Name + "-" + r.capability.ShortName(), Namespace: r.dynakube.Namespace}, actualService)
		require.NoError(t, err)
		assert.NotNil(t, actualService)

		assert.Equal(t, int32(443), actualService.Spec.Ports[0].Port)

		require.NotEqual(t, actualService, service)
		require.NotEqual(t, actualService.Spec.Ports, service.Spec.Ports)
	})
	t.Run(`labels get updated`, func(t *testing.T) {
		r := NewReconciler(clt, capability.NewMultiCapability(dynakube), dynakube, mockStatefulSetReconciler, mockCustompropertiesReconciler).(*Reconciler)
		verifyReconciler(t, r)

		err := r.createOrUpdateService(context.Background())
		require.NoError(t, err)

		service := &corev1.Service{}
		err = r.client.Get(context.Background(), client.ObjectKey{Name: r.dynakube.Name + "-" + r.capability.ShortName(), Namespace: r.dynakube.Namespace}, service)
		require.NoError(t, err)
		assert.NotNil(t, service)

		service.Labels = map[string]string{}

		err = r.createOrUpdateService(context.Background())
		require.NoError(t, err)

		actualService := &corev1.Service{}

		err = r.client.Get(context.Background(), client.ObjectKey{Name: r.dynakube.Name + "-" + r.capability.ShortName(), Namespace: r.dynakube.Namespace}, actualService)
		require.NoError(t, err)
		assert.NotNil(t, service)

		require.NotEqual(t, actualService, service)
		require.NotEqual(t, actualService.Labels, service.Labels)
	})
}

func TestPortsAreOutdated(t *testing.T) {
	clt := createClient()
	dynakube := buildDynakube(capabilitiesWithService)
	mockStatefulSetReconciler := getMockReconciler(t, nil)
	mockCustompropertiesReconciler := getMockReconciler(t, nil)

	r := NewReconciler(clt, capability.NewMultiCapability(dynakube), dynakube, mockStatefulSetReconciler, mockCustompropertiesReconciler).(*Reconciler)
	verifyReconciler(t, r)

	desiredService := CreateService(r.dynakube, r.capability.ShortName())

	err := r.Reconcile(context.Background())
	require.NoError(t, err)

	t.Run(`ports are detected as outdated`, func(t *testing.T) {
		service := &corev1.Service{}
		err = r.client.Get(context.Background(), client.ObjectKey{Name: r.dynakube.Name + "-" + r.capability.ShortName(), Namespace: r.dynakube.Namespace}, service)
		require.NoError(t, err)
		assert.NotNil(t, service)

		assert.False(t, r.portsAreOutdated(service, desiredService))

		service.Spec.Ports = []corev1.ServicePort{}

		assert.True(t, r.portsAreOutdated(service, desiredService))
	})
}

func TestLabelsAreOutdated(t *testing.T) {
	clt := createClient()
	dynakube := buildDynakube(capabilitiesWithService)
	mockStatefulSetReconciler := getMockReconciler(t, nil)
	mockCustompropertiesReconciler := getMockReconciler(t, nil)
	r := NewReconciler(clt, capability.NewMultiCapability(dynakube), dynakube, mockStatefulSetReconciler, mockCustompropertiesReconciler).(*Reconciler)
	verifyReconciler(t, r)

	desiredService := CreateService(r.dynakube, r.capability.ShortName())

	err := r.Reconcile(context.Background())
	require.NoError(t, err)

	t.Run(`labels are detected as outdated`, func(t *testing.T) {
		service := &corev1.Service{}
		err = r.client.Get(context.Background(), client.ObjectKey{Name: r.dynakube.Name + "-" + r.capability.ShortName(), Namespace: r.dynakube.Namespace}, service)
		require.NoError(t, err)
		assert.NotNil(t, service)

		assert.False(t, r.labelsAreOutdated(service, desiredService))

		service.Labels = map[string]string{}

		assert.True(t, r.labelsAreOutdated(service, desiredService))
	})
	t.Run(`labelSelectors are detected as outdated`, func(t *testing.T) {
		service := &corev1.Service{}
		err = r.client.Get(context.Background(), client.ObjectKey{Name: r.dynakube.Name + "-" + r.capability.ShortName(), Namespace: r.dynakube.Namespace}, service)
		require.NoError(t, err)
		assert.NotNil(t, service)

		assert.False(t, r.labelsAreOutdated(service, desiredService))

		service.Spec.Selector = map[string]string{}

		assert.True(t, r.labelsAreOutdated(service, desiredService))
	})
}
