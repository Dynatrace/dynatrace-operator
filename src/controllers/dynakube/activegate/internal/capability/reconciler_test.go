package capability

import (
	"context"
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/activegate/capability"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/activegate/internal/authtoken"
	"github.com/Dynatrace/dynatrace-operator/src/kubesystem"
	"github.com/Dynatrace/dynatrace-operator/src/scheme"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
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

func createInstance() *dynatracev1beta1.DynaKube {
	instance := &dynatracev1beta1.DynaKube{
		Spec: dynatracev1beta1.DynaKubeSpec{
			ActiveGate: dynatracev1beta1.ActiveGateSpec{
				Capabilities: []dynatracev1beta1.CapabilityDisplayName{
					dynatracev1beta1.RoutingCapability.DisplayName,
					dynatracev1beta1.KubeMonCapability.DisplayName,
					dynatracev1beta1.MetricsIngestCapability.DisplayName,
					dynatracev1beta1.DynatraceApiCapability.DisplayName,
					dynatracev1beta1.SyntheticCapability.DisplayName,
				},
			},
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      testDynakube,
			Namespace: testNamespace,
		}}
	return instance
}

func createInstanceWithoutActiveGateService() *dynatracev1beta1.DynaKube {
	instance := &dynatracev1beta1.DynaKube{
		Spec: dynatracev1beta1.DynaKubeSpec{
			ActiveGate: dynatracev1beta1.ActiveGateSpec{
				Capabilities: []dynatracev1beta1.CapabilityDisplayName{
					dynatracev1beta1.KubeMonCapability.DisplayName,
					dynatracev1beta1.SyntheticCapability.DisplayName,
				},
			},
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      testDynakube,
			Namespace: testNamespace,
		}}
	return instance
}

func getMockReconciler() *MockReconciler {
	mockReconciler := MockReconciler{}
	mockReconciler.On("Reconcile").Return(nil)
	return &mockReconciler
}

func verifyReconciler(t *testing.T, r *Reconciler) {
	require.NotNil(t, r)
	require.NotNil(t, r.client)
	require.NotNil(t, r)
	require.NotNil(t, r.dynakube)
}

func TestReconcile(t *testing.T) {
	t.Run(`Reconciler works with multiple capabilities`, func(t *testing.T) {
		clt := createClient()
		instance := createInstance()
		mockStatefulSetReconciler := getMockReconciler()
		mockCustompropertiesReconciler := getMockReconciler()

		r := NewReconciler(clt, capability.NewMultiCapability(instance), instance, mockStatefulSetReconciler, mockCustompropertiesReconciler)
		verifyReconciler(t, r)

		err := r.Reconcile()
		mockStatefulSetReconciler.AssertCalled(t, "Reconcile")
		mockCustompropertiesReconciler.AssertCalled(t, "Reconcile")
		require.NoError(t, err)
	})
	t.Run(`statefulSetReconciler errors`, func(t *testing.T) {
		clt := createClient()
		instance := createInstanceWithoutActiveGateService()
		mockStatefulSetReconciler := &MockReconciler{}
		mockStatefulSetReconciler.On("Reconcile").Return(errors.New(""))
		mockCustompropertiesReconciler := getMockReconciler()

		r := NewReconciler(clt, capability.NewMultiCapability(instance), instance, mockStatefulSetReconciler, mockCustompropertiesReconciler)
		verifyReconciler(t, r)

		err := r.Reconcile()
		mockStatefulSetReconciler.AssertCalled(t, "Reconcile")
		mockCustompropertiesReconciler.AssertCalled(t, "Reconcile")
		require.Error(t, err)
	})
	t.Run(`customPropertiesReconciler errors`, func(t *testing.T) {
		clt := createClient()
		instance := createInstanceWithoutActiveGateService()
		mockStatefulSetReconciler := getMockReconciler()
		mockCustompropertiesReconciler := &MockReconciler{}
		mockCustompropertiesReconciler.On("Reconcile").Return(errors.New(""))

		r := NewReconciler(clt, capability.NewMultiCapability(instance), instance, mockStatefulSetReconciler, mockCustompropertiesReconciler)
		verifyReconciler(t, r)

		err := r.Reconcile()
		mockCustompropertiesReconciler.AssertCalled(t, "Reconcile")
		require.Error(t, err)
	})
	t.Run(`statefulSetReconciler and customPropertiesReconciler error`, func(t *testing.T) {
		clt := createClient()
		instance := createInstanceWithoutActiveGateService()
		mockStatefulSetReconciler := &MockReconciler{}
		mockStatefulSetReconciler.On("Reconcile").Return(errors.New(""))
		mockCustompropertiesReconciler := &MockReconciler{}
		mockCustompropertiesReconciler.On("Reconcile").Return(errors.New(""))

		r := NewReconciler(clt, capability.NewMultiCapability(instance), instance, mockStatefulSetReconciler, mockCustompropertiesReconciler)
		verifyReconciler(t, r)

		err := r.Reconcile()
		require.Error(t, err)
	})
	t.Run(`Service gets created`, func(t *testing.T) {
		clt := createClient()
		instance := createInstance()
		mockStatefulSetReconciler := getMockReconciler()
		mockCustompropertiesReconciler := getMockReconciler()

		r := NewReconciler(clt, capability.NewMultiCapability(instance), instance, mockStatefulSetReconciler, mockCustompropertiesReconciler)
		verifyReconciler(t, r)

		err := r.Reconcile()
		mockStatefulSetReconciler.AssertCalled(t, "Reconcile")
		mockCustompropertiesReconciler.AssertCalled(t, "Reconcile")
		require.NoError(t, err)

		service := corev1.Service{}
		err = r.client.Get(context.TODO(), client.ObjectKey{Name: r.dynakube.Name + "-" + r.capability.ShortName(), Namespace: r.dynakube.Namespace}, &service)

		assert.NotNil(t, service)
		assert.NoError(t, err)
	})
	t.Run(`Service doesnt get created when missing capabilities`, func(t *testing.T) {
		clt := createClient()
		instance := createInstanceWithoutActiveGateService()
		mockStatefulSetReconciler := getMockReconciler()
		mockCustompropertiesReconciler := getMockReconciler()

		r := NewReconciler(clt, capability.NewMultiCapability(instance), instance, mockStatefulSetReconciler, mockCustompropertiesReconciler)
		verifyReconciler(t, r)

		err := r.Reconcile()
		mockStatefulSetReconciler.AssertCalled(t, "Reconcile")
		mockCustompropertiesReconciler.AssertCalled(t, "Reconcile")
		require.NoError(t, err)

		service := corev1.Service{}
		err = r.client.Get(context.TODO(), client.ObjectKey{Name: r.dynakube.Name + "-" + r.capability.ShortName(), Namespace: r.dynakube.Namespace}, &service)

		assert.Empty(t, service)
		assert.Error(t, err)
	})
}

func TestCreateOrUpdateService(t *testing.T) {
	t.Run(`createOrUpdateService works`, func(t *testing.T) {
		clt := createClient()
		instance := createInstance()
		mockStatefulSetReconciler := getMockReconciler()
		mockCustompropertiesReconciler := getMockReconciler()

		r := NewReconciler(clt, capability.NewMultiCapability(instance), instance, mockStatefulSetReconciler, mockCustompropertiesReconciler)
		verifyReconciler(t, r)

		service := &corev1.Service{}
		err := r.client.Get(context.TODO(), client.ObjectKey{Name: r.dynakube.Name + "-" + r.capability.ShortName(), Namespace: r.dynakube.Namespace}, service)
		require.Error(t, err)
		assert.NotNil(t, service)

		err = r.createOrUpdateService()
		require.NoError(t, err)

		service = &corev1.Service{}
		err = r.client.Get(context.TODO(), client.ObjectKey{Name: r.dynakube.Name + "-" + r.capability.ShortName(), Namespace: r.dynakube.Namespace}, service)
		require.NoError(t, err)
		assert.NotNil(t, service)
	})
	t.Run(`Update works for ports`, func(t *testing.T) {
		clt := createClient()
		instance := createInstance()
		mockStatefulSetReconciler := getMockReconciler()
		mockCustompropertiesReconciler := getMockReconciler()

		r := NewReconciler(clt, capability.NewMultiCapability(instance), instance, mockStatefulSetReconciler, mockCustompropertiesReconciler)
		verifyReconciler(t, r)

		err := r.createOrUpdateService()
		require.NoError(t, err)

		service := &corev1.Service{}
		err = r.client.Get(context.TODO(), client.ObjectKey{Name: r.dynakube.Name + "-" + r.capability.ShortName(), Namespace: r.dynakube.Namespace}, service)
		require.NoError(t, err)
		assert.NotNil(t, service)

		tmpService := &corev1.Service{}

		service.Spec.Ports = []corev1.ServicePort{}

		err = r.createOrUpdateService()
		require.NoError(t, err)

		err = r.client.Get(context.TODO(), client.ObjectKey{Name: r.dynakube.Name + "-" + r.capability.ShortName(), Namespace: r.dynakube.Namespace}, tmpService)
		require.NoError(t, err)
		assert.NotNil(t, tmpService)

		require.NotEqual(t, tmpService, service)
	})
	t.Run(`update works for labels`, func(t *testing.T) {
		clt := createClient()
		instance := createInstance()
		mockStatefulSetReconciler := getMockReconciler()
		mockCustompropertiesReconciler := getMockReconciler()

		r := NewReconciler(clt, capability.NewMultiCapability(instance), instance, mockStatefulSetReconciler, mockCustompropertiesReconciler)
		verifyReconciler(t, r)

		err := r.createOrUpdateService()
		require.NoError(t, err)

		service := &corev1.Service{}
		err = r.client.Get(context.TODO(), client.ObjectKey{Name: r.dynakube.Name + "-" + r.capability.ShortName(), Namespace: r.dynakube.Namespace}, service)
		require.NoError(t, err)
		assert.NotNil(t, service)

		tmpService := &corev1.Service{}

		service.Labels = map[string]string{}

		err = r.createOrUpdateService()
		require.NoError(t, err)

		err = r.client.Get(context.TODO(), client.ObjectKey{Name: r.dynakube.Name + "-" + r.capability.ShortName(), Namespace: r.dynakube.Namespace}, tmpService)
		require.NoError(t, err)
		assert.NotNil(t, service)

		require.NotEqual(t, tmpService, service)
	})
}

func TestPortsAreOutdated(t *testing.T) {
	t.Run(`portsAreOutdated works`, func(t *testing.T) {
		clt := createClient()
		instance := createInstance()
		mockStatefulSetReconciler := getMockReconciler()
		mockCustompropertiesReconciler := getMockReconciler()

		r := NewReconciler(clt, capability.NewMultiCapability(instance), instance, mockStatefulSetReconciler, mockCustompropertiesReconciler)
		verifyReconciler(t, r)

		desiredService := CreateService(r.dynakube, r.capability.ShortName())

		err := r.Reconcile()
		require.NoError(t, err)

		service := &corev1.Service{}
		err = r.client.Get(context.TODO(), client.ObjectKey{Name: r.dynakube.Name + "-" + r.capability.ShortName(), Namespace: r.dynakube.Namespace}, service)
		require.NoError(t, err)
		assert.NotNil(t, service)

		assert.False(t, r.portsAreOutdated(service, desiredService))

		service.Spec.Ports = []corev1.ServicePort{}

		assert.True(t, r.portsAreOutdated(service, desiredService))
	})
}

func TestLabelsAreOutdated(t *testing.T) {
	t.Run(`labelsAreOutdated works for labels`, func(t *testing.T) {
		clt := createClient()
		instance := createInstance()
		mockStatefulSetReconciler := getMockReconciler()
		mockCustompropertiesReconciler := getMockReconciler()

		r := NewReconciler(clt, capability.NewMultiCapability(instance), instance, mockStatefulSetReconciler, mockCustompropertiesReconciler)
		verifyReconciler(t, r)

		desiredService := CreateService(r.dynakube, r.capability.ShortName())

		err := r.Reconcile()
		require.NoError(t, err)

		service := &corev1.Service{}
		err = r.client.Get(context.TODO(), client.ObjectKey{Name: r.dynakube.Name + "-" + r.capability.ShortName(), Namespace: r.dynakube.Namespace}, service)
		require.NoError(t, err)
		assert.NotNil(t, service)

		assert.False(t, r.labelsAreOutdated(service, desiredService))

		service.Labels = map[string]string{}

		assert.True(t, r.labelsAreOutdated(service, desiredService))
	})
	t.Run(`labelsAreOutdated works for selectors`, func(t *testing.T) {
		clt := createClient()
		instance := createInstance()
		mockStatefulSetReconciler := getMockReconciler()
		mockCustompropertiesReconciler := getMockReconciler()

		r := NewReconciler(clt, capability.NewMultiCapability(instance), instance, mockStatefulSetReconciler, mockCustompropertiesReconciler)
		verifyReconciler(t, r)

		desiredService := CreateService(r.dynakube, r.capability.ShortName())

		err := r.Reconcile()
		require.NoError(t, err)

		service := &corev1.Service{}
		err = r.client.Get(context.TODO(), client.ObjectKey{Name: r.dynakube.Name + "-" + r.capability.ShortName(), Namespace: r.dynakube.Namespace}, service)
		require.NoError(t, err)
		assert.NotNil(t, service)

		assert.False(t, r.labelsAreOutdated(service, desiredService))

		service.Spec.Selector = map[string]string{}

		assert.True(t, r.labelsAreOutdated(service, desiredService))
	})
}
