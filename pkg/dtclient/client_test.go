package dtclient

import (
	"net/http"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewClient(t *testing.T) {
	{
		c, err := NewClient("https://aabb.live.dynatrace.com/api", "foo", "bar")
		if assert.NoError(t, err) {
			assert.NotNil(t, c)
		}
	}
	{
		c, err := NewClient("https://aabb.live.dynatrace.com/api", "foo", "bar", SkipCertificateValidation(false))
		if assert.NoError(t, err) {
			assert.NotNil(t, c)
		}
	}
	{
		c, err := NewClient("https://aabb.live.dynatrace.com/api", "foo", "bar", SkipCertificateValidation(true))
		if assert.NoError(t, err) {
			assert.NotNil(t, c)
		}
	}

	{
		_, err := NewClient("https://aabb.live.dynatrace.com/api", "", "")
		assert.Error(t, err, "tokens are empty")
	}
	{
		_, err := NewClient("", "foo", "bar")
		assert.Error(t, err, "empty URL")
	}
}

func TestProxy(t *testing.T) {
	dynatraceServer, _ := createTestDynatraceClient(t, clientHandlerStub())
	defer dynatraceServer.Close()

	dtc := dynatraceClient{
		url:        dynatraceServer.URL,
		apiToken:   apiToken,
		paasToken:  paasToken,
		httpClient: dynatraceServer.Client(),
		hostCache:  nil,
		logger:     logf.Log.WithName("dtc"),
	}
	transport := dtc.httpClient.Transport.(*http.Transport)
	rawURL := "working.url"

	options := Proxy(rawURL)
	assert.NotNil(t, options)
	options(&dtc)

	url, err := transport.Proxy(&http.Request{})
	assert.NoError(t, err)
	assert.NotNil(t, url)
	assert.Equal(t, rawURL, url.Path)

	options = Proxy("{!.*&%")
	assert.NotNil(t, options)
	options(&dtc)
}

func TestCerts(t *testing.T) {
	dynatraceServer, _ := createTestDynatraceClient(t, clientHandlerStub())
	defer dynatraceServer.Close()

	dtc := dynatraceClient{
		url:        dynatraceServer.URL,
		apiToken:   apiToken,
		paasToken:  paasToken,
		httpClient: dynatraceServer.Client(),
		hostCache:  nil,
		logger:     logf.Log.WithName("dtc"),
	}
	transport := dtc.httpClient.Transport.(*http.Transport)

	certs := Certs(nil)
	assert.NotNil(t, certs)
	certs(&dtc)
	assert.Equal(t, [][]uint8{}, transport.TLSClientConfig.RootCAs.Subjects())
}

func clientHandlerStub() http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {

	}
}
