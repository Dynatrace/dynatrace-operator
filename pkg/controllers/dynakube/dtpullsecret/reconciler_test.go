package dtpullsecret

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/token"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	testPaasToken = "test-paas-token"
)

type errorClient struct {
	client.Client
}

func (clt errorClient) Get(_ context.Context, _ client.ObjectKey, _ client.Object, _ ...client.GetOption) error {
	return errors.New("fake error")
}

func (clt errorClient) Create(_ context.Context, _ client.Object, _ ...client.CreateOption) error {
	return errors.New("fake error")
}

func TestReconciler_Reconcile(t *testing.T) {
	t.Run(`Create works with minimal setup`, func(t *testing.T) {
		dk := createTestDynakube()
		fakeClient := fake.NewClient()
		r := NewReconciler(fakeClient, fakeClient, dk, token.Tokens{
			dtclient.ApiToken: &token.Token{Value: testValue},
		})

		err := r.Reconcile(context.Background())

		require.NoError(t, err)

		var pullSecret corev1.Secret
		err = fakeClient.Get(context.Background(),
			client.ObjectKey{Name: testName + "-pull-secret", Namespace: testNamespace},
			&pullSecret)

		require.NoError(t, err)
		assert.NotNil(t, pullSecret)
		assert.NotEmpty(t, pullSecret.Data)
		assert.Contains(t, pullSecret.Data, ".dockerconfigjson")
		assert.NotEmpty(t, pullSecret.Data[".dockerconfigjson"])
	})
	t.Run(`Error when accessing K8s API`, func(t *testing.T) {
		dk := createTestDynakube()
		fakeClient := errorClient{}
		r := NewReconciler(fakeClient, fakeClient, dk, token.Tokens{
			dtclient.ApiToken: &token.Token{Value: testValue},
		})

		err := r.Reconcile(context.Background())
		require.Error(t, err)
	})
	t.Run(`Error when tenant UUID is missing`, func(t *testing.T) {
		dk := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: testNamespace,
				Name:      testName,
			},
			Spec: dynakube.DynaKubeSpec{
				OneAgent: dynakube.OneAgentSpec{CloudNativeFullStack: &dynakube.CloudNativeFullStackSpec{}},
			},
		}
		fakeClient := errorClient{}
		r := NewReconciler(fakeClient, fakeClient, dk, token.Tokens{
			dtclient.ApiToken: &token.Token{Value: testValue},
		})

		err := r.Reconcile(context.Background())
		require.Error(t, err)
	})
	t.Run(`Error when creating secret`, func(t *testing.T) {
		dk := createTestDynakube()
		fakeErrorClient := errorClient{}
		fakeClient := fake.NewClient()
		r := NewReconciler(fakeErrorClient, fakeClient, dk, token.Tokens{
			dtclient.ApiToken: &token.Token{Value: testValue},
		})

		err := r.Reconcile(context.Background())
		require.Error(t, err)
		assert.Equal(t, "failed to create or update secret: failed to create secret test-name-pull-secret: fake error", err.Error())
	})
	t.Run(`Create does not reconcile with custom pull secret`, func(t *testing.T) {
		dk := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: testNamespace,
				Name:      testName,
			},
			Spec: dynakube.DynaKubeSpec{
				CustomPullSecret: testValue,
			}}
		r := NewReconciler(nil, nil, dk, nil)
		err := r.Reconcile(context.Background())

		require.NoError(t, err)
	})
	t.Run(`Create creates correct docker config`, func(t *testing.T) {
		expectedJSON := `{"auths":{"test-api-url":{"username":"test-tenant","password":"test-value","auth":"dGVzdC10ZW5hbnQ6dGVzdC12YWx1ZQ=="}}}`
		dk := createTestDynakube()
		fakeClient := fake.NewClient()
		r := NewReconciler(fakeClient, fakeClient, dk, token.Tokens{
			dtclient.ApiToken: &token.Token{Value: testValue},
		})

		err := r.Reconcile(context.Background())

		require.NoError(t, err)

		var pullSecret corev1.Secret
		err = fakeClient.Get(context.Background(),
			client.ObjectKey{Name: testName + "-pull-secret", Namespace: testNamespace},
			&pullSecret)

		require.NoError(t, err)
		assert.NotNil(t, pullSecret)
		assert.NotEmpty(t, pullSecret.Data)
		assert.Contains(t, pullSecret.Data, ".dockerconfigjson")
		assert.NotEmpty(t, pullSecret.Data[".dockerconfigjson"])
		assert.Equal(t, expectedJSON, string(pullSecret.Data[".dockerconfigjson"]))
	})
	t.Run(`Create update secret if data changed`, func(t *testing.T) {
		expectedJSON := `{"auths":{"test-api-url":{"username":"test-tenant","password":"test-value","auth":"dGVzdC10ZW5hbnQ6dGVzdC12YWx1ZQ=="}}}`
		dk := createTestDynakube()
		fakeClient := fake.NewClient()
		r := NewReconciler(fakeClient, fakeClient, dk, token.Tokens{
			dtclient.ApiToken: &token.Token{Value: testValue},
		})

		err := r.Reconcile(context.Background())

		require.NoError(t, err)

		var pullSecret corev1.Secret
		err = fakeClient.Get(context.Background(),
			client.ObjectKey{Name: testName + "-pull-secret", Namespace: testNamespace},
			&pullSecret)

		require.NoError(t, err)

		pullSecret.Data = nil
		err = fakeClient.Update(context.Background(), &pullSecret)

		require.NoError(t, err)

		r.timeprovider.Set(r.timeprovider.Now().Add(1 * time.Hour))
		err = r.Reconcile(context.Background())

		require.NoError(t, err)

		err = fakeClient.Get(context.Background(),
			client.ObjectKey{Name: testName + "-pull-secret", Namespace: testNamespace},
			&pullSecret)

		require.NoError(t, err)
		assert.NotNil(t, pullSecret)
		assert.NotEmpty(t, pullSecret.Data)
		assert.Contains(t, pullSecret.Data, ".dockerconfigjson")
		assert.NotEmpty(t, pullSecret.Data[".dockerconfigjson"])
		assert.Equal(t, expectedJSON, string(pullSecret.Data[".dockerconfigjson"]))
	})
	t.Run(`Reconciliation only runs every 15 min`, func(t *testing.T) {
		dk := createTestDynakube()
		dk.Spec.DynatraceApiRequestThreshold = dynakube.DefaultMinRequestThresholdMinutes
		fakeClient := fake.NewClient()
		r := NewReconciler(fakeClient, fakeClient, dk, token.Tokens{
			dtclient.ApiToken: &token.Token{Value: testValue},
		})

		err := r.Reconcile(context.Background())

		require.NoError(t, err)

		var pullSecret corev1.Secret
		err = fakeClient.Get(context.Background(),
			client.ObjectKey{Name: testName + "-pull-secret", Namespace: testNamespace},
			&pullSecret)

		require.NoError(t, err)

		pullSecret.Data = nil
		err = fakeClient.Update(context.Background(), &pullSecret)

		require.NoError(t, err)

		err = r.Reconcile(context.Background())

		require.NoError(t, err)

		err = fakeClient.Get(context.Background(),
			client.ObjectKey{Name: testName + "-pull-secret", Namespace: testNamespace},
			&pullSecret)

		require.NoError(t, err)
		assert.NotNil(t, pullSecret)
		assert.Empty(t, pullSecret.Data)
	})
	t.Run(`Cleanup works`, func(t *testing.T) {
		dk := createTestDynakube()
		fakeClient := fake.NewClient()
		r := NewReconciler(fakeClient, fakeClient, dk, token.Tokens{
			dtclient.ApiToken: &token.Token{Value: testValue},
		})

		err := r.Reconcile(context.Background())

		require.NoError(t, err)
		assert.NotEmpty(t, meta.FindStatusCondition(*dk.Conditions(), PullSecretConditionType))

		var pullSecret corev1.Secret
		err = fakeClient.Get(context.Background(),
			client.ObjectKey{Name: testName + "-pull-secret", Namespace: testNamespace},
			&pullSecret)

		require.NoError(t, err)

		dk.Spec.OneAgent = dynakube.OneAgentSpec{}
		err = r.Reconcile(context.Background())
		require.NoError(t, err)

		err = fakeClient.Get(context.Background(),
			client.ObjectKey{Name: testName + "-pull-secret", Namespace: testNamespace},
			&pullSecret)

		assert.True(t, k8serrors.IsNotFound(err))
		assert.Empty(t, meta.FindStatusCondition(*dk.Conditions(), PullSecretConditionType))
	})
}

func createTestDynakube() *dynakube.DynaKube {
	return addFakeTennantUUID(&dynakube.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNamespace,
			Name:      testName,
		},
		Spec: dynakube.DynaKubeSpec{
			APIURL:   testApiUrl,
			OneAgent: dynakube.OneAgentSpec{CloudNativeFullStack: &dynakube.CloudNativeFullStackSpec{}},
		},
	})
}

func addFakeTennantUUID(dk *dynakube.DynaKube) *dynakube.DynaKube {
	dk.Status.OneAgent.ConnectionInfoStatus.ConnectionInfo.TenantUUID = testTenant

	return dk
}
