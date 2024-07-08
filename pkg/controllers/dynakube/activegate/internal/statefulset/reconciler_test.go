package statefulset

import (
	"context"
	"fmt"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme"
	dynafake "github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/capability"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/internal/authtoken"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/internal/customproperties"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/conditions"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubesystem"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
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
				Name:      testName + dynakube.AuthTokenSecretSuffix,
				Namespace: testNamespace,
			},
			Data: map[string][]byte{authtoken.ActiveGateAuthTokenName: []byte(testToken)},
		}).
		Build()
	instance := &dynakube.DynaKube{
		Spec: dynakube.DynaKubeSpec{
			ActiveGate: dynakube.ActiveGateSpec{
				Capabilities: []dynakube.CapabilityDisplayName{
					dynakube.RoutingCapability.DisplayName,
				}},
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNamespace,
			Name:      testName,
		}}

	capability.NewMultiCapability(dk)

	r := NewReconciler(clt, clt, instance, capability.NewMultiCapability(instance)).(*Reconciler)
	r.dk.Annotations = map[string]string{}
	require.NotNil(t, r)
	require.NotNil(t, r.client)
	require.NotNil(t, r.dk)

	return r
}

func TestReconcile(t *testing.T) {
	t.Run(`create stateful set`, func(t *testing.T) {
		r := createDefaultReconciler(t)
		err := r.Reconcile(context.Background())

		require.NoError(t, err)

		statefulSet := &appsv1.StatefulSet{}
		err = r.client.Get(context.Background(), client.ObjectKey{Name: r.dk.Name + "-" + r.capability.ShortName(), Namespace: r.dk.Namespace}, statefulSet)

		assert.NotNil(t, statefulSet)
		require.NoError(t, err)

		condition := meta.FindStatusCondition(r.dk.Status.Conditions, ActiveGateStatefulSetConditionType)
		assert.Equal(t, metav1.ConditionTrue, condition.Status)
		assert.Equal(t, conditions.StatefulSetCreatedReason, condition.Reason)
		assert.Equal(t, fmt.Sprintf("%s-activegate created", testName), condition.Message)
	})
	t.Run(`update stateful set`, func(t *testing.T) {
		r := createDefaultReconciler(t)
		err := r.Reconcile(context.Background())

		require.NoError(t, err)

		statefulSet := &appsv1.StatefulSet{}
		err = r.client.Get(context.Background(), client.ObjectKey{Name: r.dk.Name + "-" + r.capability.ShortName(), Namespace: r.dk.Namespace}, statefulSet)

		assert.NotNil(t, statefulSet)
		require.NoError(t, err)

		r.dk.Spec.Proxy = &dynakube.DynaKubeProxy{Value: testValue}
		err = r.Reconcile(context.Background())

		require.NoError(t, err)

		newStatefulSet := &appsv1.StatefulSet{}
		err = r.client.Get(context.Background(), client.ObjectKey{Name: r.dk.Name + "-" + r.capability.ShortName(), Namespace: r.dk.Namespace}, newStatefulSet)

		assert.NotNil(t, statefulSet)
		require.NoError(t, err)

		found := 0

		for _, vm := range newStatefulSet.Spec.Template.Spec.Containers[0].VolumeMounts {
			if vm.Name == InternalProxySecretVolumeName {
				found++
			}
		}

		assert.Equal(t, 1, found)

		condition := meta.FindStatusCondition(r.dk.Status.Conditions, ActiveGateStatefulSetConditionType)
		assert.Equal(t, metav1.ConditionTrue, condition.Status)
		assert.Equal(t, conditions.StatefulSetUpdatedReason, condition.Reason)
		assert.Equal(t, testName+"-activegate updated", condition.Message)
	})
	t.Run(`stateful set error is logged in condition`, func(t *testing.T) {
		r := createDefaultReconciler(t)
		fakeClient := dynafake.NewClientWithInterceptors(interceptor.Funcs{
			Get: func(ctx context.Context, client client.WithWatch, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
				return fmt.Errorf("BOOM")
			},
		})
		r.client = fakeClient
		err := r.Reconcile(context.Background())

		require.Error(t, err)

		condition := meta.FindStatusCondition(r.dk.Status.Conditions, ActiveGateStatefulSetConditionType)
		assert.Equal(t, metav1.ConditionFalse, condition.Status)
		assert.Equal(t, conditions.KubeApiErrorReason, condition.Reason)
		assert.Equal(t, "A problem occurred when using the Kubernetes API: "+err.Error(), condition.Message)
	})
}

