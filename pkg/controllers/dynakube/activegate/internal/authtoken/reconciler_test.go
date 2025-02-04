package authtoken

import (
	"context"
	"testing"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube/activegate"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/conditions"
	dtclientmock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/clients/dynatrace"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

const (
	testDynakubeName = "test-dynakube"
	testNamespace    = "test-namespace"
	secretName       = testDynakubeName + activegate.AuthTokenSecretSuffix
	testToken        = "dt.testtoken.test"
)

var (
	testAgAuthTokenResponse = &dtclient.ActiveGateAuthTokenInfo{
		TokenId: "test",
		Token:   "dt.some.valuegoeshere",
	}
)

func newDynaKube() *dynakube.DynaKube {
	return &dynakube.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNamespace,
			Name:      testDynakubeName,
		},
		Spec: dynakube.DynaKubeSpec{
			APIURL: "https://testing.dev.dynatracelabs.com/api",
			ActiveGate: activegate.Spec{
				Capabilities: []activegate.CapabilityDisplayName{
					activegate.RoutingCapability.DisplayName,
				},
			},
		},
	}
}

func newTestReconciler(t *testing.T, client client.Client, dk *dynakube.DynaKube) *Reconciler {
	dtc := dtclientmock.NewClient(t)
	dtc.On("GetActiveGateAuthToken", mock.AnythingOfType("context.backgroundCtx"), mock.Anything).Return(testAgAuthTokenResponse, nil)

	r := NewReconciler(client, client, dk, dtc)

	return r
}

func clientCreateWithTimestamp() func(ctx context.Context, client client.WithWatch, obj client.Object, opts ...client.CreateOption) error {
	return func(ctx context.Context, client client.WithWatch, obj client.Object, opts ...client.CreateOption) error {
		obj.SetCreationTimestamp(metav1.Time{Time: time.Now()})

		return client.Create(ctx, obj, opts...)
	}
}

