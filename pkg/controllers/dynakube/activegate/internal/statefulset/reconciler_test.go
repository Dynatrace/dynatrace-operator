package statefulset

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme"
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/capability"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/internal/authtoken"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/internal/customproperties"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubesystem"
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
	testName      = "test-name"
	testNamespace = "test-namespace"
	testValue     = "test-value"
	testUID       = "test-uid"
	testToken     = "test-token"
)

func TestNewReconciler(t *testing.T) {
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

	r := NewReconciler(clt, clt, scheme.Scheme, instance, capability.NewRoutingCapability(instance)).(*Reconciler)
	r.dynakube.Annotations = map[string]string{}
	require.NotNil(t, r)
	require.NotNil(t, r.client)
	require.NotNil(t, r.scheme)
	require.NotNil(t, r.dynakube)

	return r
}

func TestReconcile(t *testing.T) {
	t.Run(`create stateful set`, func(t *testing.T) {
		r := createDefaultReconciler(t)
		err := r.Reconcile(context.Background())

		assert.NoError(t, err)

		statefulSet := &appsv1.StatefulSet{}
		err = r.client.Get(context.Background(), client.ObjectKey{Name: r.dynakube.Name + "-" + r.capability.ShortName(), Namespace: r.dynakube.Namespace}, statefulSet)

		assert.NotNil(t, statefulSet)
		assert.NoError(t, err)
	})
	t.Run(`update stateful set`, func(t *testing.T) {
		r := createDefaultReconciler(t)
		err := r.Reconcile(context.Background())

		assert.NoError(t, err)

		statefulSet := &appsv1.StatefulSet{}
		err = r.client.Get(context.Background(), client.ObjectKey{Name: r.dynakube.Name + "-" + r.capability.ShortName(), Namespace: r.dynakube.Namespace}, statefulSet)

		assert.NotNil(t, statefulSet)
		assert.NoError(t, err)

		r.dynakube.Spec.Proxy = &dynatracev1beta1.DynaKubeProxy{Value: testValue}
		err = r.Reconcile(context.Background())

		assert.NoError(t, err)

		newStatefulSet := &appsv1.StatefulSet{}
		err = r.client.Get(context.Background(), client.ObjectKey{Name: r.dynakube.Name + "-" + r.capability.ShortName(), Namespace: r.dynakube.Namespace}, newStatefulSet)

		assert.NotNil(t, statefulSet)
		assert.NoError(t, err)

		found := 0
		for _, vm := range newStatefulSet.Spec.Template.Spec.Containers[0].VolumeMounts {
			if vm.Name == InternalProxySecretVolumeName {
				found++
			}
		}
		assert.Equal(t, 4, found)
	})
}

func TestReconcile_GetStatefulSet(t *testing.T) {
	r := createDefaultReconciler(t)
	err := r.Reconcile(context.Background())
	assert.NoError(t, err)

	desiredSts, err := r.buildDesiredStatefulSet(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, desiredSts)

	desiredSts.Kind = "StatefulSet"
	desiredSts.APIVersion = "apps/v1"
	desiredSts.ResourceVersion = "1"
	err = controllerutil.SetControllerReference(r.dynakube, desiredSts, r.scheme)
	require.NoError(t, err)

	sts, err := r.getStatefulSet(context.Background(), desiredSts)
	assert.NoError(t, err)
	assert.Equal(t, *desiredSts, *sts)
}

func TestReconcile_CreateStatefulSetIfNotExists(t *testing.T) {
	r := createDefaultReconciler(t)
	desiredSts, err := r.buildDesiredStatefulSet(context.Background())
	require.NoError(t, err)
	require.NotNil(t, desiredSts)

	created, err := r.createStatefulSetIfNotExists(context.Background(), desiredSts)
	assert.NoError(t, err)
	assert.True(t, created)

	created, err = r.createStatefulSetIfNotExists(context.Background(), desiredSts)
	assert.NoError(t, err)
	assert.False(t, created)
}

func TestReconcile_UpdateStatefulSetIfOutdated(t *testing.T) {
	r := createDefaultReconciler(t)
	desiredSts, err := r.buildDesiredStatefulSet(context.Background())
	require.NoError(t, err)
	require.NotNil(t, desiredSts)

	updated, err := r.updateStatefulSetIfOutdated(context.Background(), desiredSts)
	assert.Error(t, err)
	assert.False(t, updated)
	assert.True(t, k8serrors.IsNotFound(errors.Cause(err)))

	created, err := r.createStatefulSetIfNotExists(context.Background(), desiredSts)
	require.True(t, created)
	require.NoError(t, err)

	updated, err = r.updateStatefulSetIfOutdated(context.Background(), desiredSts)
	assert.NoError(t, err)
	assert.False(t, updated)

	r.dynakube.Spec.Proxy = &dynatracev1beta1.DynaKubeProxy{Value: testValue}
	desiredSts, err = r.buildDesiredStatefulSet(context.Background())
	require.NoError(t, err)

	updated, err = r.updateStatefulSetIfOutdated(context.Background(), desiredSts)
	assert.NoError(t, err)
	assert.True(t, updated)
}

