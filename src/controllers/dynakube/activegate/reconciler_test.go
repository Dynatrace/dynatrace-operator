package activegate

import (
	"context"
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/activegate/internal/secret"
	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
	"github.com/Dynatrace/dynatrace-operator/src/scheme"
	"github.com/Dynatrace/dynatrace-operator/src/scheme/fake"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

const (
	testName        = "test-name"
	testNamespace   = "test-namespace"
	testProxyName   = "test-proxy"
	testServiceName = testName + "-activegate"
)

var (
	testKubeSystemNamespace = &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "kube-system",
			UID:  "01234-5678-9012-3456",
		},
	}

	tenantInfoResponse = &dtclient.ActiveGateTenantInfo{
		TenantInfo: dtclient.TenantInfo{
			UUID:  "testUUID",
			Token: "dt.some.valuegoeshere",
		},
		Endpoints: "someEndpoints",
	}
)

func TestReconciler_Reconcile(t *testing.T) {
	dtc := &dtclient.MockDynatraceClient{}
	dtc.On("GetActiveGateTenantInfo").Return(tenantInfoResponse, nil)
	dtc.On("GetActiveGateAuthToken", testName).Return(&dtclient.ActiveGateAuthTokenInfo{}, nil)

	t.Run(`Reconcile works with minimal setup`, func(t *testing.T) {
		instance := &dynatracev1beta1.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: testNamespace,
				Name:      testName,
			}}
		fakeClient := fake.NewClient()
		r, err := NewReconciler(context.TODO(), fakeClient, fakeClient, scheme.Scheme, instance, dtc)
		assert.NoError(t, err)
		upd, err := r.Reconcile()
		assert.NoError(t, err)
		assert.True(t, upd)

		tenantSecret := &corev1.Secret{}
		err = fakeClient.Get(context.TODO(), types.NamespacedName{Name: instance.AGTenantSecret(), Namespace: testNamespace}, tenantSecret)
		assert.NoError(t, err)
		assert.Equal(t, tenantInfoResponse.Token, string(tenantSecret.Data[secret.TenantTokenName]))
		assert.Equal(t, tenantInfoResponse.UUID, string(tenantSecret.Data[secret.TenantUuidName]))
		assert.Equal(t, tenantInfoResponse.Endpoints, string(tenantSecret.Data[secret.CommunicationEndpointsName]))
	})
	t.Run(`Reconcile AG proxy secret`, func(t *testing.T) {
		instance := &dynatracev1beta1.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: testNamespace,
				Name:      testName,
			},
			Spec: dynatracev1beta1.DynaKubeSpec{
				Proxy: &dynatracev1beta1.DynaKubeProxy{Value: testProxyName},
			},
		}
		fakeClient := fake.NewClient()
		r, err := NewReconciler(context.TODO(), fakeClient, fakeClient, scheme.Scheme, instance, dtc)
		assert.NoError(t, err)
		upd, err := r.Reconcile()
		assert.NoError(t, err)
		assert.True(t, upd)

		var proxySecret corev1.Secret
		err = fakeClient.Get(context.TODO(), types.NamespacedName{Name: "dynatrace-activegate-internal-proxy", Namespace: testNamespace}, &proxySecret)
		assert.NoError(t, err)
	})
	t.Run(`Reconcile AG capability (creation and deletion)`, func(t *testing.T) {
		instance := &dynatracev1beta1.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: testNamespace,
				Name:      testName,
			},
			Spec: dynatracev1beta1.DynaKubeSpec{
				ActiveGate: dynatracev1beta1.ActiveGateSpec{
					Capabilities: []dynatracev1beta1.CapabilityDisplayName{dynatracev1beta1.RoutingCapability.DisplayName},
				},
			},
		}
		fakeClient := fake.NewClient(testKubeSystemNamespace)
		r, err := NewReconciler(context.TODO(), fakeClient, fakeClient, scheme.Scheme, instance, dtc)
		assert.NoError(t, err)
		upd, err := r.Reconcile()
		assert.NoError(t, err)
		assert.True(t, upd)

		var service corev1.Service
		err = fakeClient.Get(context.TODO(), types.NamespacedName{Name: testServiceName, Namespace: testNamespace}, &service)
		assert.NoError(t, err)

		// remove AG from spec
		instance.Spec.ActiveGate = dynatracev1beta1.ActiveGateSpec{}
		upd, err = r.Reconcile()
		assert.NoError(t, err)
		assert.True(t, upd)
		err = fakeClient.Get(context.TODO(), types.NamespacedName{Name: testServiceName, Namespace: testNamespace}, &service)
		assert.True(t, errors.IsNotFound(err))
	})
}
