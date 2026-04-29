package secret

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/proxy"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha2/edgeconnect"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/edgeconnect/config"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/edgeconnect/consts"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestPrepareConfigFile(t *testing.T) {
	const (
		testName      = "test-name-edgeconnect"
		testNamespace = "test-namespace"
		testToken     = "dummy-token"
		testProxyAuthRef               = "proxy-auth-ref"
		testCreatedID                  = "id"
		testCreatedOauthClientID       = "created-client-id"
		testCreatedOauthClientSecret   = "created-client-secret"
		testCreatedOauthClientResource = "created-client-resource"
	)

	testNewSecret := func(name, namespace string, kv map[string]string) *corev1.Secret {
		t.Helper()
		data := make(map[string][]byte)
		for k, v := range kv {
			data[k] = []byte(v)
		}

		return &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace}, Data: data}
	}

	testClientSecret := func(name string, namespace string) *corev1.Secret {
		t.Helper()

		return testNewSecret(name, namespace, map[string]string{
			consts.KeyEdgeConnectID:                testCreatedID,
			consts.KeyEdgeConnectOauthClientID:     testCreatedOauthClientID,
			consts.KeyEdgeConnectOauthClientSecret: testCreatedOauthClientSecret,
			consts.KeyEdgeConnectOauthResource:     testCreatedOauthClientResource,
		})
	}

	t.Run("Create basic config", func(t *testing.T) {
		const testSecretName = "test-secret"

		ec := &edgeconnect.EdgeConnect{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testName,
				Namespace: testNamespace,
			},
			Spec: edgeconnect.EdgeConnectSpec{
				APIServer: "abc12345.dynatrace.com",
				OAuth: edgeconnect.OAuthSpec{
					Endpoint:     "https://test.com/sso/oauth2/token",
					Resource:     "urn:dtenvironment:test12345",
					ClientSecret: testSecretName,
				},
			},
		}

		kubeReader := fake.NewClient(testClientSecret(testSecretName, testNamespace))
		cfg, err := PrepareConfigFile(t.Context(), ec, kubeReader, testToken)

		require.NoError(t, err)

		expected := `name: test-name-edgeconnect
api_endpoint_host: abc12345.dynatrace.com
oauth:
    endpoint: https://test.com/sso/oauth2/token
    client_id: created-client-id
    client_secret: created-client-secret
    resource: urn:dtenvironment:test12345
root_certificate_paths:
    - /var/run/secrets/kubernetes.io/serviceaccount/ca.crt
`
		assert.Equal(t, expected, string(cfg))
	})

	t.Run("Create full config", func(t *testing.T) {
		const testSecretName = "test-secret"

		ec := &edgeconnect.EdgeConnect{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testName,
				Namespace: testNamespace,
			},
			Spec: edgeconnect.EdgeConnectSpec{
				APIServer: "abc12345.dynatrace.com",
				OAuth: edgeconnect.OAuthSpec{
					Endpoint:     "https://test.com/sso/oauth2/token",
					Resource:     "urn:dtenvironment:test12345",
					ClientSecret: testSecretName,
				},
				CaCertsRef: "certs",
				Proxy: &proxy.Spec{
					Host:    "proxy.com",
					NoProxy: "*.internal.com",
					Port:    443,
					AuthRef: testProxyAuthRef,
				},
			},
		}

		authRef := testNewSecret(testProxyAuthRef, ec.Namespace, map[string]string{
			edgeconnect.ProxyAuthUserKey:     "user",
			edgeconnect.ProxyAuthPasswordKey: "pass",
		})
		kubeReader := fake.NewClient(testClientSecret(testSecretName, testNamespace), authRef)
		cfg, err := PrepareConfigFile(t.Context(), ec, kubeReader, testToken)

		require.NoError(t, err)

		expected := `name: test-name-edgeconnect
api_endpoint_host: abc12345.dynatrace.com
oauth:
    endpoint: https://test.com/sso/oauth2/token
    client_id: created-client-id
    client_secret: created-client-secret
    resource: urn:dtenvironment:test12345
root_certificate_paths:
    - /etc/edge_connect_certs/certificate.pem
    - /var/run/secrets/kubernetes.io/serviceaccount/ca.crt
proxy:
    auth:
        user: user
        password: pass
    server: proxy.com
    exceptions: '*.internal.com'
    port: 443
`
		assert.Equal(t, expected, string(cfg))
	})

	t.Run("Create config k8s automation enabled", func(t *testing.T) {
		const testSecretName = "test-secret"

		ec := &edgeconnect.EdgeConnect{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testName,
				Namespace: testNamespace,
			},
			Spec: edgeconnect.EdgeConnectSpec{
				APIServer: "abc12345.dynatrace.com",
				OAuth: edgeconnect.OAuthSpec{
					Endpoint:     "https://test.com/sso/oauth2/token",
					Resource:     "urn:dtenvironment:test12345",
					ClientSecret: testSecretName,
				},
				KubernetesAutomation: &edgeconnect.KubernetesAutomationSpec{
					Enabled: true,
				},
			},
		}

		kubeReader := fake.NewClient(testClientSecret(testSecretName, testNamespace))
		cfg, err := PrepareConfigFile(t.Context(), ec, kubeReader, testToken)

		require.NoError(t, err)

		expected := `name: test-name-edgeconnect
api_endpoint_host: abc12345.dynatrace.com
oauth:
    endpoint: https://test.com/sso/oauth2/token
    client_id: created-client-id
    client_secret: created-client-secret
    resource: urn:dtenvironment:test12345
root_certificate_paths:
    - /var/run/secrets/kubernetes.io/serviceaccount/ca.crt
secrets:
    - name: K8S_SERVICE_ACCOUNT_TOKEN
      token: dummy-token
      from_file: /var/run/secrets/kubernetes.io/serviceaccount/token
      restrict_hosts_to:
        - kubernetes.default.svc.cluster.local
`
		assert.Equal(t, expected, string(cfg))
	})
}

func Test_safeEdgeConnectCfg(t *testing.T) {
	t.Run("redacts client secret and token", func(t *testing.T) {
		cfg := config.EdgeConnect{
			Name:            "test",
			APIEndpointHost: "test",
			OAuth: config.OAuth{
				Endpoint:     "endpoint",
				ClientID:     "id",
				ClientSecret: "super secret",
				Resource:     "resource",
			},
			RestrictHostsTo:      []string{"host"},
			RootCertificatePaths: []string{"path"},
			Proxy: config.Proxy{
				Server:     "server",
				Exceptions: "exception",
				Port:       2,
				Auth: config.Auth{
					User:     "user",
					Password: "password",
				},
			},
			Secrets: []config.Secret{
				{
					Name:            "secret",
					Token:           "token",
					FromFile:        "file",
					RestrictHostsTo: []string{"hosts"},
				},
			},
		}
		expected := `name: test
api_endpoint_host: test
oauth:
    endpoint: endpoint
    client_id: id
    resource: resource
restrict_hosts_to:
    - host
root_certificate_paths:
    - path
proxy:
    user: user
    server: server
    exceptions: exception
    port: 2
secrets:
    - name: secret
      restrict_hosts_to:
        - hosts
`
		assert.Equal(t, expected, safeEdgeConnectCfg(cfg))
	})
}
