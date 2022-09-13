package tenantinfo

import (
	"context"
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
	"github.com/Dynatrace/dynatrace-operator/src/scheme"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	testDynakubeName = "test-dynakube"
	testNamespace    = "test-namespace"
)

var (
	tenantInfoResponse = &dtclient.ActiveGateTenantInfo{
		TenantInfo: dtclient.TenantInfo{
			UUID:  "testUUID",
			Token: "dt.some.valuegoeshere",
		},
		Endpoints: "someEndpoints",
	}
)

func newTestReconcilerWithInstance(client client.Client) *Reconciler {
	instance := &dynatracev1beta1.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNamespace,
			Name:      testDynakubeName,
		},
		Spec: dynatracev1beta1.DynaKubeSpec{
			APIURL: "https://testing.dev.dynatracelabs.com/api",
		},
	}
	dtc := &dtclient.MockDynatraceClient{}
	dtc.On("GetActiveGateTenantInfo", mock.Anything).Return(tenantInfoResponse, nil)

	r := NewReconciler(client, client, scheme.Scheme, instance, dtc)
	return r
}

func TestReconcile(t *testing.T) {
	t.Run(`reconcile tenant info for first time`, func(t *testing.T) {
		r := newTestReconcilerWithInstance(fake.NewClientBuilder().Build())
		update, err := r.Reconcile()

		require.NoError(t, err)

		var tenantInfoSecret corev1.Secret
		_ = r.Client.Get(context.TODO(), client.ObjectKey{Name: extendWithAGSecretSuffix(r.dynakube.Name), Namespace: testNamespace}, &tenantInfoSecret)

		assert.Equal(t, []byte(tenantInfoResponse.UUID), tenantInfoSecret.Data[TenantUuidName])
		assert.Equal(t, []byte(tenantInfoResponse.Token), tenantInfoSecret.Data[TenantTokenName])
		assert.Equal(t, []byte(tenantInfoResponse.Endpoints), tenantInfoSecret.Data[CommunicationEndpointsName])
		assert.True(t, update)
	})

	t.Run(`reconcile tenant info changed`, func(t *testing.T) {
		r := newTestReconcilerWithInstance(fake.NewClientBuilder().Build())
		update, err := r.Reconcile()
		require.NoError(t, err)
		assert.True(t, update)

		var newTenantToken = "dt.someOtherToken"
		tenantInfoResponse.UUID = newTenantToken

		update, err = r.Reconcile()

		require.NoError(t, err)

		var tenantInfoSecret corev1.Secret
		_ = r.Client.Get(context.TODO(), client.ObjectKey{Name: extendWithAGSecretSuffix(r.dynakube.Name), Namespace: testNamespace}, &tenantInfoSecret)

		assert.Equal(t, []byte(tenantInfoResponse.UUID), tenantInfoSecret.Data[TenantUuidName])
		assert.Equal(t, []byte(tenantInfoResponse.Token), tenantInfoSecret.Data[TenantTokenName])
		assert.Equal(t, []byte(tenantInfoResponse.Endpoints), tenantInfoSecret.Data[CommunicationEndpointsName])
		assert.True(t, update)
	})

	t.Run(`reconcile tenant info returns error`, func(t *testing.T) {
		r := newTestReconcilerWithInstance(fake.NewClientBuilder().Build())
		var dtClient = &dtclient.MockDynatraceClient{}
		dtClient.On("GetActiveGateTenantInfo", mock.Anything).Return(&dtclient.ActiveGateTenantInfo{}, errors.New("error"))
		r.dtc = dtClient
		update, err := r.Reconcile()

		require.Error(t, err)
		assert.False(t, update)
	})
}
