package statefulset

import (
	"context"
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/activegate/capability"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/activegate/customproperties"
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

func createDefaultReconciler(t *testing.T) *ActiveGateStatefulSetReconciler {
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
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNamespace,
		}}

	capability.NewRoutingCapability(instance)

	reconciler := NewAGStatefulSetReconciler(clt, clt, scheme.Scheme, instance, capability.NewRoutingCapability(instance))
	require.NotNil(t, reconciler)
	require.NotNil(t, reconciler.Client)
	require.NotNil(t, reconciler.scheme)
	require.NotNil(t, reconciler.Instance)

	return reconciler
}

func TestReconcile(t *testing.T) {
	t.Run(`reconcile custom properties`, func(t *testing.T) {
		reconciler := createDefaultReconciler(t)
		reconciler.Instance.Spec.Routing.CustomProperties = &dynatracev1beta1.DynaKubeValueSource{
			Value: testValue,
		}
		_, err := reconciler.Reconcile()

		assert.NoError(t, err)

		var customProperties corev1.Secret
		err = reconciler.Get(context.TODO(), client.ObjectKey{Name: reconciler.Instance.Name + "-" + reconciler.feature + "-" + customproperties.Suffix, Namespace: reconciler.Instance.Namespace}, &customProperties)
		assert.NoError(t, err)
		assert.NotNil(t, customProperties)
		assert.Contains(t, customProperties.Data, customproperties.DataKey)
		assert.Equal(t, testValue, string(customProperties.Data[customproperties.DataKey]))
	})
	t.Run(`create stateful set`, func(t *testing.T) {
		reconciler := createDefaultReconciler(t)
		update, err := reconciler.Reconcile()

		assert.True(t, update)
		assert.NoError(t, err)

		statefulSet := &appsv1.StatefulSet{}
		err = reconciler.Get(context.TODO(), client.ObjectKey{Name: reconciler.Instance.Name + "-" + reconciler.feature, Namespace: reconciler.Instance.Namespace}, statefulSet)

		assert.NotNil(t, statefulSet)
		assert.NoError(t, err)
	})
	t.Run(`update stateful set`, func(t *testing.T) {
		reconciler := createDefaultReconciler(t)
		update, err := reconciler.Reconcile()

		assert.True(t, update)
		assert.NoError(t, err)

		statefulSet := &appsv1.StatefulSet{}
		err = reconciler.Get(context.TODO(), client.ObjectKey{Name: reconciler.Instance.Name + "-" + reconciler.feature, Namespace: reconciler.Instance.Namespace}, statefulSet)

		assert.NotNil(t, statefulSet)
		assert.NoError(t, err)

		reconciler.Instance.Spec.Proxy = &dynatracev1beta1.DynaKubeProxy{Value: testValue}
		update, err = reconciler.Reconcile()

		assert.True(t, update)
		assert.NoError(t, err)

		newStatefulSet := &appsv1.StatefulSet{}
		err = reconciler.Get(context.TODO(), client.ObjectKey{Name: reconciler.Instance.Name + "-" + reconciler.feature, Namespace: reconciler.Instance.Namespace}, newStatefulSet)

		assert.NotNil(t, statefulSet)
		assert.NoError(t, err)

		found := false
		for _, env := range newStatefulSet.Spec.Template.Spec.Containers[0].Env {
			if env.Name == DTInternalProxy {
				found = true
				assert.Equal(t, testValue, env.Value)
			}
		}
		assert.True(t, found)
	})
}

func TestReconcile_GetStatefulSet(t *testing.T) {
	reconciler := createDefaultReconciler(t)
	update, err := reconciler.Reconcile()
	assert.True(t, update)
	assert.NoError(t, err)

	desiredSts, err := reconciler.buildDesiredStatefulSet()
	assert.NoError(t, err)
	assert.NotNil(t, desiredSts)

	desiredSts.Kind = "StatefulSet"
	desiredSts.APIVersion = "apps/v1"
	desiredSts.ResourceVersion = "1"
	err = controllerutil.SetControllerReference(reconciler.Instance, desiredSts, reconciler.scheme)
	require.NoError(t, err)

	sts, err := reconciler.getStatefulSet(desiredSts)
	assert.NoError(t, err)
	assert.Equal(t, *desiredSts, *sts)
}

