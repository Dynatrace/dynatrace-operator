package registry

import (
	"context"
	"net/http"
	"net/url"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/value"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestProxy(t *testing.T) {
	proxyRawURL := "proxy.url"

	t.Run("set NO_PROXY", func(t *testing.T) {
		dk := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "Dynakube",
				Namespace: "dynatrace",
				Annotations: map[string]string{
					dynakube.AnnotationFeatureNoProxy: "working.url,url.working",
				},
			},
			Spec: dynakube.DynaKubeSpec{
				Proxy:  &value.Source{Value: proxyRawURL},
				APIURL: "https://testApiUrl.dev.dynatracelabs.com/api",
				OneAgent: dynakube.OneAgentSpec{
					CloudNativeFullStack: &dynakube.CloudNativeFullStackSpec{},
				},
			},
		}
		transport := http.DefaultTransport.(*http.Transport).Clone()
		transport, err := PrepareTransportForDynaKube(context.TODO(), nil, transport, dk)

		require.NoError(t, err)

		checkProxyForUrl(t, *transport, proxyRawURL, "http://working.url", true)
		checkProxyForUrl(t, *transport, proxyRawURL, "https://working.url", true)

		checkProxyForUrl(t, *transport, proxyRawURL, "http://url.working", true)
		checkProxyForUrl(t, *transport, proxyRawURL, "https://url.working", true)

		checkProxyForUrl(t, *transport, proxyRawURL, "http://proxied.url", false)
		checkProxyForUrl(t, *transport, proxyRawURL, "https://proxied.url", false)
	})
}

func TestSkipCertCheck(t *testing.T) {
	dk := &dynakube.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "Dynakube",
			Namespace: "dynatrace",
		},
		Spec: dynakube.DynaKubeSpec{
			APIURL: "https://testApiUrl.dev.dynatracelabs.com/api",
			OneAgent: dynakube.OneAgentSpec{
				CloudNativeFullStack: &dynakube.CloudNativeFullStackSpec{},
			},
		},
	}

	t.Run("has skipCertCheck enabled", func(t *testing.T) {
		dk.Spec.SkipCertCheck = true
		transport := http.DefaultTransport.(*http.Transport).Clone()
		transport, err := PrepareTransportForDynaKube(context.TODO(), nil, transport, dk)
		require.NoError(t, err)
		assert.True(t, transport.TLSClientConfig.InsecureSkipVerify)
	})
	t.Run("has skipCertCheck disabled", func(t *testing.T) {
		dk.Spec.SkipCertCheck = false
		transport := http.DefaultTransport.(*http.Transport).Clone()
		transport, err := PrepareTransportForDynaKube(context.TODO(), nil, transport, dk)
		require.NoError(t, err)
		assert.False(t, transport.TLSClientConfig.InsecureSkipVerify)
	})
}

func checkProxyForUrl(t *testing.T, transport http.Transport, proxyRawURL, targetRawURL string, noProxy bool) {
	targetURL, err := url.Parse(targetRawURL)
	require.NoError(t, err)

	url, err := transport.Proxy(&http.Request{URL: targetURL})
	require.NoError(t, err)

	if !noProxy {
		require.NotNil(t, url)
		assert.Equal(t, proxyRawURL, url.Host)
	} else {
		require.Nil(t, url)
	}
}