func TestReconcile_GetStatefulSet(t *testing.T) {
	r := createDefaultReconciler(t)
	err := r.Reconcile(context.Background())
	require.NoError(t, err)

	desiredSts, err := r.buildDesiredStatefulSet(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, desiredSts)

	desiredSts.Kind = "StatefulSet"
	desiredSts.APIVersion = "apps/v1"
	desiredSts.ResourceVersion = "1"
	err = controllerutil.SetControllerReference(r.dk, desiredSts, scheme.Scheme)
	require.NoError(t, err)

	sts, err := r.getStatefulSet(context.Background(), desiredSts)
	require.NoError(t, err)
	assert.Equal(t, *desiredSts, *sts)
}

func TestReconcile_CreateStatefulSetIfNotExists(t *testing.T) {
	r := createDefaultReconciler(t)
	desiredSts, err := r.buildDesiredStatefulSet(context.Background())
	require.NoError(t, err)
	require.NotNil(t, desiredSts)

	created, err := r.createStatefulSetIfNotExists(context.Background(), desiredSts)
	require.NoError(t, err)
	assert.True(t, created)

	created, err = r.createStatefulSetIfNotExists(context.Background(), desiredSts)
	require.NoError(t, err)
	assert.False(t, created)
}

func TestReconcile_UpdateStatefulSetIfOutdated(t *testing.T) {
	r := createDefaultReconciler(t)
	desiredSts, err := r.buildDesiredStatefulSet(context.Background())
	require.NoError(t, err)
	require.NotNil(t, desiredSts)

	updated, err := r.updateStatefulSetIfOutdated(context.Background(), desiredSts)
	require.Error(t, err)
	assert.False(t, updated)
	assert.True(t, k8serrors.IsNotFound(errors.Cause(err)))

	created, err := r.createStatefulSetIfNotExists(context.Background(), desiredSts)
	require.True(t, created)
	require.NoError(t, err)

	updated, err = r.updateStatefulSetIfOutdated(context.Background(), desiredSts)
	require.NoError(t, err)
	assert.False(t, updated)

	r.dk.Spec.Proxy = &dynakube.DynaKubeProxy{Value: testValue}
	desiredSts, err = r.buildDesiredStatefulSet(context.Background())
	require.NoError(t, err)

	updated, err = r.updateStatefulSetIfOutdated(context.Background(), desiredSts)
	require.NoError(t, err)
	assert.True(t, updated)
}

func TestReconcile_DeleteStatefulSetIfOldLabelsAreUsed(t *testing.T) {
	t.Run("statefulset is deleted when old labels are used", func(t *testing.T) {
		r := createDefaultReconciler(t)
		desiredSts, err := r.buildDesiredStatefulSet(context.Background())
		require.NoError(t, err)
		require.NotNil(t, desiredSts)

		deleted, err := r.deleteStatefulSetIfSelectorChanged(context.Background(), desiredSts)
		require.Error(t, err)
		assert.False(t, deleted)
		assert.True(t, k8serrors.IsNotFound(errors.Cause(err)))

		created, err := r.createStatefulSetIfNotExists(context.Background(), desiredSts)
		require.True(t, created)
		require.NoError(t, err)

		deleted, err = r.deleteStatefulSetIfSelectorChanged(context.Background(), desiredSts)
		require.NoError(t, err)
		assert.False(t, deleted)

		r.dk.Spec.Proxy = &dynakube.DynaKubeProxy{Value: testValue}
		desiredSts, err = r.buildDesiredStatefulSet(context.Background())
		require.NoError(t, err)

		correctLabels := desiredSts.Spec.Selector.MatchLabels
		desiredSts.Spec.Selector.MatchLabels = map[string]string{"activegate": "dynakube"}
		err = r.client.Update(context.Background(), desiredSts)
		require.NoError(t, err)

		desiredSts.Spec.Selector.MatchLabels = correctLabels
		deleted, err = r.deleteStatefulSetIfSelectorChanged(context.Background(), desiredSts)
		require.NoError(t, err)
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

		require.NoError(t, err)
		assert.False(t, deleted)
	})
}

