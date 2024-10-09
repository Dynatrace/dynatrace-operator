package statefulset

import (
	"context"
	"fmt"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme"
	dynafake "github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/value"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube/activegate"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/capability"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/internal/authtoken"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/internal/customproperties"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/conditions"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/statefulset"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubesystem"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
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
				Name:      testName + activegate.AuthTokenSecretSuffix,
				Namespace: testNamespace,
			},
			Data: map[string][]byte{authtoken.ActiveGateAuthTokenName: []byte(testToken)},
		}).
		Build()
	dk := &dynakube.DynaKube{
		Spec: dynakube.DynaKubeSpec{
			ActiveGate: activegate.Spec{
				Capabilities: []activegate.CapabilityDisplayName{
					activegate.RoutingCapability.DisplayName,
				}},
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNamespace,
			Name:      testName,
		}}

	capability.NewMultiCapability(dk)

	r := NewReconciler(clt, clt, dk, capability.NewMultiCapability(dk)).(*Reconciler)
	r.dk.Annotations = map[string]string{}
	require.NotNil(t, r)
	require.NotNil(t, r.client)
	require.NotNil(t, r.dk)

	return r
}

func TestReconcile(t *testing.T) {
	ctx := context.Background()

	t.Run("create stateful set", func(t *testing.T) {
		r := createDefaultReconciler(t)
		err := r.Reconcile(ctx)

		require.NoError(t, err)

		statefulSet := &appsv1.StatefulSet{}
		err = r.client.Get(ctx, client.ObjectKey{Name: r.dk.Name + "-" + r.capability.ShortName(), Namespace: r.dk.Namespace}, statefulSet)

		assert.NotNil(t, statefulSet)
		require.NoError(t, err)

		condition := meta.FindStatusCondition(r.dk.Status.Conditions, ActiveGateStatefulSetConditionType)
		assert.Equal(t, metav1.ConditionTrue, condition.Status)
		assert.Equal(t, conditions.StatefulSetCreatedReason, condition.Reason)
		assert.Equal(t, fmt.Sprintf("%s-activegate created", testName), condition.Message)
	})
	t.Run("update stateful set", func(t *testing.T) {
		r := createDefaultReconciler(t)
		err := r.Reconcile(ctx)

		require.NoError(t, err)

		statefulSet := &appsv1.StatefulSet{}
		err = r.client.Get(ctx, client.ObjectKey{Name: r.dk.Name + "-" + r.capability.ShortName(), Namespace: r.dk.Namespace}, statefulSet)

		assert.NotNil(t, statefulSet)
		require.NoError(t, err)

		r.dk.Spec.Proxy = &value.Source{Value: testValue}
		err = r.Reconcile(ctx)

		require.NoError(t, err)

		newStatefulSet := &appsv1.StatefulSet{}
		err = r.client.Get(ctx, client.ObjectKey{Name: r.dk.Name + "-" + r.capability.ShortName(), Namespace: r.dk.Namespace}, newStatefulSet)

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
		assert.Equal(t, conditions.StatefulSetCreatedReason, condition.Reason)
		assert.Equal(t, testName+"-activegate created", condition.Message)
	})
	t.Run("stateful set error is logged in condition", func(t *testing.T) {
		r := createDefaultReconciler(t)
		fakeClient := dynafake.NewClientWithInterceptors(interceptor.Funcs{
			Get: func(ctx context.Context, client client.WithWatch, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
				return fmt.Errorf("BOOM")
			},
		})
		r.apiReader = fakeClient

		err := r.Reconcile(ctx)
		require.Error(t, err)

		condition := meta.FindStatusCondition(r.dk.Status.Conditions, ActiveGateStatefulSetConditionType)
		require.NotNil(t, condition)
		assert.Equal(t, metav1.ConditionFalse, condition.Status)
		assert.Equal(t, conditions.KubeApiErrorReason, condition.Reason)
		assert.Equal(t, "A problem occurred when using the Kubernetes API: "+err.Error(), condition.Message)
	})
}

func TestReconcile_GetCustomPropertyHash(t *testing.T) {
	ctx := context.Background()
	r := createDefaultReconciler(t)
	hash, err := r.calculateActiveGateConfigurationHash(ctx)
	require.NoError(t, err)
	assert.NotEmpty(t, hash)

	r.dk.Spec.ActiveGate.CustomProperties = &value.Source{Value: testValue}
	hash, err = r.calculateActiveGateConfigurationHash(ctx)
	require.NoError(t, err)
	assert.NotEmpty(t, hash)

	r.dk.Spec.ActiveGate.CustomProperties = &value.Source{ValueFrom: testName}
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
			Name:      r.dk.ActiveGate().GetAuthTokenSecretName(),
			Namespace: r.dk.Namespace,
		},
		Data: map[string][]byte{
			authtoken.ActiveGateAuthTokenName: []byte(testValue),
		},
	})
	require.Error(t, err)
}

func TestManageStatefulSet(t *testing.T) {
	ctx := context.Background()

	t.Run("do not delete statefulset if custom labels were added", func(t *testing.T) {
		r := createDefaultReconciler(t)
		desiredStatefulSet, err := r.buildDesiredStatefulSet(ctx)

		require.NoError(t, err)

		err = r.manageStatefulSet(ctx)
		require.NoError(t, err)

		actualStatefulSet, err := statefulset.Query(r.client, r.apiReader, log).Get(ctx, client.ObjectKeyFromObject(desiredStatefulSet))
		require.NoError(t, err)
		assert.NotNil(t, actualStatefulSet)

		actualStatefulSet.Labels[testName] = testValue
		err = r.client.Update(ctx, actualStatefulSet)

		require.NoError(t, err)

		err = r.manageStatefulSet(ctx)
		require.NoError(t, err)

		actualStatefulSet, err = statefulset.Query(r.client, r.apiReader, log).Get(ctx, client.ObjectKeyFromObject(desiredStatefulSet))
		require.NoError(t, err)
		assert.NotNil(t, actualStatefulSet)
		assert.Contains(t, actualStatefulSet.Labels, testName)
	})
	t.Run("update statefulset if selector differs", func(t *testing.T) {
		r := createDefaultReconciler(t)
		desiredStatefulSet, err := r.buildDesiredStatefulSet(ctx)

		require.NoError(t, err)

		err = r.manageStatefulSet(ctx)
		require.NoError(t, err)

		actualStatefulSet, err := statefulset.Query(r.client, r.apiReader, log).Get(ctx, client.ObjectKeyFromObject(desiredStatefulSet))
		require.NoError(t, err)
		assert.NotNil(t, actualStatefulSet)

		actualStatefulSet.Spec.Selector.MatchLabels["activegate"] = testValue
		err = r.client.Update(ctx, actualStatefulSet)

		require.NoError(t, err)

		err = r.manageStatefulSet(ctx)
		require.NoError(t, err)

		actualStatefulSet, err = statefulset.Query(r.client, r.apiReader, log).Get(ctx, client.ObjectKeyFromObject(desiredStatefulSet))
		require.NoError(t, err)

		labelValue, ok := actualStatefulSet.Spec.Selector.MatchLabels["activegate"]
		require.True(t, ok)
		assert.Equal(t, testValue, labelValue)
	})
}