func TestReconcile_CreateStatefulSetIfNotExists(t *testing.T) {
	reconciler := createDefaultReconciler(t)
	desiredSts, err := reconciler.buildDesiredStatefulSet()
	require.NoError(t, err)
	require.NotNil(t, desiredSts)

	created, err := reconciler.createStatefulSetIfNotExists(desiredSts)
	assert.NoError(t, err)
	assert.True(t, created)

	created, err = reconciler.createStatefulSetIfNotExists(desiredSts)
	assert.NoError(t, err)
	assert.False(t, created)
}

func TestReconcile_UpdateStatefulSetIfOutdated(t *testing.T) {
	reconciler := createDefaultReconciler(t)
	desiredSts, err := reconciler.buildDesiredStatefulSet()
	require.NoError(t, err)
	require.NotNil(t, desiredSts)

	updated, err := reconciler.updateStatefulSetIfOutdated(desiredSts)
	assert.Error(t, err)
	assert.False(t, updated)
	assert.True(t, k8serrors.IsNotFound(errors.Cause(err)))

	created, err := reconciler.createStatefulSetIfNotExists(desiredSts)
	require.True(t, created)
	require.NoError(t, err)

	updated, err = reconciler.updateStatefulSetIfOutdated(desiredSts)
	assert.NoError(t, err)
	assert.False(t, updated)

	reconciler.Instance.Spec.Proxy = &dynatracev1beta1.DynaKubeProxy{Value: testValue}
	desiredSts, err = reconciler.buildDesiredStatefulSet()
	require.NoError(t, err)

	updated, err = reconciler.updateStatefulSetIfOutdated(desiredSts)
	assert.NoError(t, err)
	assert.True(t, updated)
}

func TestReconcile_DeleteStatefulSetIfOldLabelsAreUsed(t *testing.T) {
	reconciler := createDefaultReconciler(t)
	desiredSts, err := reconciler.buildDesiredStatefulSet()
	require.NoError(t, err)
	require.NotNil(t, desiredSts)

	deleted, err := reconciler.deleteStatefulSetIfOldLabelsAreUsed(desiredSts)
	assert.Error(t, err)
	assert.False(t, deleted)
	assert.True(t, k8serrors.IsNotFound(errors.Cause(err)))

	created, err := reconciler.createStatefulSetIfNotExists(desiredSts)
	require.True(t, created)
	require.NoError(t, err)

	deleted, err = reconciler.deleteStatefulSetIfOldLabelsAreUsed(desiredSts)
	assert.NoError(t, err)
	assert.False(t, deleted)

	reconciler.Instance.Spec.Proxy = &dynatracev1beta1.DynaKubeProxy{Value: testValue}
	desiredSts, err = reconciler.buildDesiredStatefulSet()
	require.NoError(t, err)
	correctLabels := desiredSts.Labels
	desiredSts.Labels = map[string]string{"activegate": "dynakube"}
	err = reconciler.Update(context.TODO(), desiredSts)
	assert.NoError(t, err)

	desiredSts.Labels = correctLabels
	deleted, err = reconciler.deleteStatefulSetIfOldLabelsAreUsed(desiredSts)
	assert.NoError(t, err)
	assert.True(t, deleted)
}

func TestReconcile_GetCustomPropertyHash(t *testing.T) {
	reconciler := createDefaultReconciler(t)
	hash, err := reconciler.calculateCustomPropertyHash()
	assert.NoError(t, err)
	assert.Empty(t, hash)

	reconciler.Instance.Spec.Routing.CustomProperties = &dynatracev1beta1.DynaKubeValueSource{Value: testValue}
	hash, err = reconciler.calculateCustomPropertyHash()
	assert.NoError(t, err)
	assert.NotEmpty(t, hash)

	reconciler.Instance.Spec.Routing.CustomProperties = &dynatracev1beta1.DynaKubeValueSource{ValueFrom: testName}
	hash, err = reconciler.calculateCustomPropertyHash()
	assert.Error(t, err)
	assert.Empty(t, hash)

	err = reconciler.Create(context.TODO(), &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testName,
			Namespace: testNamespace,
		},
		Data: map[string][]byte{
			customproperties.DataKey: []byte(testValue),
		},
	})
	require.NoError(t, err)

	hash, err = reconciler.calculateCustomPropertyHash()
	assert.NoError(t, err)
	assert.NotEmpty(t, hash)
}
