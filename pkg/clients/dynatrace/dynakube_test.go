package dynatrace

import (
	"net/http"
	"testing"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/value"
	operatorversion "github.com/Dynatrace/dynatrace-operator/pkg/version"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

const (
	testNamespace   = "test-namespace"
	testAPIURL      = "https://test.endpoint.com/api"
	testCertsCMName = "test-certs-cm"
	testProxySecret = "test-proxy-secret"
	testProxyURL    = "http://proxy.example.com:8080"
	testNoProxy     = "no.proxy"
)

func Test_optionsFromDynakube(t *testing.T) {
	getDynakube := func() dynakube.DynaKube {
		return dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{Namespace: testNamespace},
			Spec:       dynakube.DynaKubeSpec{APIURL: testAPIURL},
		}
	}
	t.Run("sets base URL, tokens and default user agent", func(t *testing.T) {
		dk := getDynakube()

		opts, err := optionsFromDynakube(t.Context(), fake.NewClient(), dk, testAPIToken, testPaasToken, "")
		require.NoError(t, err)

		_, cfg, err := getClientAndConfig(opts...)
		require.NoError(t, err)

		assert.Equal(t, testAPIToken, cfg.APIToken)
		assert.Equal(t, testPaasToken, cfg.PaasToken)
		assert.Equal(t, testAPIURL, cfg.BaseURL.String())
		assert.Equal(t, operatorversion.UserAgent(), cfg.UserAgent)
	})

	t.Run("falls back to API token value when no PaaS token provided", func(t *testing.T) {
		dk := getDynakube()

		opts, err := optionsFromDynakube(t.Context(), fake.NewClient(), dk, testAPIToken, "", "")
		require.NoError(t, err)

		_, cfg, err := getClientAndConfig(opts...)
		require.NoError(t, err)

		assert.Equal(t, testAPIToken, cfg.APIToken)
		assert.Empty(t, cfg.PaasToken)
	})

	t.Run("appends user agent suffix to default user agent", func(t *testing.T) {
		expUserAgent := "my-controller/1.0"
		dk := getDynakube()

		opts, err := optionsFromDynakube(t.Context(), fake.NewClient(), dk, testAPIToken, testPaasToken, expUserAgent)
		require.NoError(t, err)

		_, cfg, err := getClientAndConfig(opts...)
		require.NoError(t, err)

		assert.Equal(t, operatorversion.UserAgent()+" "+expUserAgent, cfg.UserAgent)
	})

	t.Run("sets InsecureSkipVerify on transport when SkipCertCheck is true", func(t *testing.T) {
		dk := getDynakube()
		dk.Spec.SkipCertCheck = true

		opts, err := optionsFromDynakube(t.Context(), fake.NewClient(), dk, testAPIToken, testPaasToken, "")
		require.NoError(t, err)

		client, cfg, err := getClientAndConfig(opts...)
		require.NoError(t, err)

		transport, ok := client.Transport.(*http.Transport)
		require.True(t, ok)

		require.NotNil(t, transport.TLSClientConfig)
		assert.True(t, transport.TLSClientConfig.InsecureSkipVerify)
		assert.True(t, cfg.TLSConfig.InsecureSkipVerify)
	})

	t.Run("TLS transport has no InsecureSkipVerify when SkipCertCheck is false", func(t *testing.T) {
		dk := getDynakube()
		dk.Spec.SkipCertCheck = false

		opts, err := optionsFromDynakube(t.Context(), fake.NewClient(), dk, testAPIToken, testPaasToken, "")
		require.NoError(t, err)

		client, cfg, err := getClientAndConfig(opts...)
		require.NoError(t, err)

		transport, ok := client.Transport.(*http.Transport)
		require.True(t, ok)

		assert.Nil(t, transport.TLSClientConfig)
		assert.Nil(t, cfg.TLSConfig)
	})

	t.Run("sets NetworkZone when configured", func(t *testing.T) {
		expNetworkZone := "zone-1"
		dk := getDynakube()
		dk.Spec.NetworkZone = expNetworkZone

		opts, err := optionsFromDynakube(t.Context(), fake.NewClient(), dk, testAPIToken, testPaasToken, "")
		require.NoError(t, err)

		_, cfg, err := getClientAndConfig(opts...)
		require.NoError(t, err)

		assert.Equal(t, expNetworkZone, cfg.NetworkZone)
	})

	t.Run("sets HostGroup when configured", func(t *testing.T) {
		expHostGroup := "test-host"
		dk := getDynakube()
		dk.Spec.OneAgent.HostGroup = expHostGroup

		opts, err := optionsFromDynakube(t.Context(), fake.NewClient(), dk, testAPIToken, testPaasToken, "")
		require.NoError(t, err)

		_, cfg, err := getClientAndConfig(opts...)
		require.NoError(t, err)

		assert.Equal(t, expHostGroup, cfg.HostGroup)
	})

	t.Run("does not set NetworkZone when empty", func(t *testing.T) {
		dk := getDynakube()

		opts, err := optionsFromDynakube(t.Context(), fake.NewClient(), dk, testAPIToken, testPaasToken, "")
		require.NoError(t, err)

		_, cfg, err := getClientAndConfig(opts...)
		require.NoError(t, err)

		assert.Empty(t, cfg.NetworkZone)
	})

	t.Run("loads custom root CAs from configmap into TLS config", func(t *testing.T) {
		dk := getDynakube()
		dk.Spec.TrustedCAs = testCertsCMName

		fakeClient := fake.NewClient(&corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Name: testCertsCMName, Namespace: testNamespace},
			Data:       map[string]string{dynakube.TrustedCAKey: customCA},
		})
		opts, err := optionsFromDynakube(t.Context(), fakeClient, dk, testAPIToken, testPaasToken, "")
		require.NoError(t, err)

		client, cfg, err := getClientAndConfig(opts...)
		require.NoError(t, err)

		transport, ok := client.Transport.(*http.Transport)
		require.True(t, ok)

		require.NotNil(t, transport.TLSClientConfig)
		assert.NotNil(t, transport.TLSClientConfig.RootCAs)
		assert.NotNil(t, cfg.TLSConfig.RootCAs)
	})

	t.Run("returns error when trusted CA configmap is missing", func(t *testing.T) {
		dk := getDynakube()
		dk.Spec.TrustedCAs = testCertsCMName

		_, err := optionsFromDynakube(t.Context(), fake.NewClient(), dk, testAPIToken, testPaasToken, "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get certificate configmap")
	})

	t.Run("returns error when trusted CA configmap has no certs field", func(t *testing.T) {
		dk := getDynakube()
		dk.Spec.TrustedCAs = testCertsCMName

		fakeClient := fake.NewClient(&corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Name: testCertsCMName, Namespace: testNamespace},
			Data:       map[string]string{},
		})
		_, err := optionsFromDynakube(t.Context(), fakeClient, dk, testAPIToken, testPaasToken, "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "missing field certs")
	})

	t.Run("sets proxy when proxy is defined via inline value (value:)", func(t *testing.T) {
		dk := getDynakube()
		dk.Spec.Proxy = &value.Source{Value: testProxyURL}
		dk.GetObjectMeta().SetAnnotations(map[string]string{
			"feature.dynatrace.com/no-proxy": testNoProxy,
		})

		opts, err := optionsFromDynakube(t.Context(), fake.NewClient(), dk, testAPIToken, testPaasToken, "")
		require.NoError(t, err)

		_, cfg, err := getClientAndConfig(opts...)
		require.NoError(t, err)

		assert.Equal(t, testProxyURL, cfg.Proxy)
		require.Equal(t, testNoProxy, cfg.NoProxy)
	})

	t.Run("sets proxy when proxy is defined via secret reference (valueFrom:)", func(t *testing.T) {
		dk := getDynakube()
		dk.Spec.Proxy = &value.Source{ValueFrom: testProxySecret}
		dk.GetObjectMeta().SetAnnotations(map[string]string{
			"feature.dynatrace.com/no-proxy": testNoProxy,
		})

		fakeClient := fake.NewClient(&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: testProxySecret, Namespace: testNamespace},
			Data:       map[string][]byte{dynakube.ProxyKey: []byte(testProxyURL)},
		})
		opts, err := optionsFromDynakube(t.Context(), fakeClient, dk, testAPIToken, testPaasToken, "")
		require.NoError(t, err)

		_, cfg, err := getClientAndConfig(opts...)
		require.NoError(t, err)

		require.Equal(t, testProxyURL, cfg.Proxy)
		require.Equal(t, testNoProxy, cfg.NoProxy)
	})

	t.Run("returns error when proxy secret is missing", func(t *testing.T) {
		dk := dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{Namespace: testNamespace},
			Spec: dynakube.DynaKubeSpec{
				APIURL: testAPIURL,
				Proxy:  &value.Source{ValueFrom: testProxySecret},
			},
		}
		_, err := optionsFromDynakube(t.Context(), fake.NewClient(), dk, testAPIToken, testPaasToken, "")

		require.Error(t, err)
	})

	t.Run("sets cacheTTL when cache TTL is set", func(t *testing.T) {
		expCacheTTL := ptr.To(uint16(5))
		expAPIRequestThreshold := time.Duration(*expCacheTTL) * time.Minute
		dk := getDynakube()
		dk.Spec.DynatraceAPIRequestThreshold = expCacheTTL

		opts, err := optionsFromDynakube(t.Context(), fake.NewClient(), dk, testAPIToken, testPaasToken, "")
		require.NoError(t, err)

		_, cfg, err := getClientAndConfig(opts...)
		require.NoError(t, err)

		require.Equal(t, expAPIRequestThreshold, cfg.CacheEntryTTL)
	})
}

func TestNewClientFromDynakube(t *testing.T) {
	t.Run("returns a fully initialized client", func(t *testing.T) {
		dk := dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{Namespace: testNamespace},
			Spec:       dynakube.DynaKubeSpec{APIURL: testAPIURL},
		}
		dtClient, err := NewClientFromDynakube(t.Context(), fake.NewClient(), dk, testAPIToken, testPaasToken, "")

		require.NoError(t, err)
		require.NotNil(t, dtClient)
		assert.NotNil(t, dtClient.Settings)
		assert.NotNil(t, dtClient.ActiveGate)
		assert.NotNil(t, dtClient.HostEvent)
		assert.NotNil(t, dtClient.OneAgent)
		assert.NotNil(t, dtClient.Version)
		assert.NotNil(t, dtClient.Token)
	})

	t.Run("propagates option building error", func(t *testing.T) {
		dtClient, err := NewClientFromDynakube(t.Context(), fake.NewClient(), dynakube.DynaKube{}, "", "", "")

		require.Error(t, err)
		assert.Nil(t, dtClient)
	})
}
