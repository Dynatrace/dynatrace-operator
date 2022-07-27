package activegate

import (
	"context"
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/activegate/capability"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/activegate/secrets"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/activegate/statefulset"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/customproperties"
	"github.com/Dynatrace/dynatrace-operator/src/kubesystem"
	"github.com/Dynatrace/dynatrace-operator/src/scheme"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	testUID       = "test-uid"
	testName      = "test-name"
	testNamespace = "test-namespace"
	testValue     = "test-value"
)

func TestNewReconiler(t *testing.T) {
	createDefaultReconciler(t)
}

func createDefaultReconciler(t *testing.T) *Reconciler {
	clt := fake.NewClientBuilder().
		WithScheme(scheme.Scheme).
		WithObjects(&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: kubesystem.Namespace,
				UID:  testUID,
			},
		}).
		Build()
	instance := &dynatracev1beta1.DynaKube{
		Spec: dynatracev1beta1.DynaKubeSpec{
			ActiveGate: dynatracev1beta1.ActiveGateSpec{
				Capabilities: []dynatracev1beta1.CapabilityDisplayName{
					dynatracev1beta1.RoutingCapability.DisplayName,
				}},
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNamespace,
		}}

	capability.NewRoutingCapability(instance)

	r := NewReconciler(clt, clt, scheme.Scheme, instance, capability.NewRoutingCapability(instance))
	require.NotNil(t, r)
	require.NotNil(t, r.Client)
	require.NotNil(t, r.scheme)
	require.NotNil(t, r.Instance)

	return r
}

func TestReconcile(t *testing.T) {
	t.Run(`reconcile custom properties`, func(t *testing.T) {
		r := createDefaultReconciler(t)
		r.Instance.Spec.Routing.CustomProperties = &dynatracev1beta1.DynaKubeValueSource{
			Value: testValue,
		}
		_, err := r.Reconcile()

		assert.NoError(t, err)

		var customProperties corev1.Secret
		err = r.Get(context.TODO(), client.ObjectKey{Name: r.Instance.Name + "-" + r.feature + "-" + customproperties.Suffix, Namespace: r.Instance.Namespace}, &customProperties)
		assert.NoError(t, err)
		assert.NotNil(t, customProperties)
		assert.Contains(t, customProperties.Data, customproperties.DataKey)
		assert.Equal(t, testValue, string(customProperties.Data[customproperties.DataKey]))
	})
	t.Run(`create stateful set`, func(t *testing.T) {
		r := createDefaultReconciler(t)
		update, err := r.Reconcile()

		assert.True(t, update)
		assert.NoError(t, err)

		statefulSet := &appsv1.StatefulSet{}
		err = r.Get(context.TODO(), client.ObjectKey{Name: r.Instance.Name + "-" + r.feature, Namespace: r.Instance.Namespace}, statefulSet)

		assert.NotNil(t, statefulSet)
		assert.NoError(t, err)
	})
	t.Run(`update stateful set`, func(t *testing.T) {
		r := createDefaultReconciler(t)
		update, err := r.Reconcile()

		assert.True(t, update)
		assert.NoError(t, err)

		statefulSet := &appsv1.StatefulSet{}
		err = r.Get(context.TODO(), client.ObjectKey{Name: r.Instance.Name + "-" + r.feature, Namespace: r.Instance.Namespace}, statefulSet)

		assert.NotNil(t, statefulSet)
		assert.NoError(t, err)

		r.Instance.Spec.Proxy = &dynatracev1beta1.DynaKubeProxy{Value: testValue}
		update, err = r.Reconcile()

		assert.True(t, update)
		assert.NoError(t, err)

		newStatefulSet := &appsv1.StatefulSet{}
		err = r.Get(context.TODO(), client.ObjectKey{Name: r.Instance.Name + "-" + r.feature, Namespace: r.Instance.Namespace}, newStatefulSet)

		assert.NotNil(t, statefulSet)
		assert.NoError(t, err)

		found := 0
		for _, vm := range newStatefulSet.Spec.Template.Spec.Containers[0].VolumeMounts {
			if vm.Name == statefulset.InternalProxySecretVolumeName {
				found = found + 1
			}
		}
		assert.Equal(t, 4, found)
	})
}

func TestReconcile_GetStatefulSet(t *testing.T) {
	r := createDefaultReconciler(t)
	update, err := r.Reconcile()
	assert.True(t, update)
	assert.NoError(t, err)

	desiredSts, err := r.buildDesiredStatefulSet()
	assert.NoError(t, err)
	assert.NotNil(t, desiredSts)

	desiredSts.Kind = "StatefulSet"
	desiredSts.APIVersion = "apps/v1"
	desiredSts.ResourceVersion = "1"
	err = controllerutil.SetControllerReference(r.Instance, desiredSts, r.scheme)
	require.NoError(t, err)

	sts, err := r.getStatefulSet(desiredSts)
	assert.NoError(t, err)
	assert.Equal(t, *desiredSts, *sts)
}

