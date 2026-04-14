package dynatrace

import (
	"crypto/tls"
	"errors"
	"net/http"
	"net/url"
	"testing"
	"time"

	operatorversion "github.com/Dynatrace/dynatrace-operator/pkg/version"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClientV2(t *testing.T) {
	t.Run("creates client with all sub-clients initialized", func(t *testing.T) {
		client, err := NewClientV2("https://aabb.test.com/api")
		require.NoError(t, err)
		require.NotNil(t, client)
		assert.NotNil(t, client.Settings)
		assert.NotNil(t, client.ActiveGate)
		assert.NotNil(t, client.HostEvent)
		assert.NotNil(t, client.OneAgent)
		assert.NotNil(t, client.Version)
		assert.NotNil(t, client.Token)
	})

	t.Run("appends /api when path does not end with /api", func(t *testing.T) {
		client, err := NewClientV2("https://aabb.test.com")
		require.NoError(t, err)
		require.NotNil(t, client)
	})

	t.Run("does not duplicate /api when URL already ends with /api", func(t *testing.T) {
		client, err := NewClientV2("https://aabb.test.com/api")
		require.NoError(t, err)
		require.NotNil(t, client)
	})

	t.Run("returns error on invalid URL", func(t *testing.T) {
		_, err := NewClientV2("://invalid-url")
		require.Error(t, err)
	})

	t.Run("applies WithAPIToken option", func(t *testing.T) {
		client, err := NewClientV2(
			"https://aabb.test.com/api",
			WithAPIToken("my-api-token"),
		)
		require.NoError(t, err)
		require.NotNil(t, client)
	})

	t.Run("applies WithPaasToken option", func(t *testing.T) {
		client, err := NewClientV2(
			"https://aabb.test.com/api",
			WithPaasToken("my-paas-token"),
		)
		require.NoError(t, err)
		require.NotNil(t, client)
	})

	t.Run("applies WithNetworkZone option", func(t *testing.T) {
		client, err := NewClientV2(
			"https://aabb.test.com/api",
			WithNetworkZone("eu-west"),
		)
		require.NoError(t, err)
		require.NotNil(t, client)
	})

	t.Run("applies WithHostGroup option", func(t *testing.T) {
		client, err := NewClientV2(
			"https://aabb.test.com/api",
			WithHostGroup("group-a"),
		)
		require.NoError(t, err)
		require.NotNil(t, client)
	})

	t.Run("applies WithUserAgentSuffix option", func(t *testing.T) {
		client, err := NewClientV2(
			"https://aabb.test.com/api",
			WithUserAgentSuffix("my-suffix"),
		)
		require.NoError(t, err)
		require.NotNil(t, client)
	})

	t.Run("WithUserAgentSuffix empty suffix is no-op", func(t *testing.T) {
		client, err := NewClientV2(
			"https://aabb.test.com/api",
			WithUserAgentSuffix(""),
		)
		require.NoError(t, err)
		require.NotNil(t, client)
	})

	t.Run("returns error when HTTP option fails", func(t *testing.T) {
		bad := badTransportHTTPClient()
		_, err := NewClientV2(
			"https://aabb.test.com/api",
			WithHTTPClient(bad),
		)
		require.Error(t, err)
	})

	t.Run("applies multiple options", func(t *testing.T) {
		client, err := NewClientV2(
			"https://aabb.test.com/api",
			WithAPIToken("token"),
			WithPaasToken("paas"),
			WithNetworkZone("zone"),
			WithHostGroup("grp"),
			WithUserAgentSuffix("sfx"),
		)
		require.NoError(t, err)
		require.NotNil(t, client)
	})
}

func TestNewOAuthClient(t *testing.T) {
	t.Run("creates OAuth client successfully", func(t *testing.T) {
		client, err := NewOAuthClient(
			"https://aabb.test.com",
			WithClientID("client-id"),
			WithClientSecret("client-secret"),
			WithTokenURL("https://sso.test.com/sso/oauth2/token"),
		)
		require.NoError(t, err)
		require.NotNil(t, client)
		assert.NotNil(t, client.EdgeConnect)
	})

	t.Run("returns error on invalid URL", func(t *testing.T) {
		_, err := NewOAuthClient("://bad-url")
		require.Error(t, err)
	})

	t.Run("applies WithOAuthScopes", func(t *testing.T) {
		client, err := NewOAuthClient(
			"https://aabb.test.com",
			WithOAuthScopes([]string{"scope1", "scope2"}),
		)
		require.NoError(t, err)
		require.NotNil(t, client)
	})

	t.Run("returns error when HTTP option fails", func(t *testing.T) {
		bad := badTransportHTTPClient()
		_, err := NewOAuthClient(
			"https://aabb.test.com",
			WithHTTPClient(bad),
		)
		require.Error(t, err)
	})
}

func TestWithAPIToken(t *testing.T) {
	cfg := &ConfigV2{}
	opt := WithAPIToken("token")
	require.NoError(t, opt(cfg))
	assert.Equal(t, "token", cfg.APIToken)
}

func TestWithPaasToken(t *testing.T) {
	cfg := &ConfigV2{}
	opt := WithPaasToken("paas")
	require.NoError(t, opt(cfg))
	assert.Equal(t, "paas", cfg.PaasToken)
}

func TestWithNetworkZone(t *testing.T) {
	cfg := &ConfigV2{}
	opt := WithNetworkZone("zone")
	require.NoError(t, opt(cfg))
	assert.Equal(t, "zone", cfg.NetworkZone)
}

func TestWithHostGroup(t *testing.T) {
	cfg := &ConfigV2{}
	opt := WithHostGroup("hg")
	require.NoError(t, opt(cfg))
	assert.Equal(t, "hg", cfg.HostGroup)
}

func TestWithV2UserAgentSuffix(t *testing.T) {
	t.Run("appends suffix when non-empty", func(t *testing.T) {
		cfg := &ConfigV2{UserAgent: "agent"}
		require.NoError(t, WithUserAgentSuffix("sfx")(cfg))
		assert.Equal(t, "agent sfx", cfg.UserAgent)
	})

	t.Run("no change when suffix is empty", func(t *testing.T) {
		cfg := &ConfigV2{UserAgent: "agent"}
		require.NoError(t, WithUserAgentSuffix("")(cfg))
		assert.Equal(t, "agent", cfg.UserAgent)
	})
}

func TestWithClientID(t *testing.T) {
	cfg := &ConfigV2{}
	require.NoError(t, WithClientID("id")(cfg))
	assert.Equal(t, "id", cfg.ClientID)
}

func TestWithClientSecret(t *testing.T) {
	cfg := &ConfigV2{}
	require.NoError(t, WithClientSecret("secret")(cfg))
	assert.Equal(t, "secret", cfg.ClientSecret)
}

func TestWithTokenURL(t *testing.T) {
	cfg := &ConfigV2{}
	require.NoError(t, WithTokenURL("https://sso.example.com/token")(cfg))
	assert.Equal(t, "https://sso.example.com/token", cfg.TokenURL)
}

func TestWithOAuthScopes(t *testing.T) {
	cfg := &ConfigV2{}
	scopes := []string{"scope1", "scope2"}
	require.NoError(t, WithOAuthScopes(scopes)(cfg))
	assert.Equal(t, scopes, cfg.Scopes)
}

func TestWithHTTPClientOption(t *testing.T) {
	existing := &http.Client{Transport: &http.Transport{}}
	cfg := &ConfigV2{}
	require.NoError(t, WithHTTPClient(existing)(cfg))
	assert.Same(t, existing, cfg.HTTPClient)
}

func TestWithTimeoutOption(t *testing.T) {
	cfg := &ConfigV2{}
	require.NoError(t, WithTimeout(5*time.Second)(cfg))
	assert.Equal(t, 5*time.Second, cfg.Timeout)
}

func TestWithProxyOption(t *testing.T) {
	cfg := &ConfigV2{}
	require.NoError(t, WithProxy("http://p.example.com", "no.proxy")(cfg))
	assert.Equal(t, "http://p.example.com", cfg.Proxy)
	assert.Equal(t, "no.proxy", cfg.NoProxy)
}

func TestWithTLSConfigOption(t *testing.T) {
	tlsCfg := &tls.Config{MinVersion: tls.VersionTLS13}
	cfg := &ConfigV2{}
	require.NoError(t, WithTLSConfig(tlsCfg)(cfg))
	assert.Equal(t, tlsCfg, cfg.TLSConfig)
}

func TestWithKeepAliveOption(t *testing.T) {
	cfg := &ConfigV2{}

	require.NoError(t, WithKeepAlive(false)(cfg))
	assert.True(t, cfg.DisableKeepAlives)

	require.NoError(t, WithKeepAlive(true)(cfg))
	assert.False(t, cfg.DisableKeepAlives)
}

func TestWithSkipCertificateValidationOption(t *testing.T) {
	t.Run("skip creates TLS config with InsecureSkipVerify", func(t *testing.T) {
		cfg := &ConfigV2{}
		require.NoError(t, WithSkipCertificateValidation(true)(cfg))
		require.NotNil(t, cfg.TLSConfig)
		assert.True(t, cfg.TLSConfig.InsecureSkipVerify)
	})

	t.Run("skip with existing TLS config reuses it", func(t *testing.T) {
		existing := &tls.Config{MinVersion: tls.VersionTLS12}
		cfg := &ConfigV2{TLSConfig: existing}
		require.NoError(t, WithSkipCertificateValidation(true)(cfg))
		assert.Same(t, existing, cfg.TLSConfig)
		assert.True(t, cfg.TLSConfig.InsecureSkipVerify)
	})

	t.Run("no skip is a no-op", func(t *testing.T) {
		cfg := &ConfigV2{}
		require.NoError(t, WithSkipCertificateValidation(false)(cfg))
		assert.Nil(t, cfg.TLSConfig)
	})
}

func TestWithCertsOption(t *testing.T) {
	t.Run("nil certs is a no-op", func(t *testing.T) {
		cfg := &ConfigV2{}
		require.NoError(t, WithCerts(nil)(cfg))
		assert.Nil(t, cfg.TLSConfig)
	})

	t.Run("invalid PEM returns error", func(t *testing.T) {
		cfg := &ConfigV2{}
		err := WithCerts([]byte("not-a-valid-pem"))(cfg)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to append custom certs")
	})
}

func TestGetConfig(t *testing.T) {
	t.Run("default, a config with client default timeout and user agent", func(t *testing.T) {
		c, err := getConfig()
		require.NoError(t, err)
		require.NotNil(t, c)
		assert.Equal(t, 30*time.Second, c.HTTPClient.Timeout)
		assert.Equal(t, operatorversion.UserAgent(), c.UserAgent)
	})

	t.Run("with different options", func(t *testing.T) {
		c, err := getConfig(
			WithClientID("id"),
			WithClientSecret("secret"),
			WithAPIToken("apitoken"),
			WithPaasToken("paastoken"),
			WithNetworkZone("network"),
			WithHostGroup("hostgroup"),
			WithUserAgentSuffix("useragent"))

		require.NoError(t, err)
		assert.Equal(t, "id", c.ClientID)
		assert.Equal(t, "secret", c.ClientSecret)
		assert.Equal(t, "apitoken", c.APIToken)
		assert.Equal(t, "paastoken", c.PaasToken)
		assert.Equal(t, "network", c.NetworkZone)
		assert.Equal(t, "hostgroup", c.HostGroup)
		assert.Equal(t, operatorversion.UserAgent()+" useragent", c.UserAgent)
	})

	t.Run("WithTimeout overrides default", func(t *testing.T) {
		c, err := getConfig(WithTimeout(5 * time.Second))
		require.NoError(t, err)
		assert.Equal(t, 5*time.Second, c.HTTPClient.Timeout)
	})

	t.Run("WithProxy configures transport proxy", func(t *testing.T) {
		c, err := getConfig(WithProxy("http://proxy.example.com:8080", ""))
		require.NoError(t, err)
		require.NotNil(t, c)

		transport, ok := c.HTTPClient.Transport.(*http.Transport)
		require.True(t, ok)
		require.NotNil(t, transport.Proxy)

		proxyURL, err := transport.Proxy(&http.Request{URL: mustParseURL("https://some.target.com")})
		require.NoError(t, err)
		require.NotNil(t, proxyURL)
	})

	t.Run("WithProxy with NoProxy excludes matching hosts", func(t *testing.T) {
		c, err := getConfig(WithProxy("http://proxy.example.com:8080", "excluded.host"))
		require.NoError(t, err)

		transport, ok := c.HTTPClient.Transport.(*http.Transport)
		require.True(t, ok)

		noProxyURL, err := transport.Proxy(&http.Request{URL: mustParseURL("https://excluded.host/path")})
		require.NoError(t, err)
		assert.Nil(t, noProxyURL)

		proxiedURL, err := transport.Proxy(&http.Request{URL: mustParseURL("https://other.host/path")})
		require.NoError(t, err)
		require.NotNil(t, proxiedURL)
	})

	t.Run("WithProxy invalid URL returns error", func(t *testing.T) {
		_, err := getConfig(WithProxy("://bad-proxy", ""))
		require.Error(t, err)
	})

	t.Run("WithSkipCertificateValidation true sets InsecureSkipVerify", func(t *testing.T) {
		c, err := getConfig(WithSkipCertificateValidation(true))
		require.NoError(t, err)

		transport, ok := c.HTTPClient.Transport.(*http.Transport)
		require.True(t, ok)
		require.NotNil(t, transport.TLSClientConfig)
		assert.True(t, transport.TLSClientConfig.InsecureSkipVerify)
	})

	t.Run("WithSkipCertificateValidation false is no-op", func(t *testing.T) {
		c, err := getConfig(WithSkipCertificateValidation(false))
		require.NoError(t, err)

		transport, ok := c.HTTPClient.Transport.(*http.Transport)
		require.True(t, ok)
		assert.Nil(t, transport.TLSClientConfig)
	})

	t.Run("WithTLSConfig sets TLS config on transport", func(t *testing.T) {
		tlsCfg := &tls.Config{MinVersion: tls.VersionTLS13}
		c, err := getConfig(WithTLSConfig(tlsCfg))
		require.NoError(t, err)

		transport, ok := c.HTTPClient.Transport.(*http.Transport)
		require.True(t, ok)
		assert.Equal(t, tlsCfg, transport.TLSClientConfig)
	})

	t.Run("WithKeepAlive false disables keep-alives", func(t *testing.T) {
		c, err := getConfig(WithKeepAlive(false))
		require.NoError(t, err)

		transport, ok := c.HTTPClient.Transport.(*http.Transport)
		require.True(t, ok)
		assert.True(t, transport.DisableKeepAlives)
	})

	t.Run("WithKeepAlive true keeps keep-alives enabled", func(t *testing.T) {
		c, err := getConfig(WithKeepAlive(true))
		require.NoError(t, err)

		transport, ok := c.HTTPClient.Transport.(*http.Transport)
		require.True(t, ok)
		assert.False(t, transport.DisableKeepAlives)
	})

	t.Run("WithHTTPClient uses provided client", func(t *testing.T) {
		existing := &http.Client{Transport: &http.Transport{}}
		c, err := getConfig(WithHTTPClient(existing))
		require.NoError(t, err)
		assert.Same(t, existing, c.HTTPClient)
	})

	t.Run("WithHTTPClient with unexpected transport type returns error", func(t *testing.T) {
		bad := &http.Client{Transport: &badTransport{}}
		_, err := getConfig(WithHTTPClient(bad))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unexpected transport type")
	})

	t.Run("WithCerts empty bytes is no-op", func(t *testing.T) {
		c, err := getConfig(WithCerts(nil))
		require.NoError(t, err)
		require.NotNil(t, c.HTTPClient)
	})
}

func TestAsV2(t *testing.T) {
	dtc := &dynatraceClient{
		url:             "https://aabb.test.com/api",
		apiToken:        "api",
		paasToken:       "paas",
		networkZone:     "zone",
		hostGroup:       "hg",
		userAgentSuffix: "sfx",
		httpClient:      &http.Client{Transport: &http.Transport{}},
	}

	v2 := dtc.AsV2()
	require.NotNil(t, v2)
	assert.NotNil(t, v2.Settings)
	assert.NotNil(t, v2.ActiveGate)
	assert.NotNil(t, v2.HostEvent)
	assert.NotNil(t, v2.OneAgent)
	assert.NotNil(t, v2.Version)
	assert.NotNil(t, v2.Token)
}

type badTransport struct{}

func (b *badTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, errors.New("bad transport")
}

// badTransportHTTPClient returns an http.Client whose Transport is not *http.Transport,
func badTransportHTTPClient() *http.Client {
	return &http.Client{Transport: &badTransport{}}
}

func mustParseURL(raw string) *url.URL {
	u, err := url.Parse(raw)
	if err != nil {
		panic(err)
	}

	return u
}