func TestReconcile_GetCustomPropertyHash(t *testing.T) {
	ctx := context.Background()
	r := createDefaultReconciler(t)
	hash, err := r.calculateActiveGateConfigurationHash(ctx)
	require.NoError(t, err)
	assert.NotEmpty(t, hash)

	r.dk.Spec.ActiveGate.CustomProperties = &dynakube.DynaKubeValueSource{Value: testValue}
	hash, err = r.calculateActiveGateConfigurationHash(ctx)
	require.NoError(t, err)
	assert.NotEmpty(t, hash)

	r.dk.Spec.ActiveGate.CustomProperties = &dynakube.DynaKubeValueSource{ValueFrom: testName}
	hash, err = r.calculateActiveGateConfigurationHash(ctx)
	require.Error(t, err)
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

	hash, err = r.calculateActiveGateConfigurationHash(ctx)
	require.NoError(t, err)
	assert.NotEmpty(t, hash)
}

func TestReconcile_GetActiveGateAuthTokenHash(t *testing.T) {
	ctx := context.Background()
	r := createDefaultReconciler(t)
	hash, err := r.calculateActiveGateConfigurationHash(ctx)
	require.NoError(t, err)
	assert.NotEmpty(t, hash)

	err = r.client.Create(context.Background(), &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      r.dk.ActiveGateAuthTokenSecret(),
			Namespace: r.dk.Namespace,
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
		require.NoError(t, err)

		actualStatefulSet, err := r.getStatefulSet(context.Background(), desiredStatefulSet)
		require.NoError(t, err)
		assert.NotNil(t, actualStatefulSet)

		actualStatefulSet.Labels[testName] = testValue
		err = r.client.Update(context.Background(), actualStatefulSet)

		require.NoError(t, err)

		err = r.manageStatefulSet(context.Background())
		require.NoError(t, err)

		actualStatefulSet, err = r.getStatefulSet(context.Background(), desiredStatefulSet)
		require.NoError(t, err)
		assert.NotNil(t, actualStatefulSet)
		assert.Contains(t, actualStatefulSet.Labels, testName)
	})
	t.Run("update statefulset if selector differs", func(t *testing.T) {
		r := createDefaultReconciler(t)
		desiredStatefulSet, err := r.buildDesiredStatefulSet(context.Background())

		require.NoError(t, err)

		err = r.manageStatefulSet(context.Background())
		require.NoError(t, err)

		actualStatefulSet, err := r.getStatefulSet(context.Background(), desiredStatefulSet)
		require.NoError(t, err)
		assert.NotNil(t, actualStatefulSet)

		actualStatefulSet.Spec.Selector.MatchLabels["activegate"] = testValue
		err = r.client.Update(context.Background(), actualStatefulSet)

		require.NoError(t, err)

		err = r.manageStatefulSet(context.Background())
		require.NoError(t, err)

		actualStatefulSet, err = r.getStatefulSet(context.Background(), desiredStatefulSet)
		require.NoError(t, err)

		_, ok := actualStatefulSet.Spec.Selector.MatchLabels["activegate"]
		assert.False(t, ok)
	})
}