func TestReconcile_CreateStatefulSetIfNotExists(t *testing.T) {
	t.Run("stateful set is created if it does not exists", func(t *testing.T) {
		r := createDefaultReconciler(t)
		desiredSts, err := r.buildDesiredStatefulSet()
		require.NoError(t, err)
		require.NotNil(t, desiredSts)

		created, err := r.createStatefulSetIfNotExists(desiredSts)
		assert.NoError(t, err)
		assert.True(t, created)
	})
	t.Run("stateful set is not created if it exists", func(t *testing.T) {
		r := createDefaultReconciler(t)
		desiredSts, err := r.buildDesiredStatefulSet()
		require.NoError(t, err)
		require.NotNil(t, desiredSts)

		created, err := r.createStatefulSetIfNotExists(desiredSts)
		require.NoError(t, err)
		require.True(t, created)

		created, err = r.createStatefulSetIfNotExists(desiredSts)
		assert.NoError(t, err)
		assert.False(t, created)
	})
}

type failingClient struct {
	created int
	client.Client
}

func (clt *failingClient) Create(ctx context.Context, obj client.Object, opts ...client.CreateOption) error {
	if clt.created == 0 {
		clt.created++
		return errors.New("failing")
	}

	return clt.Client.Create(ctx, obj, opts...)
}

func TestCreateStatefulSet(t *testing.T) {
	t.Run("creates stateful set", func(t *testing.T) {
		fakeClient := fake.NewClientBuilder().WithScheme(scheme.Scheme).Build()
		r := &Reconciler{
			Client: fakeClient,
		}
		desiredStatefulSet := appsv1.StatefulSet{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "apps/v1",
				Kind:       "StatefulSet",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      testName,
				Namespace: testNamespace,
			}}

		err := r.createStatefulSet(&desiredStatefulSet)

		assert.NoError(t, err)

		var createdStatefulSet appsv1.StatefulSet
		err = r.Get(context.TODO(), client.ObjectKey{Namespace: testNamespace, Name: testName}, &createdStatefulSet)

		assert.NoError(t, err)
		assert.Equal(t, desiredStatefulSet, createdStatefulSet)
	})
	t.Run("changes affinities if it does not succeed at first", func(t *testing.T) {
		fakeClient := &failingClient{
			Client: fake.NewClientBuilder().
				WithScheme(scheme.Scheme).
				WithObjects(&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: kubesystem.Namespace,
						UID:  testUID,
					},
				}).Build(),
		}
		r := &Reconciler{
			Client:     fakeClient,
			apiReader:  fakeClient,
			capability: &dynatracev1beta1.CapabilityProperties{},
			Instance:   &dynatracev1beta1.DynaKube{},
		}
		desiredStatefulSet := appsv1.StatefulSet{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "apps/v1",
				Kind:       "StatefulSet",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      testName,
				Namespace: testNamespace,
			},
			Spec: appsv1.StatefulSetSpec{
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Affinity: affinity(),
					},
				},
			}}
		expectedStatefulSet := appsv1.StatefulSet{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "apps/v1",
				Kind:       "StatefulSet",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:            testName,
				Namespace:       testNamespace,
				ResourceVersion: "1",
			},
			Spec: appsv1.StatefulSetSpec{
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Affinity: affinityWithoutArch(),
					},
				},
			}}

		err := r.createStatefulSet(&desiredStatefulSet)
		assert.Error(t, err)
		// After the first error, a listener is appended to change the affinity

		newStatefulSet, err := r.buildDesiredStatefulSet()
		assert.NoError(t, err)
		assert.Equal(t, expectedStatefulSet, newStatefulSet)
	})
}

func TestReconcile_UpdateStatefulSetIfOutdated(t *testing.T) {
	r := createDefaultReconciler(t)
	desiredSts, err := r.buildDesiredStatefulSet()
	require.NoError(t, err)
	require.NotNil(t, desiredSts)

	updated, err := r.updateStatefulSetIfOutdated(desiredSts)
	assert.Error(t, err)
	assert.False(t, updated)
	assert.True(t, k8serrors.IsNotFound(errors.Cause(err)))

	created, err := r.createStatefulSetIfNotExists(desiredSts)
	require.True(t, created)
	require.NoError(t, err)

	updated, err = r.updateStatefulSetIfOutdated(desiredSts)
	assert.NoError(t, err)
	assert.False(t, updated)

	r.Instance.Spec.Proxy = &dynatracev1beta1.DynaKubeProxy{Value: testValue}
	desiredSts, err = r.buildDesiredStatefulSet()
	require.NoError(t, err)

	updated, err = r.updateStatefulSetIfOutdated(desiredSts)
	assert.NoError(t, err)
	assert.True(t, updated)
}

