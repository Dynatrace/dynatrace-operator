package statefulset

import (
	"context"
	"errors"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/activegate"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/communication"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/value"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/capability"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/internal/authtoken"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/internal/customproperties"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8sconditions"
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

func createDefaultReconciler(t *testing.T) (*Reconciler, client.WithWatch, *dynakube.DynaKube) {
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
			Namespace:   testNamespace,
			Name:        testName,
			Annotations: map[string]string{},
		}}

	r := NewReconciler(clt, clt, dk, capability.NewMultiCapability(dk))
	require.NotNil(t, r)

	return r.(*Reconciler), clt, dk
}

func getStatefulSet(t *testing.T, clt client.Client, dk *dynakube.DynaKube) *appsv1.StatefulSet {
	t.Helper()
	sts := &appsv1.StatefulSet{}
	err := clt.Get(t.Context(), client.ObjectKey{Name: capability.BuildServiceName(dk.Name), Namespace: dk.Namespace}, sts)
	require.NoError(t, err)

	return sts
}

func TestReconcile(t *testing.T) {
	assertCondition := func(t *testing.T, dk *dynakube.DynaKube, status metav1.ConditionStatus, reason, message string) {
		t.Helper()
		condition := meta.FindStatusCondition(dk.Status.Conditions, ActiveGateStatefulSetConditionType)
		require.NotNil(t, condition)
		assert.Equal(t, status, condition.Status)
		assert.Equal(t, reason, condition.Reason)
		assert.Equal(t, message, condition.Message)
	}

	t.Run("create statefulset", func(t *testing.T) {
		r, clt, dk := createDefaultReconciler(t)
		require.NoError(t, r.Reconcile(t.Context()))

		_ = getStatefulSet(t, clt, dk)
		assertCondition(t, dk, metav1.ConditionTrue, k8sconditions.StatefulSetCreatedReason, testName+"-activegate created")
	})
	t.Run("update statefulset", func(t *testing.T) {
		r, clt, dk := createDefaultReconciler(t)
		require.NoError(t, r.Reconcile(t.Context()))

		_ = getStatefulSet(t, clt, dk)

		dk.Spec.Proxy = &value.Source{Value: testValue}
		require.NoError(t, r.Reconcile(t.Context()))

		statefulSet := getStatefulSet(t, clt, dk)

		found := 0
		for _, vm := range statefulSet.Spec.Template.Spec.Containers[0].VolumeMounts {
			if vm.Name == InternalProxySecretVolumeName {
				found++
			}
		}

		assert.Equal(t, 1, found)
		assertCondition(t, dk, metav1.ConditionTrue, k8sconditions.StatefulSetCreatedReason, testName+"-activegate created")
	})
	t.Run("statefulset error is logged in condition", func(t *testing.T) {
		r, clt, dk := createDefaultReconciler(t)
		fakeClient := interceptor.NewClient(clt, interceptor.Funcs{
			Get: func(_ context.Context, _ client.WithWatch, _ client.ObjectKey, _ client.Object, _ ...client.GetOption) error {
				return errors.New("BOOM")
			},
		})
		r.apiReader = fakeClient

		err := r.Reconcile(t.Context())
		require.Error(t, err)

		assertCondition(t, dk, metav1.ConditionFalse, k8sconditions.KubeAPIErrorReason, "A problem occurred when using the Kubernetes API: "+err.Error())
	})
}

func TestReconcile_GetCustomPropertyHash(t *testing.T) {
	ctx := t.Context()
	r, clt, dk := createDefaultReconciler(t)
	hash, err := r.calculateActiveGateConfigurationHash(ctx)
	require.NoError(t, err)
	assert.NotEmpty(t, hash)

	dk.Spec.ActiveGate.CustomProperties = &value.Source{Value: testValue}
	hash, err = r.calculateActiveGateConfigurationHash(ctx)
	require.NoError(t, err)
	assert.NotEmpty(t, hash)

	dk.Spec.ActiveGate.CustomProperties = &value.Source{ValueFrom: testName}
	hash, err = r.calculateActiveGateConfigurationHash(ctx)
	require.Error(t, err)
	assert.Empty(t, hash)

	err = clt.Create(t.Context(), &corev1.Secret{
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
	ctx := t.Context()
	r, clt, _ := createDefaultReconciler(t)
	hash, err := r.calculateActiveGateConfigurationHash(ctx)
	require.NoError(t, err)
	assert.NotEmpty(t, hash)

	err = clt.Create(t.Context(), &corev1.Secret{
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
	ctx := t.Context()

	t.Run("do not delete statefulset if custom labels were added", func(t *testing.T) {
		r, clt, dk := createDefaultReconciler(t)

		err := r.manageStatefulSet(ctx)
		require.NoError(t, err)

		statefulSet := &appsv1.StatefulSet{ObjectMeta: metav1.ObjectMeta{Namespace: dk.Namespace, Name: capability.BuildServiceName(dk.Name)}}
		result, err := controllerutil.CreateOrUpdate(t.Context(), clt, statefulSet, func() error {
			statefulSet.Labels[testName] = testValue

			return nil
		})
		require.NoError(t, err)
		require.Equal(t, controllerutil.OperationResultUpdated, result)

		err = r.manageStatefulSet(ctx)
		require.NoError(t, err)

		actualStatefulSet := getStatefulSet(t, clt, dk)
		assert.Contains(t, actualStatefulSet.Labels, testName)
	})
	t.Run("update statefulset if selector differs", func(t *testing.T) {
		r, clt, dk := createDefaultReconciler(t)

		err := r.manageStatefulSet(ctx)
		require.NoError(t, err)

		statefulSet := &appsv1.StatefulSet{ObjectMeta: metav1.ObjectMeta{Namespace: dk.Namespace, Name: capability.BuildServiceName(dk.Name)}}
		result, err := controllerutil.CreateOrUpdate(t.Context(), clt, statefulSet, func() error {
			statefulSet.Spec.Selector.MatchLabels["activegate"] = testValue

			return nil
		})
		require.NoError(t, err)
		require.Equal(t, controllerutil.OperationResultUpdated, result)

		err = r.manageStatefulSet(ctx)
		require.NoError(t, err)

		actualStatefulSet := getStatefulSet(t, clt, dk)
		assert.Equal(t, testValue, actualStatefulSet.Spec.Selector.MatchLabels["activegate"])
	})
}

func TestStatefulSetUpdateWeakness(t *testing.T) {
	ctx := t.Context()

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
		},
		Status: dynakube.DynaKubeStatus{
			ActiveGate: activegate.Status{
				ConnectionInfo: communication.ConnectionInfo{
					TenantUUID: testTenantUUID,
				},
				VersionStatus: status.VersionStatus{},
			},
			KubeSystemUUID: testKubeSystemUUID,
		},
	}

	mcap := capability.NewMultiCapability(dk)
	reconciler := NewReconciler(clt, clt, dk, mcap)

	err := reconciler.Reconcile(ctx)
	require.NoError(t, err)

	dk.Spec.ActiveGate.UseEphemeralVolume = true
	err = reconciler.Reconcile(ctx)
	require.NoError(t, err)
}
