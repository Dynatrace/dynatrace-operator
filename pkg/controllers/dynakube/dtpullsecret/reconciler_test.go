package dtpullsecret

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
	dtclient2 "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/token"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	testPaasToken = "test-paas-token"
)

func TestReconciler_Reconcile(t *testing.T) {
	t.Run(`Create works with minimal setup`, func(t *testing.T) {
		mockDTC := &dtclient2.MockDynatraceClient{}
		dynakube := &dynatracev1beta1.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: testNamespace,
				Name:      testName,
			},
			Spec: dynatracev1beta1.DynaKubeSpec{
				APIURL: testApiUrl,
			},
		}
		fakeClient := fake.NewClient()
		r := NewReconciler(fakeClient, fakeClient, scheme.Scheme, dynakube, token.Tokens{
			dtclient2.DynatraceApiToken: token.Token{Value: testValue},
		})

		mockDTC.
			On("GetOneAgentConnectionInfo").
			Return(dtclient2.OneAgentConnectionInfo{}, nil)

		err := r.Reconcile(context.Background())

		assert.NoError(t, err)

		var pullSecret corev1.Secret
		err = fakeClient.Get(context.Background(),
			client.ObjectKey{Name: testName + "-pull-secret", Namespace: testNamespace},
			&pullSecret)

		assert.NoError(t, err)
		assert.NotNil(t, pullSecret)
		assert.NotEmpty(t, pullSecret.Data)
		assert.Contains(t, pullSecret.Data, ".dockerconfigjson")
		assert.NotEmpty(t, pullSecret.Data[".dockerconfigjson"])
	})
	t.Run(`Create does not reconcile with custom pull secret`, func(t *testing.T) {
		dynakube := &dynatracev1beta1.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: testNamespace,
				Name:      testName,
			},
			Spec: dynatracev1beta1.DynaKubeSpec{
				CustomPullSecret: testValue,
			}}
		r := NewReconciler(nil, nil, nil, dynakube, nil)
		err := r.Reconcile(context.Background())

		assert.NoError(t, err)
	})
	t.Run(`Create creates correct docker config`, func(t *testing.T) {
		expectedJSON := `{"auths":{"test-api-url":{"username":"test-tenant","password":"test-value","auth":"dGVzdC10ZW5hbnQ6dGVzdC12YWx1ZQ=="}}}`
		dynakube := &dynatracev1beta1.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: testNamespace,
				Name:      testName,
			},
			Spec: dynatracev1beta1.DynaKubeSpec{
				APIURL: testApiUrl,
			},
		}
		fakeClient := fake.NewClient()
		r := NewReconciler(fakeClient, fakeClient, scheme.Scheme, dynakube, token.Tokens{
			dtclient2.DynatraceApiToken: token.Token{Value: testValue},
		})

		err := r.Reconcile(context.Background())

		assert.NoError(t, err)

		var pullSecret corev1.Secret
		err = fakeClient.Get(context.Background(),
			client.ObjectKey{Name: testName + "-pull-secret", Namespace: testNamespace},
			&pullSecret)

		assert.NoError(t, err)
		assert.NotNil(t, pullSecret)
		assert.NotEmpty(t, pullSecret.Data)
		assert.Contains(t, pullSecret.Data, ".dockerconfigjson")
		assert.NotEmpty(t, pullSecret.Data[".dockerconfigjson"])
		assert.Equal(t, expectedJSON, string(pullSecret.Data[".dockerconfigjson"]))
	})
	t.Run(`Create update secret if data changed`, func(t *testing.T) {
		expectedJSON := `{"auths":{"test-api-url":{"username":"test-tenant","password":"test-value","auth":"dGVzdC10ZW5hbnQ6dGVzdC12YWx1ZQ=="}}}`
		dynakube := &dynatracev1beta1.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: testNamespace,
				Name:      testName,
			},
			Spec: dynatracev1beta1.DynaKubeSpec{
				APIURL: testApiUrl,
			},
		}
		fakeClient := fake.NewClient()
		r := NewReconciler(fakeClient, fakeClient, scheme.Scheme, dynakube, token.Tokens{
			dtclient2.DynatraceApiToken: token.Token{Value: testValue},
		})

		err := r.Reconcile(context.Background())

		assert.NoError(t, err)

		var pullSecret corev1.Secret
		err = fakeClient.Get(context.Background(),
			client.ObjectKey{Name: testName + "-pull-secret", Namespace: testNamespace},
			&pullSecret)

		assert.NoError(t, err)

		pullSecret.Data = nil
		err = fakeClient.Update(context.Background(), &pullSecret)

		assert.NoError(t, err)

		err = r.Reconcile(context.Background())

		assert.NoError(t, err)

		err = fakeClient.Get(context.Background(),
			client.ObjectKey{Name: testName + "-pull-secret", Namespace: testNamespace},
			&pullSecret)

		assert.NoError(t, err)
		assert.NotNil(t, pullSecret)
		assert.NotEmpty(t, pullSecret.Data)
		assert.Contains(t, pullSecret.Data, ".dockerconfigjson")
		assert.NotEmpty(t, pullSecret.Data[".dockerconfigjson"])
		assert.Equal(t, expectedJSON, string(pullSecret.Data[".dockerconfigjson"]))
	})
}
