package authtoken

import (
	"context"
	"testing"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/activegate"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8sconditions"
	dtclientmock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/clients/dynatrace"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	testDynakubeName = "test-dynakube"
	testNamespace    = "test-namespace"
	testToken        = "dt.testtoken.test"
)

var (
	testAgAuthTokenResponse = &dtclient.ActiveGateAuthTokenInfo{
		TokenID: "test",
		Token:   "dt.some.valuegoeshere",
	}

	anyCtx = mock.MatchedBy(func(context.Context) bool { return true })
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

func TestReconcile(t *testing.T) {
	t.Run("reconcile auth token for first time", func(t *testing.T) {
		dk := newDynaKube()

		clt := fake.NewClientBuilder().Build()

		dtc := dtclientmock.NewClient(t)
		dtc.EXPECT().GetActiveGateAuthToken(anyCtx, dk.Name).Return(testAgAuthTokenResponse, nil).Once()
		r := NewReconciler(clt, clt, dk, dtc)

		err := r.Reconcile(t.Context())
		require.NoError(t, err)

		authToken, err := r.secrets.Get(t.Context(), types.NamespacedName{
			Namespace: r.dk.Namespace,
			Name:      r.dk.ActiveGate().GetAuthTokenSecretName(),
		})
		require.NoError(t, err)

		assert.NotEmpty(t, authToken.Data[ActiveGateAuthTokenName])

		condition := meta.FindStatusCondition(*dk.Conditions(), ActiveGateAuthTokenSecretConditionType)
		assert.Equal(t, metav1.ConditionTrue, condition.Status)
		assert.Equal(t, k8sconditions.SecretCreatedReason, condition.Reason)
	})
	t.Run("reconcile outdated auth token", func(t *testing.T) {
		dk := newDynaKube()

		clt := fake.NewClientBuilder().Build()

		dtc := dtclientmock.NewClient(t)
		dtc.EXPECT().GetActiveGateAuthToken(anyCtx, dk.Name).Return(testAgAuthTokenResponse, nil).Twice()
		r := NewReconciler(clt, clt, dk, dtc)

		// create secret
		err := r.Reconcile(t.Context())
		require.NoError(t, err)

		condition := meta.FindStatusCondition(*dk.Conditions(), ActiveGateAuthTokenSecretConditionType)
		assert.Equal(t, metav1.ConditionTrue, condition.Status)
		assert.Equal(t, k8sconditions.SecretCreatedReason, condition.Reason)
		firstTransition := condition.LastTransitionTime

		authToken, err := r.secrets.Get(t.Context(), types.NamespacedName{
			Namespace: r.dk.Namespace,
			Name:      r.dk.ActiveGate().GetAuthTokenSecretName(),
		})
		require.NoError(t, err)
		assert.NotEmpty(t, authToken.Data[ActiveGateAuthTokenName])

		// "initialize" the secret as if it was created a month ago
		authToken.Data = map[string][]byte{ActiveGateAuthTokenName: []byte(testToken)}
		// time.Round is called because client.Update(secret)->json.Marshall(secret) rounds CreationTimestamp to seconds
		authToken.CreationTimestamp = metav1.Time{Time: time.Now().Round(1 * time.Second).Add(-AuthTokenRotationInterval).Add(-5 * time.Second)}
		err = r.secrets.Update(t.Context(), authToken)
		require.NoError(t, err)

		firstCreationTimestamp := authToken.CreationTimestamp

		// let's "wait", small difference needed to compare LastTransitionTime
		time.Sleep(1 * time.Second)

		// update secret
		err = r.Reconcile(t.Context())
		require.NoError(t, err)

		condition = meta.FindStatusCondition(*dk.Conditions(), ActiveGateAuthTokenSecretConditionType)
		assert.Equal(t, metav1.ConditionTrue, condition.Status)
		assert.Equal(t, k8sconditions.SecretCreatedReason, condition.Reason)
		secondTransition := condition.LastTransitionTime

		authToken, err = r.secrets.Get(t.Context(), types.NamespacedName{
			Namespace: r.dk.Namespace,
			Name:      r.dk.ActiveGate().GetAuthTokenSecretName(),
		})
		require.NoError(t, err)
		assert.NotEmpty(t, authToken.Data[ActiveGateAuthTokenName])
		secondCreationTimestamp := authToken.CreationTimestamp

		// token has been changed
		assert.NotEqual(t, authToken.Data[ActiveGateAuthTokenName], []byte(testToken))
		assert.NotEqual(t, firstCreationTimestamp, secondCreationTimestamp)
		assert.NotEqual(t, secondTransition, firstTransition)
	})
	t.Run("reconcile valid auth token", func(t *testing.T) {
		dk := newDynaKube()

		clt := fake.NewClientBuilder().Build()

		dtc := dtclientmock.NewClient(t)
		dtc.EXPECT().GetActiveGateAuthToken(anyCtx, dk.Name).Return(testAgAuthTokenResponse, nil).Once()
		r := NewReconciler(clt, clt, dk, dtc)

		// create secret
		err := r.Reconcile(t.Context())
		require.NoError(t, err)

		condition := meta.FindStatusCondition(*dk.Conditions(), ActiveGateAuthTokenSecretConditionType)
		assert.Equal(t, metav1.ConditionTrue, condition.Status)
		assert.Equal(t, k8sconditions.SecretCreatedReason, condition.Reason)
		firstTransition := condition.LastTransitionTime

		authToken, err := r.secrets.Get(t.Context(), types.NamespacedName{
			Namespace: r.dk.Namespace,
			Name:      r.dk.ActiveGate().GetAuthTokenSecretName(),
		})

		require.NoError(t, err)
		assert.NotEmpty(t, authToken.Data[ActiveGateAuthTokenName])

		// "initialize" the secret as if it was created a month ago
		authToken.Data = map[string][]byte{ActiveGateAuthTokenName: []byte(testToken)}
		// time.Round is called because client.Update(secret)->json.Marshall(secret) rounds CreationTimestamp to seconds
		authToken.CreationTimestamp = metav1.Time{Time: time.Now().Round(1 * time.Second).Add(-AuthTokenRotationInterval).Add(1 * time.Minute)}
		err = r.secrets.Update(t.Context(), authToken)
		require.NoError(t, err)

		firstCreationTimestamp := authToken.CreationTimestamp

		// let's "wait", small difference needed to compare LastTransitionTime
		time.Sleep(1 * time.Second)

		// do not update secret
		err = r.Reconcile(t.Context())
		require.NoError(t, err)

		condition = meta.FindStatusCondition(*dk.Conditions(), ActiveGateAuthTokenSecretConditionType)
		assert.Equal(t, metav1.ConditionTrue, condition.Status)
		assert.Equal(t, k8sconditions.SecretCreatedReason, condition.Reason)
		secondTransition := condition.LastTransitionTime

		authToken, err = r.secrets.Get(t.Context(), types.NamespacedName{
			Namespace: r.dk.Namespace,
			Name:      r.dk.ActiveGate().GetAuthTokenSecretName(),
		})
		require.NoError(t, err)
		assert.NotEmpty(t, authToken.Data[ActiveGateAuthTokenName])
		secondCreationTimestamp := authToken.CreationTimestamp

		// token hasn't been changed
		assert.Equal(t, authToken.Data[ActiveGateAuthTokenName], []byte(testToken))
		assert.Equal(t, firstCreationTimestamp, secondCreationTimestamp)
		assert.Equal(t, secondTransition, firstTransition)
	})
}