func TestReconcile_DeleteStatefulSetIfOldLabelsAreUsed(t *testing.T) {
	r := createDefaultReconciler(t)
	desiredSts, err := r.buildDesiredStatefulSet()
	require.NoError(t, err)
	require.NotNil(t, desiredSts)

	deleted, err := r.deleteStatefulSetIfOldLabelsAreUsed(desiredSts)
	assert.Error(t, err)
	assert.False(t, deleted)
	assert.True(t, k8serrors.IsNotFound(errors.Cause(err)))

	created, err := r.createStatefulSetIfNotExists(desiredSts)
	require.True(t, created)
	require.NoError(t, err)

	deleted, err = r.deleteStatefulSetIfOldLabelsAreUsed(desiredSts)
	assert.NoError(t, err)
	assert.False(t, deleted)

	r.Instance.Spec.Proxy = &dynatracev1beta1.DynaKubeProxy{Value: testValue}
	desiredSts, err = r.buildDesiredStatefulSet()
	require.NoError(t, err)
	correctLabels := desiredSts.Labels
	desiredSts.Labels = map[string]string{"activegate": "dynakube"}
	err = r.Update(context.TODO(), desiredSts)
	assert.NoError(t, err)

	desiredSts.Labels = correctLabels
	deleted, err = r.deleteStatefulSetIfOldLabelsAreUsed(desiredSts)
	assert.NoError(t, err)
	assert.True(t, deleted)
}

func TestReconcile_GetCustomPropertyHash(t *testing.T) {
	r := createDefaultReconciler(t)
	hash, err := r.calculateActiveGateConfigurationHash()
	assert.NoError(t, err)
	assert.Empty(t, hash)

	r.Instance.Spec.Routing.CustomProperties = &dynatracev1beta1.DynaKubeValueSource{Value: testValue}
	hash, err = r.calculateActiveGateConfigurationHash()
	assert.NoError(t, err)
	assert.NotEmpty(t, hash)

	r.Instance.Spec.Routing.CustomProperties = &dynatracev1beta1.DynaKubeValueSource{ValueFrom: testName}
	hash, err = r.calculateActiveGateConfigurationHash()
	assert.Error(t, err)
	assert.Empty(t, hash)

	err = r.Create(context.TODO(), &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testName,
			Namespace: testNamespace,
		},
		Data: map[string][]byte{
			customproperties.DataKey: []byte(testValue),
		},
	})
	require.NoError(t, err)

	hash, err = r.calculateActiveGateConfigurationHash()
	assert.NoError(t, err)
	assert.NotEmpty(t, hash)
}

func TestReconcile_GetActiveGateAuthTokenHash(t *testing.T) {
	r := createDefaultReconciler(t)
	hash, err := r.calculateActiveGateConfigurationHash()
	assert.NoError(t, err)
	assert.Empty(t, hash)

	r.Instance.Annotations = make(map[string]string)
	r.Instance.Annotations[dynatracev1beta1.AnnotationFeatureEnableActiveGateAuthToken] = "true"

	hash, err = r.calculateActiveGateConfigurationHash()
	assert.Error(t, err)
	assert.Empty(t, hash)

	err = r.Create(context.TODO(), &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      r.Instance.ActiveGateAuthTokenSecret(),
			Namespace: r.Instance.Namespace,
		},
		Data: map[string][]byte{
			secrets.ActiveGateAuthTokenName: []byte(testValue),
		},
	})
	require.NoError(t, err)

	hash, err = r.calculateActiveGateConfigurationHash()
	assert.NoError(t, err)
	assert.NotEmpty(t, hash)
}

func TestContains(t *testing.T) {
	t.Run("contains finds primitive types", func(t *testing.T) {
		array := []string{testKey, testName}

		assert.True(t, contains(array, testKey))
		assert.True(t, contains(array, testName))
		assert.False(t, contains(array, testValue))
	})
	t.Run("contains finds complex types", func(t *testing.T) {
		type testStruct struct {
			property        string
			complexProperty struct{ property string }
		}
		complexArray := []testStruct{
			{property: testKey, complexProperty: struct{ property string }{property: testKey}},
			{property: testName, complexProperty: struct{ property string }{property: testValue}},
		}

		assert.True(t, contains(complexArray, testStruct{property: testKey, complexProperty: struct{ property string }{property: testKey}}))
		assert.True(t, contains(complexArray, testStruct{property: testName, complexProperty: struct{ property string }{property: testValue}}))
		assert.False(t, contains(complexArray, testStruct{property: testValue, complexProperty: struct{ property string }{property: testValue}}))
		assert.False(t, contains(complexArray, testStruct{}))
	})
}