func TestReconcile_DeleteStatefulSetIfOldLabelsAreUsed(t *testing.T) {
	t.Run("statefulset is deleted when old labels are used", func(t *testing.T) {
		r := createDefaultReconciler(t)
		desiredSts, err := r.buildDesiredStatefulSet(context.Background())
		require.NoError(t, err)
		require.NotNil(t, desiredSts)

		deleted, err := r.deleteStatefulSetIfSelectorChanged(context.Background(), desiredSts)
		assert.Error(t, err)
		assert.False(t, deleted)
		assert.True(t, k8serrors.IsNotFound(errors.Cause(err)))

		created, err := r.createStatefulSetIfNotExists(context.Background(), desiredSts)
		require.True(t, created)
		require.NoError(t, err)

		deleted, err = r.deleteStatefulSetIfSelectorChanged(context.Background(), desiredSts)
		assert.NoError(t, err)
		assert.False(t, deleted)

		r.dynakube.Spec.Proxy = &dynatracev1beta1.DynaKubeProxy{Value: testValue}
		desiredSts, err = r.buildDesiredStatefulSet(context.Background())
		require.NoError(t, err)

		correctLabels := desiredSts.Spec.Selector.MatchLabels
		desiredSts.Spec.Selector.MatchLabels = map[string]string{"activegate": "dynakube"}
		err = r.client.Update(context.Background(), desiredSts)
		assert.NoError(t, err)

		desiredSts.Spec.Selector.MatchLabels = correctLabels
		deleted, err = r.deleteStatefulSetIfSelectorChanged(context.Background(), desiredSts)
		assert.NoError(t, err)
		assert.True(t, deleted)
	})
	t.Run("statefulset is not deleted when custom labels are used", func(t *testing.T) {
		r := createDefaultReconciler(t)
		appliedStatefulset, err := r.buildDesiredStatefulSet(context.Background())

		require.NoError(t, err)
		require.NotNil(t, appliedStatefulset)

		created, err := r.createStatefulSetIfNotExists(context.Background(), appliedStatefulset)

		require.True(t, created)
		require.NoError(t, err)

		appliedStatefulset.Labels[testName] = testValue
		err = r.client.Update(context.Background(), appliedStatefulset)

		require.NoError(t, err)

		desiredStatefulset, err := r.buildDesiredStatefulSet(context.Background())

		require.NoError(t, err)

		deleted, err := r.deleteStatefulSetIfSelectorChanged(context.Background(), desiredStatefulset)

		assert.NoError(t, err)
		assert.False(t, deleted)
	})
}

func TestReconcile_GetCustomPropertyHash(t *testing.T) {
	r := createDefaultReconciler(t)
	hash, err := r.calculateActiveGateConfigurationHash()
	assert.NoError(t, err)
	assert.NotEmpty(t, hash)

	r.dynakube.Spec.Routing.CustomProperties = &dynatracev1beta1.DynaKubeValueSource{Value: testValue}
	hash, err = r.calculateActiveGateConfigurationHash()
	assert.NoError(t, err)
	assert.NotEmpty(t, hash)

	r.dynakube.Spec.Routing.CustomProperties = &dynatracev1beta1.DynaKubeValueSource{ValueFrom: testName}
	hash, err = r.calculateActiveGateConfigurationHash()
	assert.Error(t, err)
	assert.Empty(t, hash)

	err = r.client.Create(context.Background(), &corev1.Secret{
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

	err = r.client.Create(context.Background(), &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      r.dynakube.ActiveGateAuthTokenSecret(),
			Namespace: r.dynakube.Namespace,
		},
		Data: map[string][]byte{
			authtoken.ActiveGateAuthTokenName: []byte(testValue),
		},
	})
	require.Error(t, err)
}

func TestManageStatefulSet(t *testing.T) {
	t.Run("do not delete statefulset if custom labels were added", func(t *testing.T) {
		r := createDefaultReconciler(t)
		desiredStatefulSet, err := r.buildDesiredStatefulSet(context.Background())

		require.NoError(t, err)

		err = r.manageStatefulSet(context.Background())
		assert.NoError(t, err)

		actualStatefulSet, err := r.getStatefulSet(context.Background(), desiredStatefulSet)
		assert.NoError(t, err)
		assert.NotNil(t, actualStatefulSet)

		actualStatefulSet.Labels[testName] = testValue
		err = r.client.Update(context.Background(), actualStatefulSet)

		require.NoError(t, err)

		err = r.manageStatefulSet(context.Background())
		assert.NoError(t, err)

		actualStatefulSet, err = r.getStatefulSet(context.Background(), desiredStatefulSet)
		assert.NoError(t, err)
		assert.NotNil(t, actualStatefulSet)
		assert.Contains(t, actualStatefulSet.Labels, testName)
	})
	t.Run("delete statefulset if selector differs", func(t *testing.T) {
		r := createDefaultReconciler(t)
		desiredStatefulSet, err := r.buildDesiredStatefulSet(context.Background())

		require.NoError(t, err)

		err = r.manageStatefulSet(context.Background())
		assert.NoError(t, err)

		actualStatefulSet, err := r.getStatefulSet(context.Background(), desiredStatefulSet)
		assert.NoError(t, err)
		assert.NotNil(t, actualStatefulSet)

		actualStatefulSet.Spec.Selector.MatchLabels["activegate"] = testValue
		err = r.client.Update(context.Background(), actualStatefulSet)

		require.NoError(t, err)

		err = r.manageStatefulSet(context.Background())
		assert.NoError(t, err)

		actualStatefulSet, err = r.getStatefulSet(context.Background(), desiredStatefulSet)
		assert.Error(t, err)
		assert.Nil(t, actualStatefulSet)
		assert.True(t, k8serrors.IsNotFound(err))
	})
}
