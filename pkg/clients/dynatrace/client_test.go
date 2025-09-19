package dynatrace

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClient(t *testing.T) {
	{
		c, err := NewClient("https://aabb.live.dynatrace.com/api", "foo", "bar")
		require.NoError(t, err)
		assert.NotNil(t, c)
	}
	{
		c, err := NewClient("https://aabb.live.dynatrace.com/api", "foo", "bar", SkipCertificateValidation(false))
		require.NoError(t, err)
		assert.NotNil(t, c)
	}
	{
		c, err := NewClient("https://aabb.live.dynatrace.com/api", "foo", "bar", SkipCertificateValidation(true))
		require.NoError(t, err)
		assert.NotNil(t, c)
	}

	{
		_, err := NewClient("https://aabb.live.dynatrace.com/api", "", "")
		require.Error(t, err, "tokens are empty")
	}
	{
		_, err := NewClient("", "foo", "bar")
		require.Error(t, err, "empty URL")
	}
}

func TestProxy(t *testing.T) {
	proxyRawURL := "proxy.url"

	// each subtest needs new instance of the http Server or actually http Server-Client pair
	// because otherwise a modification of the client.Transport.Proxy field affects http Client
	// in the subsequent subtests

	t.Run("set correct proxy (both http and https)", func(t *testing.T) {
		dynatraceServer, _ := createTestDynatraceServer(t, http.NotFoundHandler(), "")
		defer dynatraceServer.Close()

		dtc := createTestDynatraceClient(*dynatraceServer)
		options := Proxy(proxyRawURL, "")
		assert.NotNil(t, options)
		options(&dtc)

		transport := dtc.httpClient.Transport.(*http.Transport)

		checkProxyForUrl(t, *transport, proxyRawURL, "http://working.url", false)
		checkProxyForUrl(t, *transport, proxyRawURL, "https://working.url", false)
	})
	t.Run("set NO_PROXY", func(t *testing.T) {
		dynatraceServer, _ := createTestDynatraceServer(t, http.NotFoundHandler(), "")
		defer dynatraceServer.Close()

		dtc := createTestDynatraceClient(*dynatraceServer)
		noProxy := "working.url,url.working"
		options := Proxy(proxyRawURL, noProxy)
		assert.NotNil(t, options)
		options(&dtc)

		transport := dtc.httpClient.Transport.(*http.Transport)

		checkProxyForUrl(t, *transport, proxyRawURL, "http://working.url", true)
		checkProxyForUrl(t, *transport, proxyRawURL, "https://working.url", true)

		checkProxyForUrl(t, *transport, proxyRawURL, "http://url.working", true)
		checkProxyForUrl(t, *transport, proxyRawURL, "https://url.working", true)

		checkProxyForUrl(t, *transport, proxyRawURL, "http://proxied.url", false)
		checkProxyForUrl(t, *transport, proxyRawURL, "https://proxied.url", false)
	})
	t.Run("set incorrect proxy", func(t *testing.T) {
		dynatraceServer, _ := createTestDynatraceServer(t, http.NotFoundHandler(), "")
		defer dynatraceServer.Close()

		dtc := createTestDynatraceClient(*dynatraceServer)

		options := Proxy("{!.*&%", "")
		require.NotNil(t, options)
		options(&dtc)
		assert.Nil(t, dtc.httpClient.Transport.(*http.Transport).Proxy)
	})
}

func TestCerts(t *testing.T) {
	dynatraceServer, _ := createTestDynatraceServer(t, http.NotFoundHandler(), "")
	defer dynatraceServer.Close()

	dtc := createTestDynatraceClient(*dynatraceServer)
	transport := dtc.httpClient.Transport.(*http.Transport)

	certs := Certs(nil)
	assert.NotNil(t, certs)
	certs(&dtc)
	assert.NotNil(t, transport.TLSClientConfig.RootCAs)
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

func createTestDynatraceClient(dynatraceServer httptest.Server) dynatraceClient {
	return dynatraceClient{
		url:        dynatraceServer.URL,
		apiToken:   apiToken,
		paasToken:  paasToken,
		hostCache:  nil,
		httpClient: dynatraceServer.Client(),
	}
}
