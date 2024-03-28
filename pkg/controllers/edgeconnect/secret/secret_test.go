package secret

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	edgeconnectv1alpha1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha1/edgeconnect"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/edgeconnect/consts"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	testName                       = "test-name-edgeconnectv1alpha1"
	testNamespace                  = "test-namespace"
	testOauthClientId              = "client-id"
	testOauthClientSecret          = "client-secret"
	testOauthClientResource        = "client-resource"
	testCreatedOauthClientId       = "created-client-id"
	testCreatedOauthClientSecret   = "created-client-secret"
	testCreatedOauthClientResource = "created-client-resource"
	testCreatedId                  = "id"
)

func Test_prepareEdgeConnectConfigFile(t *testing.T) {
	t.Run("Create basic config", func(t *testing.T) {
		testEdgeConnect := &edgeconnectv1alpha1.EdgeConnect{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testName,
				Namespace: testNamespace,
			},
			Spec: edgeconnectv1alpha1.EdgeConnectSpec{
				ApiServer: "abc12345.dynatrace.com",
				OAuth: edgeconnectv1alpha1.OAuthSpec{
					Endpoint:     "https://sso-dev.dynatracelabs.com/sso/oauth2/token",
					Resource:     "urn:dtenvironment:test12345",
					ClientSecret: "test-secret",
				},
			},
		}

		testSecretName := "test-secret"
		kubeReader := fake.NewClient(createClientSecret(testSecretName, testNamespace))
		cfg, err := PrepareConfigFile(testEdgeConnect, kubeReader)

		require.NoError(t, err)

		expected := `name: test-name-edgeconnectv1alpha1
api_endpoint_host: abc12345.dynatrace.com
oauth:
    endpoint: https://sso-dev.dynatracelabs.com/sso/oauth2/token
    client_id: created-client-id
    client_secret: created-client-secret
    resource: urn:dtenvironment:test12345
`
		assert.Equal(t, expected, string(cfg))
	})

	t.Run("Create full config", func(t *testing.T) {
		testEdgeConnect := &edgeconnectv1alpha1.EdgeConnect{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testName,
				Namespace: testNamespace,
			},
			Spec: edgeconnectv1alpha1.EdgeConnectSpec{
				ApiServer: "abc12345.dynatrace.com",
				OAuth: edgeconnectv1alpha1.OAuthSpec{
					Endpoint:     "https://sso-dev.dynatracelabs.com/sso/oauth2/token",
					Resource:     "urn:dtenvironment:test12345",
					ClientSecret: "test-secret",
				},
				CaCertsRef: "certs",
				Proxy: &edgeconnectv1alpha1.ProxySpec{
					Host:    "proxy.com",
					NoProxy: "*.internal.com",
					Port:    443,
				},
			},
		}
		testSecretName := "test-secret"
		kubeReader := fake.NewClient(createClientSecret(testSecretName, testNamespace))
		cfg, err := PrepareConfigFile(testEdgeConnect, kubeReader)

		require.NoError(t, err)

		expected := `name: test-name-edgeconnectv1alpha1
api_endpoint_host: abc12345.dynatrace.com
oauth:
    endpoint: https://sso-dev.dynatracelabs.com/sso/oauth2/token
    client_id: created-client-id
    client_secret: created-client-secret
    resource: urn:dtenvironment:test12345
root_certificate_paths:
    - /etc/ssl/certificate.cer
proxy:
    server: proxy.com
    exceptions: '*.internal.com'
    port: 443
`
		assert.Equal(t, expected, string(cfg))
	})
}

func createClientSecret(name string, namespace string) *corev1.Secret {
	return newSecret(name, namespace, map[string]string{
		consts.KeyEdgeConnectId:                testCreatedId,
		consts.KeyEdgeConnectOauthClientID:     testCreatedOauthClientId,
		consts.KeyEdgeConnectOauthClientSecret: testCreatedOauthClientSecret,
		consts.KeyEdgeConnectOauthResource:     testCreatedOauthClientResource,
	})
}

func newSecret(name, namespace string, kv map[string]string) *corev1.Secret {
	data := make(map[string][]byte)
	for k, v := range kv {
		data[k] = []byte(v)
	}

	return &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace}, Data: data}
}
