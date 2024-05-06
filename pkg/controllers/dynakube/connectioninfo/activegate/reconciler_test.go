package activegate

import (
	"context"
	"fmt"
	"testing"
	"time"

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
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
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

	t.Run("cleanup when activegate is not needed", func(t *testing.T) {
		dynakube := getTestDynakube()
		dynakube.Spec = dynatracev1beta1.DynaKubeSpec{}
		meta.SetStatusCondition(&dynakube.Status.Conditions, metav1.Condition{
			Type:   activeGateConnectionInfoConditionType,
			Status: metav1.ConditionTrue,
		})

		fakeClient := fake.NewClient(buildActiveGateSecret(*dynakube, testTenantUUID))
		dtc := dtclientmock.NewClient(t)

		r := NewReconciler(fakeClient, fakeClient, dtc, dynakube)
		err := r.Reconcile(ctx)
		require.NoError(t, err)
		assert.Empty(t, dynakube.Status.ActiveGate.ConnectionInfoStatus)
		assert.Nil(t, meta.FindStatusCondition(dynakube.Status.Conditions, activeGateConnectionInfoConditionType))

		var actualSecret corev1.Secret
		err = fakeClient.Get(ctx, client.ObjectKey{Name: dynakube.ActivegateTenantSecret(), Namespace: testNamespace}, &actualSecret)
		require.Error(t, err)
		assert.True(t, k8serrors.IsNotFound(err))
	})

	t.Run(`store ActiveGate connection info to DynaKube status + create tenant secret`, func(t *testing.T) {
		dynakube := getTestDynakube()

		dtc := dtclientmock.NewClient(t)
		dtc.On("GetActiveGateConnectionInfo", mock.AnythingOfType("context.backgroundCtx")).Return(getTestActiveGateConnectionInfo(), nil)

		fakeClient := fake.NewClient(dynakube)
		r := NewReconciler(fakeClient, fakeClient, dtc, dynakube)
		err := r.Reconcile(ctx)
		require.NoError(t, err)

		condition := meta.FindStatusCondition(dynakube.Status.Conditions, activeGateConnectionInfoConditionType)
		require.NotNil(t, condition)
		assert.Equal(t, metav1.ConditionTrue, condition.Status)
		assert.Equal(t, conditions.SecretCreatedReason, condition.Reason)

		assert.Equal(t, testTenantUUID, dynakube.Status.ActiveGate.ConnectionInfoStatus.TenantUUID)
		assert.Equal(t, testTenantEndpoints, dynakube.Status.ActiveGate.ConnectionInfoStatus.Endpoints)

		var actualSecret corev1.Secret
		err = fakeClient.Get(ctx, client.ObjectKey{Name: dynakube.ActivegateTenantSecret(), Namespace: testNamespace}, &actualSecret)
		require.NoError(t, err)
		assert.Equal(t, []byte(testTenantToken), actualSecret.Data[connectioninfo.TenantTokenKey])
	})

	t.Run(`update ActiveGate connection info + update tenant secret`, func(t *testing.T) {
		dynakube := getTestDynakube()

		dtc := dtclientmock.NewClient(t)
		dtc.On("GetActiveGateConnectionInfo", mock.AnythingOfType("context.backgroundCtx")).Return(getTestActiveGateConnectionInfo(), nil)

		fakeClient := fake.NewClient(dynakube, buildActiveGateSecret(*dynakube, testTenantUUID))
		dynakube.Status.ActiveGate.ConnectionInfoStatus = dynatracev1beta1.ActiveGateConnectionInfoStatus{
			ConnectionInfoStatus: dynatracev1beta1.ConnectionInfoStatus{
				TenantUUID: testOutdated,
				Endpoints:  testOutdated,
			},
		}

		r := NewReconciler(fakeClient, fakeClient, dtc, dynakube)
		rec := r.(*reconciler)
		rec.timeProvider.Set(rec.timeProvider.Now().Add(time.Minute * 20))

		err := r.Reconcile(ctx)
		require.NoError(t, err)

		condition := meta.FindStatusCondition(dynakube.Status.Conditions, activeGateConnectionInfoConditionType)
		assert.Equal(t, metav1.ConditionTrue, condition.Status)
		assert.Equal(t, conditions.SecretCreatedReason, condition.Reason)

		assert.Equal(t, testTenantUUID, dynakube.Status.ActiveGate.ConnectionInfoStatus.TenantUUID)
		assert.Equal(t, testTenantEndpoints, dynakube.Status.ActiveGate.ConnectionInfoStatus.Endpoints)

		var actualSecret corev1.Secret
		err = fakeClient.Get(ctx, client.ObjectKey{Name: dynakube.ActivegateTenantSecret(), Namespace: testNamespace}, &actualSecret)
		require.NoError(t, err)
		assert.Equal(t, []byte(testTenantToken), actualSecret.Data[connectioninfo.TenantTokenKey])
	})
	t.Run(`update ActiveGate connection info if tenant secret is missing, ignore timestamp`, func(t *testing.T) {
		dynakube := getTestDynakube()

		dtc := dtclientmock.NewClient(t)
		dtc.On("GetActiveGateConnectionInfo", mock.AnythingOfType("context.backgroundCtx")).Return(getTestActiveGateConnectionInfo(), nil)

		fakeClient := fake.NewClient(dynakube)

		dynakube.Status.ActiveGate.ConnectionInfoStatus = dynatracev1beta1.ActiveGateConnectionInfoStatus{
			ConnectionInfoStatus: dynatracev1beta1.ConnectionInfoStatus{
				TenantUUID: testOutdated,
				Endpoints:  testOutdated,
			},
		}

		r := NewReconciler(fakeClient, fakeClient, dtc, dynakube)
		err := r.Reconcile(ctx)
		require.NoError(t, err)

		condition := meta.FindStatusCondition(dynakube.Status.Conditions, activeGateConnectionInfoConditionType)
		assert.Equal(t, metav1.ConditionTrue, condition.Status)
		assert.Equal(t, conditions.SecretCreatedReason, condition.Reason)

		assert.Equal(t, testTenantUUID, dynakube.Status.ActiveGate.ConnectionInfoStatus.TenantUUID)
		assert.Equal(t, testTenantEndpoints, dynakube.Status.ActiveGate.ConnectionInfoStatus.Endpoints)

		var actualSecret corev1.Secret
		err = fakeClient.Get(ctx, client.ObjectKey{Name: dynakube.ActivegateTenantSecret(), Namespace: testNamespace}, &actualSecret)
		require.NoError(t, err)
		assert.Equal(t, []byte(testTenantToken), actualSecret.Data[connectioninfo.TenantTokenKey])
	})
	t.Run(`ActiveGate connection info error shown in conditions`, func(t *testing.T) {
		dynakube := getTestDynakube()
		fakeClient := fake.NewClientWithInterceptors(interceptor.Funcs{
			Get: func(ctx context.Context, client client.WithWatch, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
				return fmt.Errorf("BOOM")
			},
		})

		dtc := dtclientmock.NewClient(t)
		dtc.On("GetActiveGateConnectionInfo", mock.AnythingOfType("context.backgroundCtx")).Return(getTestActiveGateConnectionInfo(), nil).Maybe()

		r := NewReconciler(fakeClient, fakeClient, dtc, dynakube)
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

func getTestDynakube() *dynatracev1beta1.DynaKube {
	return &dynatracev1beta1.DynaKube{
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
