package authtoken

import (
	"testing"
	"time"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
	"github.com/Dynatrace/dynatrace-operator/src/scheme"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
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
	testAgAuthTokenResponse = &dtclient.ActiveGateAuthTokenInfo{
		TokenId: "test",
		Token:   "dt.some.valuegoeshere",
	}
)

func newTestReconciler(client client.Client) *Reconciler {
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
	dtc.On("GetActiveGateAuthToken", mock.Anything).Return(testAgAuthTokenResponse, nil)

	r := NewReconciler(client, client, scheme.Scheme, instance, dtc)
	return r
}

func TestReconcile(t *testing.T) {
	t.Run(`reconcile auth token for first time`, func(t *testing.T) {
		r := newTestReconciler(fake.NewClientBuilder().Build())
		update, err := r.Reconcile()
		assert.True(t, update)
		assert.NoError(t, err)
	})
	t.Run(`reconcile outdated auth token`, func(t *testing.T) {
		clt := fake.NewClientBuilder().
			WithScheme(scheme.Scheme).
			WithObjects(&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:              testDynakubeName + dynatracev1beta1.AuthTokenSecretSuffix,
					Namespace:         testNamespace,
					CreationTimestamp: metav1.Time{Time: time.Now().Add(-AuthTokenRotationInterval).Add(-5 * time.Second)},
				},
			}).
			Build()

		r := newTestReconciler(clt)
		update, err := r.Reconcile()
		assert.True(t, update)
		assert.NoError(t, err)
	})
}
