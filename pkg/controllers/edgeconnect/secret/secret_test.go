package secret

import (
	"context"
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

const (
	testName                       = "test-name-edgeconnect"
	testNamespace                  = "test-namespace"
	testOauthClientId              = "client-id"
	testOauthClientSecret          = "client-secret"
	testOauthClientResource        = "client-resource"
	testToken                      = "dummy-token"
	testCreatedOauthClientId       = "created-client-id"
	testCreatedOauthClientSecret   = "created-client-secret"
	testCreatedOauthClientResource = "created-client-resource"
	testCreatedId                  = "id"
	testProxyAuthRef               = "proxy-auth-ref"
)

func Test_prepareEdgeConnectConfigFile(t *testing.T) {
	t.Run("Create basic config", func(t *testing.T) {
		ec := &edgeconnect.EdgeConnect{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testName,
				Namespace: testNamespace,
			},
			Spec: edgeconnect.EdgeConnectSpec{
				ApiServer: "abc12345.dynatrace.com",
				OAuth: edgeconnect.OAuthSpec{
					Endpoint:     "https://test.com/sso/oauth2/token",
					Resource:     "urn:dtenvironment:test12345",
					ClientSecret: "test-secret",
				},
			},
		}

		testSecretName := "test-secret"
		kubeReader := fake.NewClient(createClientSecret(testSecretName, testNamespace))
		cfg, err := PrepareConfigFile(context.Background(), ec, kubeReader, testToken)

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
		ec := &edgeconnect.EdgeConnect{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testName,
				Namespace: testNamespace,
			},
			Spec: edgeconnect.EdgeConnectSpec{
				ApiServer: "abc12345.dynatrace.com",
				OAuth: edgeconnect.OAuthSpec{
					Endpoint:     "https://test.com/sso/oauth2/token",
					Resource:     "urn:dtenvironment:test12345",
					ClientSecret: "test-secret",
				},
				CaCertsRef:         "certs",
				ServiceAccountName: "test",
				Proxy: &proxy.Spec{
					Host:    "proxy.com",
					NoProxy: "*.internal.com",
					Port:    443,
					AuthRef: testProxyAuthRef,
				},
			},
		}
		testSecretName := "test-secret"
		authRef := newSecret(testProxyAuthRef, ec.Namespace, map[string]string{
			edgeconnect.ProxyAuthUserKey:     "user",
			edgeconnect.ProxyAuthPasswordKey: "pass",
		})
		kubeReader := fake.NewClient(createClientSecret(testSecretName, testNamespace), authRef)
		cfg, err := PrepareConfigFile(context.Background(), ec, kubeReader, testToken)

		require.NoError(t, err)

		expected := `name: test-name-edgeconnect
api_endpoint_host: abc12345.dynatrace.com
oauth:
    endpoint: https://test.com/sso/oauth2/token
    client_id: created-client-id
    client_secret: created-client-secret
    resource: urn:dtenvironment:test12345
root_certificate_paths:
    - /etc/ssl/certificate.cer
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
		ec := &edgeconnect.EdgeConnect{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testName,
				Namespace: testNamespace,
			},
			Spec: edgeconnect.EdgeConnectSpec{
				ApiServer: "abc12345.dynatrace.com",
				OAuth: edgeconnect.OAuthSpec{
					Endpoint:     "https://test.com/sso/oauth2/token",
					Resource:     "urn:dtenvironment:test12345",
					ClientSecret: "test-secret",
				},
				KubernetesAutomation: &edgeconnect.KubernetesAutomationSpec{
					Enabled: true,
				},
				ServiceAccountName: "test",
			},
		}
		testSecretName := "test-secret"
		kubeReader := fake.NewClient(createClientSecret(testSecretName, testNamespace))
		cfg, err := PrepareConfigFile(context.Background(), ec, kubeReader, testToken)

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

	t.Run("safeEdgeConnectCfg", func(t *testing.T) {
		cfg := config.EdgeConnect{
			Name:            "test",
			ApiEndpointHost: "test",
			OAuth: config.OAuth{
				ClientSecret: "super secret",
			},
		}
		expected := `name: test
api_endpoint_host: test
oauth:
    endpoint: ""
    client_id: ""
    client_secret: '****'
    resource: ""
`
		assert.Equal(t, expected, safeEdgeConnectCfg(cfg))
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
