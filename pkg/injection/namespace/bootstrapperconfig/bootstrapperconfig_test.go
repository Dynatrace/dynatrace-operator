package bootstrapperconfig

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook"
	dtclientmock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/clients/dynatrace"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	testPaasToken       = "test-paas-token"
	testAPIToken        = "test-api-token"
	testDataIngestToken = "test-ingest-token"

	testUUID                  = "test-uuid"
	testTenantToken           = "abcd"
	testCommunicationEndpoint = "https://tenant.dev.dynatracelabs.com:443"

	testHost = "test-host"

	testDynakube   = "test-dynakube"
	testNamespace  = "test-namespace"
	testNamespace2 = "test-namespace2"

	testNamespaceDynatrace = "dynatrace"

	testApiUrl = "https://" + testHost + "/e/" + testUUID + "/api"
)

func TestNewSecretGenerator(t *testing.T) {
	client := fake.NewClient()
	mockDTClient := dtclientmock.NewClient(t)

	secretGenerator := NewSecretGenerator(client, client, mockDTClient)
	assert.NotNil(t, secretGenerator)

	assert.Equal(t, client, secretGenerator.client)
	assert.Equal(t, client, secretGenerator.apiReader)
	assert.Equal(t, mockDTClient, secretGenerator.dtClient)
}

func TestGenerateForDynakube(t *testing.T) {
	t.Run("succcessfully generate secret for dynakube", func(t *testing.T) {
		dynakube := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testDynakube,
				Namespace: testNamespaceDynatrace,
			},
			Spec: dynakube.DynaKubeSpec{
				APIURL: testApiUrl,
			},
		}

		clt := fake.NewClientWithIndex(
			dynakube,
			clientInjectedNamespace(testNamespace, testDynakube),
			clientSecret(testDynakube, testNamespaceDynatrace, map[string][]byte{
				dtclient.ApiToken:  []byte(testAPIToken),
				dtclient.PaasToken: []byte(testPaasToken),
			}),
			clientSecret(dynakube.OneAgent().GetTenantSecret(), testNamespaceDynatrace, map[string][]byte{
				"tenant-token": []byte(testTenantToken),
			}),
			clientSecret(consts.BootstrapperInitSecretName, testNamespace, nil),
		)

		mockDTClient := dtclientmock.NewClient(t)

		mockDTClient.On("GetProcessModuleConfig", mock.AnythingOfType("context.backgroundCtx"), mock.AnythingOfType("uint")).Return(&dtclient.ProcessModuleConfig{}, nil)

		secretGenerator := NewSecretGenerator(clt, clt, mockDTClient)
		err := secretGenerator.GenerateForDynakube(context.Background(), dynakube)
		require.NoError(t, err)
	})
	t.Run("fail while generating secret for dynakube", func(t *testing.T) {
		dynakube := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testDynakube,
				Namespace: testNamespaceDynatrace,
			},
			Spec: dynakube.DynaKubeSpec{
				APIURL: testApiUrl,
			},
		}

		clt := fake.NewClientWithIndex(
			dynakube,
			clientInjectedNamespace(testNamespace, testDynakube),
			clientSecret(testDynakube, testNamespaceDynatrace, map[string][]byte{
				dtclient.ApiToken:  []byte(testAPIToken),
				dtclient.PaasToken: []byte(testPaasToken),
			}),
		)

		mockDTClient := dtclientmock.NewClient(t)

		mockDTClient.On("GetProcessModuleConfig", mock.AnythingOfType("context.backgroundCtx"), mock.AnythingOfType("uint")).Return(&dtclient.ProcessModuleConfig{}, nil)

		secretGenerator := NewSecretGenerator(clt, clt, mockDTClient)
		err := secretGenerator.GenerateForDynakube(context.Background(), dynakube)
		require.Error(t, err)
	})
}

func TestCleanup(t *testing.T) {
	dynakube := &dynakube.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testDynakube,
			Namespace: testNamespaceDynatrace,
		},
		Spec: dynakube.DynaKubeSpec{
			APIURL: testApiUrl,
		},
	}

	clt := fake.NewClientWithIndex(
		dynakube,
		clientInjectedNamespace(testNamespace, testDynakube),
		clientSecret(testDynakube, testNamespaceDynatrace, map[string][]byte{
			dtclient.ApiToken:  []byte(testAPIToken),
			dtclient.PaasToken: []byte(testPaasToken),
		}),
		clientSecret(dynakube.OneAgent().GetTenantSecret(), testNamespaceDynatrace, map[string][]byte{
			"tenant-token": []byte(testTenantToken),
		}),
		clientSecret(consts.BootstrapperInitSecretName, testNamespace, nil),
		clientSecret(consts.BootstrapperInitSecretName, testNamespace2, nil),
	)
	namespaces := []corev1.Namespace{
		{ObjectMeta: metav1.ObjectMeta{Name: testNamespace}},
		{ObjectMeta: metav1.ObjectMeta{Name: testNamespace2}},
	}

	var secretNS1 corev1.Secret
	err := clt.Get(context.Background(), client.ObjectKey{Name: consts.BootstrapperInitSecretName, Namespace: testNamespace}, &secretNS1)
	require.NoError(t, err)

	require.NotEmpty(t, secretNS1)
	assert.Equal(t, consts.BootstrapperInitSecretName, secretNS1.Name)

	var secretNS2 corev1.Secret
	err = clt.Get(context.Background(), client.ObjectKey{Name: consts.BootstrapperInitSecretName, Namespace: testNamespace}, &secretNS2)
	require.NoError(t, err)

	require.NotEmpty(t, secretNS2)
	assert.Equal(t, consts.BootstrapperInitSecretName, secretNS2.Name)

	err = Cleanup(context.Background(), clt, clt, namespaces)
	require.NoError(t, err)

	var deletedSecretNS1 corev1.Secret
	err = clt.Get(context.Background(), client.ObjectKey{Name: consts.BootstrapperInitSecretName, Namespace: testNamespace}, &deletedSecretNS1)
	require.Error(t, err)

	require.Empty(t, deletedSecretNS1)
	assert.NotEqual(t, consts.BootstrapperInitSecretName, deletedSecretNS1.Name)

	var deletedSecretNS2 corev1.Secret
	err = clt.Get(context.Background(), client.ObjectKey{Name: consts.BootstrapperInitSecretName, Namespace: testNamespace}, &deletedSecretNS2)
	require.Error(t, err)

	require.Empty(t, deletedSecretNS2)
	assert.NotEqual(t, consts.BootstrapperInitSecretName, deletedSecretNS2.Name)
}

func clientSecret(secretName string, namespaceName string, data map[string][]byte) *corev1.Secret {
	return &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "core/v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: namespaceName,
		},
		Data: data,
	}
}

func clientInjectedNamespace(namespaceName string, dynakubeName string) *corev1.Namespace {
	return &corev1.Namespace{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "corev1",
			Kind:       "Namespace",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: namespaceName,
			Labels: map[string]string{
				dtwebhook.InjectionInstanceLabel: dynakubeName,
			},
		},
	}
}
