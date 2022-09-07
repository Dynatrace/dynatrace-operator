package statefulset

import (
	"context"
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/activegate/capability"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/activegate/internal/authtoken"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/activegate/internal/customproperties"
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
		WithObjects(&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      dynatracev1beta1.AuthTokenSecretSuffix,
				Namespace: testNamespace,
			},
			Data: map[string][]byte{authtoken.ActiveGateAuthTokenName: []byte(testToken)},
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
	r.Dynakube.Annotations = map[string]string{}
	require.NotNil(t, r)
	require.NotNil(t, r.Client)
	require.NotNil(t, r.scheme)
	require.NotNil(t, r.Dynakube)

	return r
}

func TestReconcile(t *testing.T) {
	t.Run(`create stateful set`, func(t *testing.T) {
		r := createDefaultReconciler(t)
		update, err := r.Reconcile()

		assert.True(t, update)
		assert.NoError(t, err)

		statefulSet := &appsv1.StatefulSet{}
		err = r.Get(context.TODO(), client.ObjectKey{Name: r.Dynakube.Name + "-" + r.capability.ShortName(), Namespace: r.Dynakube.Namespace}, statefulSet)

		assert.NotNil(t, statefulSet)
		assert.NoError(t, err)
	})
	t.Run(`update stateful set`, func(t *testing.T) {
		r := createDefaultReconciler(t)
		update, err := r.Reconcile()

		assert.True(t, update)
		assert.NoError(t, err)

		statefulSet := &appsv1.StatefulSet{}
		err = r.Get(context.TODO(), client.ObjectKey{Name: r.Dynakube.Name + "-" + r.capability.ShortName(), Namespace: r.Dynakube.Namespace}, statefulSet)

		assert.NotNil(t, statefulSet)
		assert.NoError(t, err)

		r.Dynakube.Spec.Proxy = &dynatracev1beta1.DynaKubeProxy{Value: testValue}
		update, err = r.Reconcile()

		assert.True(t, update)
		assert.NoError(t, err)

		newStatefulSet := &appsv1.StatefulSet{}
		err = r.Get(context.TODO(), client.ObjectKey{Name: r.Dynakube.Name + "-" + r.capability.ShortName(), Namespace: r.Dynakube.Namespace}, newStatefulSet)

		assert.NotNil(t, statefulSet)
		assert.NoError(t, err)

		found := 0
		for _, vm := range newStatefulSet.Spec.Template.Spec.Containers[0].VolumeMounts {
			if vm.Name == InternalProxySecretVolumeName {
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
	err = controllerutil.SetControllerReference(r.Dynakube, desiredSts, r.scheme)
	require.NoError(t, err)

	sts, err := r.getStatefulSet(desiredSts)
	assert.NoError(t, err)
	assert.Equal(t, *desiredSts, *sts)
}

func TestReconcile_CreateStatefulSetIfNotExists(t *testing.T) {
	r := createDefaultReconciler(t)
	desiredSts, err := r.buildDesiredStatefulSet()
	require.NoError(t, err)
	require.NotNil(t, desiredSts)

	created, err := r.createStatefulSetIfNotExists(desiredSts)
	assert.NoError(t, err)
	assert.True(t, created)

	created, err = r.createStatefulSetIfNotExists(desiredSts)
	assert.NoError(t, err)
	assert.False(t, created)
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

	r.Dynakube.Spec.Proxy = &dynatracev1beta1.DynaKubeProxy{Value: testValue}
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

	r.Dynakube.Spec.Proxy = &dynatracev1beta1.DynaKubeProxy{Value: testValue}
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
	assert.NotEmpty(t, hash)

	r.Dynakube.Spec.Routing.CustomProperties = &dynatracev1beta1.DynaKubeValueSource{Value: testValue}
	hash, err = r.calculateActiveGateConfigurationHash()
	assert.NoError(t, err)
	assert.NotEmpty(t, hash)

	r.Dynakube.Spec.Routing.CustomProperties = &dynatracev1beta1.DynaKubeValueSource{ValueFrom: testName}
	hash, err = r.calculateActiveGateConfigurationHash()
	r.Dynakube.Annotations[dynatracev1beta1.AnnotationFeatureActiveGateAuthToken] = "false"
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
	assert.NotEmpty(t, hash)

	hash, err = r.calculateActiveGateConfigurationHash()
	assert.NoError(t, err)
	assert.NotEmpty(t, hash)

	err = r.Create(context.TODO(), &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      r.Dynakube.ActiveGateAuthTokenSecret(),
			Namespace: r.Dynakube.Namespace,
		},
		Data: map[string][]byte{
			authtoken.ActiveGateAuthTokenName: []byte(testValue),
		},
	})
	require.Error(t, err)

	r = createDefaultReconciler(t)
	r.Dynakube.Annotations[dynatracev1beta1.AnnotationFeatureActiveGateAuthToken] = "false"
	hash, err = r.calculateActiveGateConfigurationHash()
	assert.NoError(t, err)
	assert.Empty(t, hash)
}