func TestReconcile(t *testing.T) {
	interceptorFuncs := interceptor.Funcs{
		Create: clientCreateWithTimestamp(),
	}

	t.Run(`reconcile auth token for first time`, func(t *testing.T) {
		dk := newDynaKube()

		clt := fake.NewClientBuilder().Build()

		r := newTestReconciler(t, clt, dk)
		err := r.Reconcile(context.Background())
		require.NoError(t, err)

		var authToken corev1.Secret
		_ = r.client.Get(context.Background(), client.ObjectKey{Name: r.dk.ActiveGate().GetAuthTokenSecretName(), Namespace: testNamespace}, &authToken)

		assert.NotEmpty(t, authToken.Data[ActiveGateAuthTokenName])

		condition := meta.FindStatusCondition(*dk.Conditions(), ActiveGateAuthTokenSecretConditionType)
		assert.Equal(t, metav1.ConditionTrue, condition.Status)
		assert.Equal(t, conditions.SecretCreatedReason, condition.Reason)
	})
	t.Run(`reconcile outdated auth token`, func(t *testing.T) {
		dk := newDynaKube()

		clt := interceptor.NewClient(fake.NewClientBuilder().Build(), interceptorFuncs)

		r := newTestReconciler(t, clt, dk)

		// create secret
		err := r.Reconcile(context.Background())
		require.NoError(t, err)

		condition := meta.FindStatusCondition(*dk.Conditions(), ActiveGateAuthTokenSecretConditionType)
		assert.Equal(t, metav1.ConditionTrue, condition.Status)
		assert.Equal(t, conditions.SecretCreatedReason, condition.Reason)
		firstTransition := condition.LastTransitionTime

		var authToken corev1.Secret
		err = r.client.Get(context.Background(), client.ObjectKey{Name: r.dk.ActiveGate().GetAuthTokenSecretName(), Namespace: testNamespace}, &authToken)
		require.NoError(t, err)
		assert.NotEmpty(t, authToken.Data[ActiveGateAuthTokenName])

		// "initialize" the secret as if it was created a month ago
		authToken.Data = map[string][]byte{ActiveGateAuthTokenName: []byte(testToken)}
		// time.Round is called because client.Update(secret)->json.Marshall(secret) rounds CreationTimestamp to seconds
		authToken.CreationTimestamp = metav1.Time{Time: time.Now().Round(1 * time.Second).Add(-AuthTokenRotationInterval).Add(-5 * time.Second)}
		err = r.client.Update(context.Background(), &authToken)
		require.NoError(t, err)

		firstCreationTimestamp := authToken.CreationTimestamp

		// let's "wait", small difference needed to compare LastTransitionTime
		time.Sleep(1 * time.Second)

		// update secret
		err = r.Reconcile(context.Background())
		require.NoError(t, err)

		condition = meta.FindStatusCondition(*dk.Conditions(), ActiveGateAuthTokenSecretConditionType)
		assert.Equal(t, metav1.ConditionTrue, condition.Status)
		assert.Equal(t, conditions.SecretCreatedReason, condition.Reason)
		secondTransition := condition.LastTransitionTime

		err = r.client.Get(context.Background(), client.ObjectKey{Name: r.dk.ActiveGate().GetAuthTokenSecretName(), Namespace: testNamespace}, &authToken)
		require.NoError(t, err)
		assert.NotEmpty(t, authToken.Data[ActiveGateAuthTokenName])
		secondCreationTimestamp := authToken.CreationTimestamp

		// token has been changed
		assert.NotEqual(t, authToken.Data[ActiveGateAuthTokenName], []byte(testToken))
		assert.NotEqual(t, firstCreationTimestamp, secondCreationTimestamp)
		assert.NotEqual(t, secondTransition, firstTransition)
	})
	t.Run(`reconcile valid auth token`, func(t *testing.T) {
		dk := newDynaKube()

		clt := interceptor.NewClient(fake.NewClientBuilder().Build(), interceptorFuncs)

		r := newTestReconciler(t, clt, dk)

		// create secret
		err := r.Reconcile(context.Background())
		require.NoError(t, err)

		condition := meta.FindStatusCondition(*dk.Conditions(), ActiveGateAuthTokenSecretConditionType)
		assert.Equal(t, metav1.ConditionTrue, condition.Status)
		assert.Equal(t, conditions.SecretCreatedReason, condition.Reason)
		firstTransition := condition.LastTransitionTime

		var authToken corev1.Secret
		_ = r.client.Get(context.Background(), client.ObjectKey{Name: r.dk.ActiveGate().GetAuthTokenSecretName(), Namespace: testNamespace}, &authToken)

		require.NoError(t, err)
		assert.NotEmpty(t, authToken.Data[ActiveGateAuthTokenName])

		// "initialize" the secret as if it was created a month ago
		authToken.Data = map[string][]byte{ActiveGateAuthTokenName: []byte(testToken)}
		// time.Round is called because client.Update(secret)->json.Marshall(secret) rounds CreationTimestamp to seconds
		authToken.CreationTimestamp = metav1.Time{Time: time.Now().Round(1 * time.Second).Add(-AuthTokenRotationInterval).Add(1 * time.Minute)}
		err = r.client.Update(context.Background(), &authToken)
		require.NoError(t, err)

		firstCreationTimestamp := authToken.CreationTimestamp

		// let's "wait", small difference needed to compare LastTransitionTime
		time.Sleep(1 * time.Second)

		// do not update secret
		err = r.Reconcile(context.Background())
		require.NoError(t, err)

		condition = meta.FindStatusCondition(*dk.Conditions(), ActiveGateAuthTokenSecretConditionType)
		assert.Equal(t, metav1.ConditionTrue, condition.Status)
		assert.Equal(t, conditions.SecretCreatedReason, condition.Reason)
		secondTransition := condition.LastTransitionTime

		err = r.client.Get(context.Background(), client.ObjectKey{Name: r.dk.ActiveGate().GetAuthTokenSecretName(), Namespace: testNamespace}, &authToken)
		require.NoError(t, err)
		assert.NotEmpty(t, authToken.Data[ActiveGateAuthTokenName])
		secondCreationTimestamp := authToken.CreationTimestamp

		// token hasn't been changed
		assert.Equal(t, authToken.Data[ActiveGateAuthTokenName], []byte(testToken))
		assert.Equal(t, firstCreationTimestamp, secondCreationTimestamp)
		assert.Equal(t, secondTransition, firstTransition)
	})
}
