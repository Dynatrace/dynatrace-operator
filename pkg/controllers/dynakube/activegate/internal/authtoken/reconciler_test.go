package authtoken

import (
	"context"
	"testing"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme"
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	dtclientmock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/clients/dynatrace"
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
	secretName       = testDynakubeName + dynatracev1beta1.AuthTokenSecretSuffix
	testToken        = "dt.testtoken.test"
)

var (
	testAgAuthTokenResponse = &dtclient.ActiveGateAuthTokenInfo{
		TokenId: "test",
		Token:   "dt.some.valuegoeshere",
	}
)

func newTestReconcilerWithInstance(t *testing.T, client client.Client) *Reconciler {
	instance := &dynatracev1beta1.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNamespace,
			Name:      testDynakubeName,
		},
		Spec: dynatracev1beta1.DynaKubeSpec{
			APIURL: "https://testing.dev.dynatracelabs.com/api",
			ActiveGate: dynatracev1beta1.ActiveGateSpec{
				Capabilities: []dynatracev1beta1.CapabilityDisplayName{
					dynatracev1beta1.RoutingCapability.DisplayName,
				},
			},
		},
	}
	dtc := dtclientmock.NewClient(t)
	dtc.On("GetActiveGateAuthToken", mock.AnythingOfType("context.backgroundCtx"), mock.Anything).Return(testAgAuthTokenResponse, nil).Maybe()

	r := NewReconciler(client, client, instance, dtc)

	return r
}

func TestReconcile(t *testing.T) {
	t.Run(`reconcile auth token for first time`, func(t *testing.T) {
		r := newTestReconcilerWithInstance(t, fake.NewClientBuilder().Build())
		err := r.Reconcile(context.Background())
		require.NoError(t, err)

		var authToken corev1.Secret
		_ = r.client.Get(context.Background(), client.ObjectKey{Name: r.dynakube.ActiveGateAuthTokenSecret(), Namespace: testNamespace}, &authToken)

		assert.NotEmpty(t, authToken.Data[ActiveGateAuthTokenName])
	})
	t.Run(`reconcile outdated auth token`, func(t *testing.T) {
		clt := fake.NewClientBuilder().
			WithScheme(scheme.Scheme).
			WithObjects(&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:              secretName,
					Namespace:         testNamespace,
					CreationTimestamp: metav1.Time{Time: time.Now().Add(-AuthTokenRotationInterval).Add(-5 * time.Second)},
				},
				Data: map[string][]byte{ActiveGateAuthTokenName: []byte(testToken)},
			}).
			Build()

		r := newTestReconcilerWithInstance(t, clt)
		err := r.Reconcile(context.Background())
		require.NoError(t, err)

		var authToken corev1.Secret
		_ = r.client.Get(context.Background(), client.ObjectKey{Name: r.dynakube.ActiveGateAuthTokenSecret(), Namespace: testNamespace}, &authToken)

		assert.NotEqual(t, authToken.Data[ActiveGateAuthTokenName], []byte(testToken))
	})
	t.Run(`reconcile valid auth token`, func(t *testing.T) {
		clt := fake.NewClientBuilder().
			WithScheme(scheme.Scheme).
			WithObjects(&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:              secretName,
					Namespace:         testNamespace,
					CreationTimestamp: metav1.Time{Time: time.Now().Add(-AuthTokenRotationInterval).Add(1 * time.Minute)},
				},
				Data: map[string][]byte{ActiveGateAuthTokenName: []byte(testToken)},
			}).
			Build()
		r := newTestReconcilerWithInstance(t, clt)

		err := r.Reconcile(context.Background())
		require.NoError(t, err)

		var authToken corev1.Secret
		_ = r.client.Get(context.Background(), client.ObjectKey{Name: r.dynakube.ActiveGateAuthTokenSecret(), Namespace: testNamespace}, &authToken)

		assert.Equal(t, authToken.Data[ActiveGateAuthTokenName], []byte(testToken))
	})
}
