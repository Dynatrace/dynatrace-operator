package activegate

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/connectioninfo"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/conditions"
	dtclientmock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/clients/dynatrace"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

const (
	testName            = "test-name"
	testNamespace       = "test-namespace"
	testTenantToken     = "test-token"
	testTenantUUID      = "test-uuid"
	testTenantEndpoints = "test-endpoints"
	testOutdated        = "outdated"
)

func TestReconcile(t *testing.T) {
	ctx := context.Background()
	originalDynakube := dynatracev1beta1.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNamespace,
			Name:      testName,
		},
		Spec: dynatracev1beta1.DynaKubeSpec{
			ActiveGate: dynatracev1beta1.ActiveGateSpec{
				Capabilities: []dynatracev1beta1.CapabilityDisplayName{
					dynatracev1beta1.RoutingCapability.DisplayName},
			}},
	}

	dtc := dtclientmock.NewClient(t)
	dtc.On("GetActiveGateConnectionInfo", mock.AnythingOfType("context.backgroundCtx")).Return(getTestActiveGateConnectionInfo(), nil).Maybe()

	t.Run(`store ActiveGate connection info to DynaKube status`, func(t *testing.T) {
		dynakube := *originalDynakube.DeepCopy()
		fakeClient := fake.NewClient(&dynakube)
		r := NewReconciler(fakeClient, fakeClient, scheme.Scheme, dtc, &dynakube)
		err := r.Reconcile(ctx)
		require.NoError(t, err)

		condition := meta.FindStatusCondition(dynakube.Status.Conditions, activeGateConnectionInfoConditionType)
		require.NotNil(t, condition)
		assert.Equal(t, metav1.ConditionTrue, condition.Status)
		assert.Equal(t, conditions.SecretCreatedReason, condition.Reason)

		assert.Equal(t, testTenantUUID, dynakube.Status.ActiveGate.ConnectionInfoStatus.TenantUUID)
		assert.Equal(t, testTenantEndpoints, dynakube.Status.ActiveGate.ConnectionInfoStatus.Endpoints)
	})
	t.Run(`update ActiveGate connection info`, func(t *testing.T) {
		dynakube := *originalDynakube.DeepCopy()
		fakeClient := fake.NewClient(&dynakube)
		dynakube.Status.ActiveGate.ConnectionInfoStatus = dynatracev1beta1.ActiveGateConnectionInfoStatus{
			ConnectionInfoStatus: dynatracev1beta1.ConnectionInfoStatus{
				TenantUUID:  testOutdated,
				Endpoints:   testOutdated,
				LastRequest: metav1.NewTime(time.Now()),
			},
		}
		resetCachedTimestamps(&dynakube.Status)

		r := NewReconciler(fakeClient, fakeClient, scheme.Scheme, dtc, &dynakube)
		err := r.Reconcile(ctx)
		require.NoError(t, err)

		condition := meta.FindStatusCondition(dynakube.Status.Conditions, activeGateConnectionInfoConditionType)
		assert.Equal(t, metav1.ConditionTrue, condition.Status)
		assert.Equal(t, conditions.SecretCreatedReason, condition.Reason)

		assert.Equal(t, testTenantUUID, dynakube.Status.ActiveGate.ConnectionInfoStatus.TenantUUID)
		assert.Equal(t, testTenantEndpoints, dynakube.Status.ActiveGate.ConnectionInfoStatus.Endpoints)
	})
	t.Run(`update ActiveGate connection info if tenant secret is missing, ignore timestamp`, func(t *testing.T) {
		dynakube := *originalDynakube.DeepCopy()
		fakeClient := fake.NewClient(&dynakube)
		dynakube.Status.ActiveGate.ConnectionInfoStatus = dynatracev1beta1.ActiveGateConnectionInfoStatus{
			ConnectionInfoStatus: dynatracev1beta1.ConnectionInfoStatus{
				TenantUUID:  testOutdated,
				Endpoints:   testOutdated,
				LastRequest: metav1.NewTime(time.Now()),
			},
		}

		r := NewReconciler(fakeClient, fakeClient, scheme.Scheme, dtc, &dynakube)
		err := r.Reconcile(ctx)
		require.NoError(t, err)

		condition := meta.FindStatusCondition(dynakube.Status.Conditions, activeGateConnectionInfoConditionType)
		assert.Equal(t, metav1.ConditionTrue, condition.Status)
		assert.Equal(t, conditions.SecretCreatedReason, condition.Reason)

		assert.Equal(t, testTenantUUID, dynakube.Status.ActiveGate.ConnectionInfoStatus.TenantUUID)
		assert.Equal(t, testTenantEndpoints, dynakube.Status.ActiveGate.ConnectionInfoStatus.Endpoints)
	})
	t.Run(`ActiveGate connection info error shown in conditions`, func(t *testing.T) {
		dynakube := *originalDynakube.DeepCopy()
		fakeClient := fake.NewClientWithInterceptors(interceptor.Funcs{
			Get: func(ctx context.Context, client client.WithWatch, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
				return fmt.Errorf("BOOM")
			},
		})
		r := NewReconciler(fakeClient, fakeClient, scheme.Scheme, dtc, &dynakube)
		err := r.Reconcile(ctx)
		require.Error(t, err)

		condition := meta.FindStatusCondition(dynakube.Status.Conditions, activeGateConnectionInfoConditionType)
		assert.Equal(t, metav1.ConditionFalse, condition.Status)
		assert.Equal(t, conditions.KubeApiErrorReason, condition.Reason)
		assert.Equal(t, "A problem occurred when using the Kubernetes API: "+err.Error(), condition.Message)
	})
}

func getTestActiveGateConnectionInfo() dtclient.ActiveGateConnectionInfo {
	return dtclient.ActiveGateConnectionInfo{
		ConnectionInfo: dtclient.ConnectionInfo{
			TenantUUID:  testTenantUUID,
			TenantToken: testTenantToken,
			Endpoints:   testTenantEndpoints,
		},
	}
}

func resetCachedTimestamps(dynakubeStatus *dynatracev1beta1.DynaKubeStatus) {
	dynakubeStatus.DynatraceApi.LastTokenScopeRequest = metav1.Time{}
	dynakubeStatus.ActiveGate.ConnectionInfoStatus.LastRequest = metav1.Time{}

	if dynakubeStatus.Conditions != nil {
		for i := range dynakubeStatus.Conditions {
			dynakubeStatus.Conditions[i].LastTransitionTime = metav1.Time{}
		}
	}
}

func TestReconcile_TenantSecret(t *testing.T) {
	ctx := context.Background()
	dynakube := &dynatracev1beta1.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNamespace,
			Name:      testName,
		},
		Spec: dynatracev1beta1.DynaKubeSpec{
			ActiveGate: dynatracev1beta1.ActiveGateSpec{
				Capabilities: []dynatracev1beta1.CapabilityDisplayName{
					dynatracev1beta1.RoutingCapability.DisplayName,
				},
			},
		},
	}
	dtc := dtclientmock.NewClient(t)
	dtc.On("GetActiveGateConnectionInfo", mock.AnythingOfType("context.backgroundCtx")).Return(getTestActiveGateConnectionInfo(), nil)

	t.Run(`create activegate secret`, func(t *testing.T) {
		fakeClient := fake.NewClient(dynakube)

		r := NewReconciler(fakeClient, fakeClient, scheme.Scheme, dtc, dynakube)
		err := r.Reconcile(ctx)
		require.NoError(t, err)

		var actualSecret corev1.Secret
		err = fakeClient.Get(context.Background(), client.ObjectKey{Name: dynakube.ActivegateTenantSecret(), Namespace: testNamespace}, &actualSecret)
		require.NoError(t, err)
		assert.Equal(t, []byte(testTenantToken), actualSecret.Data[connectioninfo.TenantTokenKey])
	})
	t.Run(`update activegate secret`, func(t *testing.T) {
		fakeClient := fake.NewClient(dynakube,
			buildActiveGateSecret(*dynakube, testOutdated))

		resetCachedTimestamps(&dynakube.Status)

		r := NewReconciler(fakeClient, fakeClient, scheme.Scheme, dtc, dynakube)
		err := r.Reconcile(ctx)
		require.NoError(t, err)

		var actualSecret corev1.Secret
		err = fakeClient.Get(context.Background(), client.ObjectKey{Name: dynakube.ActivegateTenantSecret(), Namespace: testNamespace}, &actualSecret)
		require.NoError(t, err)
		assert.Equal(t, []byte(testTenantToken), actualSecret.Data[connectioninfo.TenantTokenKey])
	})
	t.Run(`check activegate secret caches`, func(t *testing.T) {
		fakeClient := fake.NewClient(dynakube, buildActiveGateSecret(*dynakube, testOutdated))
		r := NewReconciler(fakeClient, fakeClient, scheme.Scheme, dtc, dynakube)
		err := r.Reconcile(ctx)
		require.NoError(t, err)

		var actualSecret corev1.Secret
		err = fakeClient.Get(context.Background(), client.ObjectKey{Name: dynakube.ActivegateTenantSecret(), Namespace: testNamespace}, &actualSecret)
		require.NoError(t, err)
		assert.Equal(t, []byte(testOutdated), actualSecret.Data[connectioninfo.TenantTokenKey])
	})
	t.Run(`up to date activegate secret`, func(t *testing.T) {
		fakeClient := fake.NewClient(dynakube, buildActiveGateSecret(*dynakube, testTenantToken))

		r := NewReconciler(fakeClient, fakeClient, scheme.Scheme, dtc, dynakube)
		err := r.Reconcile(ctx)
		require.NoError(t, err)
	})
}

func buildActiveGateSecret(dynakube dynatracev1beta1.DynaKube, token string) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      dynakube.ActivegateTenantSecret(),
			Namespace: testNamespace,
		},
		Data: map[string][]byte{
			connectioninfo.TenantTokenKey: []byte(token),
		},
	}
}
