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
	"golang.org/x/oauth2/clientcredentials"
)

func TestNewClient(t *testing.T) {
	t.Run("creates client with all sub-clients initialized", func(t *testing.T) {
		client, err := NewClient(WithBaseURL("https://aabb.test.com/api"), WithAPIToken("foo"), WithPaasToken("bar"))
		require.NoError(t, err)
		require.NotNil(t, client)
		assert.NotNil(t, client.Settings)
		assert.NotNil(t, client.ActiveGate)
		assert.NotNil(t, client.HostEvent)
		assert.NotNil(t, client.OneAgent)
		assert.NotNil(t, client.Version)
		assert.NotNil(t, client.Token)
	})

	t.Run("returns error on invalid option", func(t *testing.T) {
		_, err := NewClient(WithBaseURL("://invalid-url"))
		require.Error(t, err)
	})

	t.Run("returns error on empty option", func(t *testing.T) {
		_, err := NewClient(WithBaseURL(""))
		require.Error(t, err)
	})
}

func TestNewOAuthClient(t *testing.T) {
	t.Run("creates OAuth client successfully", func(t *testing.T) {
		client, err := NewOAuthClient(
			clientcredentials.Config{
				ClientID:     "client-id",
				ClientSecret: "client-secret",
				TokenURL:     "https://sso.test.com/sso/oauth2/token",
			},
			WithBaseURL("https://aabb.test.com"),
		)
		require.NoError(t, err)
		require.NotNil(t, client)
		assert.NotNil(t, client.EdgeConnect)
	})

	t.Run("returns error on invalid option", func(t *testing.T) {
		_, err := NewOAuthClient(clientcredentials.Config{}, WithBaseURL("://bad-url"))
		require.Error(t, err)
	})
}

func TestWithAPIToken(t *testing.T) {
	cfg := &Config{}
	require.NoError(t, WithAPIToken("token")(cfg))
	assert.Equal(t, "token", cfg.APIToken)
}

func TestWithPaasToken(t *testing.T) {
	cfg := &Config{}
	require.NoError(t, WithPaasToken("paas")(cfg))
	assert.Equal(t, "paas", cfg.PaasToken)
}

func TestWithNetworkZone(t *testing.T) {
	cfg := &Config{}
	require.NoError(t, WithNetworkZone("zone")(cfg))
	assert.Equal(t, "zone", cfg.NetworkZone)
}

func TestWithHostGroup(t *testing.T) {
	cfg := &Config{}
	require.NoError(t, WithHostGroup("hg")(cfg))
	assert.Equal(t, "hg", cfg.HostGroup)
}

func TestWithBaseURL(t *testing.T) {
	t.Run("valid URL is parsed and stored", func(t *testing.T) {
		cfg := &Config{}
		require.NoError(t, WithBaseURL("https://aabb.test.com")(cfg))
		require.NotNil(t, cfg.BaseURL)
		assert.Equal(t, "https://aabb.test.com", cfg.BaseURL.String())
	})

	t.Run("invalid URL returns error", func(t *testing.T) {
		cfg := &Config{}
		err := WithBaseURL("://invalid-url")(cfg)
		require.Error(t, err)
		assert.Nil(t, cfg.BaseURL)
	})
}

func TestWithUserAgentSuffix(t *testing.T) {
	t.Run("appends suffix when non-empty", func(t *testing.T) {
		cfg := &Config{UserAgent: "agent"}
		require.NoError(t, WithUserAgentSuffix("sfx")(cfg))
		assert.Equal(t, "agent sfx", cfg.UserAgent)
	})

	t.Run("no change when suffix is empty", func(t *testing.T) {
		cfg := &Config{UserAgent: "agent"}
		require.NoError(t, WithUserAgentSuffix("")(cfg))
		assert.Equal(t, "agent", cfg.UserAgent)
	})
}

func TestWithHTTPClient(t *testing.T) {
	existing := &http.Client{Transport: &http.Transport{}}
	cfg := &Config{}
	require.NoError(t, WithHTTPClient(existing)(cfg))
	assert.Same(t, existing, cfg.HTTPClient)
}

func TestWithProxy(t *testing.T) {
	cfg := &Config{}
	require.NoError(t, WithProxy("http://p.example.com", "no.proxy")(cfg))
	assert.Equal(t, "http://p.example.com", cfg.Proxy)
	assert.Equal(t, "no.proxy", cfg.NoProxy)
}

func TestWithTLSConfig(t *testing.T) {
	tlsCfg := &tls.Config{MinVersion: tls.VersionTLS13}
	cfg := &Config{}
	require.NoError(t, WithTLSConfig(tlsCfg)(cfg))
	assert.Equal(t, tlsCfg, cfg.TLSConfig)
}

func TestWithKeepAlive(t *testing.T) {
	cfg := &Config{}

	require.NoError(t, WithKeepAlive(false)(cfg))
	assert.True(t, cfg.DisableKeepAlives)

	require.NoError(t, WithKeepAlive(true)(cfg))
	assert.False(t, cfg.DisableKeepAlives)
}

func TestWithSkipCertificateValidation(t *testing.T) {
	t.Run("skip creates TLS config with InsecureSkipVerify", func(t *testing.T) {
		cfg := &Config{}
		require.NoError(t, WithSkipCertificateValidation(true)(cfg))
		require.NotNil(t, cfg.TLSConfig)
		assert.True(t, cfg.TLSConfig.InsecureSkipVerify)
	})

	t.Run("skip with existing TLS config reuses it", func(t *testing.T) {
		existing := &tls.Config{MinVersion: tls.VersionTLS12}
		cfg := &Config{TLSConfig: existing}
		require.NoError(t, WithSkipCertificateValidation(true)(cfg))
		assert.Same(t, existing, cfg.TLSConfig)
		assert.True(t, cfg.TLSConfig.InsecureSkipVerify)
	})

	t.Run("no skip is a no-op", func(t *testing.T) {
		cfg := &Config{}
		require.NoError(t, WithSkipCertificateValidation(false)(cfg))
		assert.Nil(t, cfg.TLSConfig)
	})
}

func TestWithCerts(t *testing.T) {
	t.Run("nil certs is a no-op", func(t *testing.T) {
		cfg := &Config{}
		require.NoError(t, WithCerts(nil)(cfg))
		assert.Nil(t, cfg.TLSConfig)
	})

	t.Run("valid cert is set on the tls config", func(t *testing.T) {
		cfg := &Config{}
		require.NoError(t, WithCerts([]byte(customCA))(cfg))
		assert.NotNil(t, cfg.TLSConfig)
		assert.NotNil(t, cfg.TLSConfig.RootCAs)
	})

	t.Run("invalid PEM returns error", func(t *testing.T) {
		cfg := &Config{}
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
			WithAPIToken("apitoken"),
			WithPaasToken("paastoken"),
			WithNetworkZone("network"),
			WithHostGroup("hostgroup"),
			WithBaseURL("https://aabb.test.com"),
			WithUserAgentSuffix("useragent"),
			WithKeepAlive(true),
		)

		require.NoError(t, err)
		assert.Equal(t, "apitoken", c.APIToken)
		assert.Equal(t, "paastoken", c.PaasToken)
		assert.Equal(t, "network", c.NetworkZone)
		assert.Equal(t, "hostgroup", c.HostGroup)
		assert.False(t, c.DisableKeepAlives)
		assert.Equal(t, operatorversion.UserAgent()+" useragent", c.UserAgent)
	})

	t.Run("WithProxy configures transport proxy", func(t *testing.T) {
		c, err := getConfig(WithProxy("http://proxy.example.com:8080", ""))
		require.NoError(t, err)
		require.NotNil(t, c)
		assert.Equal(t, "http://proxy.example.com:8080", c.Proxy)

		transport, ok := c.HTTPClient.Transport.(*http.Transport)
		require.True(t, ok)
		require.NotNil(t, transport.Proxy)

		proxyURL, err := transport.Proxy(&http.Request{URL: mustParseURL(t, "https://some.target.com")})
		require.NoError(t, err)
		require.NotNil(t, proxyURL)
	})

	t.Run("WithProxy excludes matching no proxy hosts", func(t *testing.T) {
		c, err := getConfig(WithProxy("http://proxy.example.com:8080", "excluded.host"))
		require.NoError(t, err)
		assert.Equal(t, "excluded.host", c.NoProxy)

		transport, ok := c.HTTPClient.Transport.(*http.Transport)
		require.True(t, ok)

		noProxyURL, err := transport.Proxy(&http.Request{URL: mustParseURL(t, "https://excluded.host/path")})
		require.NoError(t, err)
		assert.Nil(t, noProxyURL)

		proxiedURL, err := transport.Proxy(&http.Request{URL: mustParseURL(t, "https://other.host/path")})
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
}

type badTransport struct{}

func (b *badTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, errors.New("bad transport")
}

func mustParseURL(t *testing.T, raw string) *url.URL {
	t.Helper()
	u, err := url.Parse(raw)
	require.NoError(t, err)

	return u
}

// Generated with:
// openssl genrsa -out ca.key 2048
// openssl req -x509 -new -nodes -key ca.key -sha256 -days 3650 -out ca.crt -subj '/CN=Test CA/C=AT/ST=UA/L=Linz/O=Dynatrace/OU=Operator' -extensions v3_ca

const customCA = `-----BEGIN CERTIFICATE-----
MIIDpTCCAo2gAwIBAgIUTCZa2IYrncHLMV2nLNbXOcQqc/4wDQYJKoZIhvcNAQEL
BQAwYjEQMA4GA1UEAwwHVGVzdCBDQTELMAkGA1UEBhMCQVQxCzAJBgNVBAgMAlVB
MQ0wCwYDVQQHDARMaW56MRIwEAYDVQQKDAlEeW5hdHJhY2UxETAPBgNVBAsMCE9w
ZXJhdG9yMB4XDTI1MTEwNjEzMDQyMVoXDTM1MTEwNDEzMDQyMVowYjEQMA4GA1UE
AwwHVGVzdCBDQTELMAkGA1UEBhMCQVQxCzAJBgNVBAgMAlVBMQ0wCwYDVQQHDARM
aW56MRIwEAYDVQQKDAlEeW5hdHJhY2UxETAPBgNVBAsMCE9wZXJhdG9yMIIBIjAN
BgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEArw564P5tXzT2uo0uRJdjhe+zGyU4
1zWdp6sIFB3J3KWKaAQ9ao7oMu75+pFo11c1XFuZcZpRmucWZ1AMWNm6Mga4yn6y
OcC+cIpDMT1kXnix+u7TH+XwOXkIty0T7I5OyiVV5JEryrl3jTjXRf4YbHRVrc4w
vspbS4JIxx+Hv6u4/sRRSvBI89hQ8miGgtOwuokGBIxcOKf/lqe10Q9SMuK+mAmP
jFlNlnOteFwTRBLWFJlDFgE+jxAyP3FGUIwLNN6w+DKzb4cmjnBk8TK3CHxhJREl
cncQnIXAp4Sq6VfR6mLGGyGpt3OWnm0L/cPASed5gp3V1CUW0T3Iz21VVwIDAQAB
o1MwUTAdBgNVHQ4EFgQULiJWJ0CXf4aoFki24ef2gRH0EU8wHwYDVR0jBBgwFoAU
LiJWJ0CXf4aoFki24ef2gRH0EU8wDwYDVR0TAQH/BAUwAwEB/zANBgkqhkiG9w0B
AQsFAAOCAQEAnLgaLr2qpVM6heHaBHt+vDWNda9YkfUGCfGU64AZf5kT9fQWFaXi
Liv0TC1NBOTHJ35DjSc4O/EshfO/qW0eMnLw8u4gfhPKs7mmADkcy4V/rhyA/hTU
1Vx+MSJsKH2vJAODaELKZZ3AiA9Rfyyt6Nv+nUtHRtBLpRmrYnVLlZHgfMvfSmnk
zWDF6rXZJXT6MJUcf740v4MOLlIWcrNj/igI9VQP9cBrhvJzthHJ0gMEjNqKJPgk
APj12zaRa05OBW3H3Ng+1MmdtrU4gAu+xwLAOz1cxT6q8LUGBGDCBYVcFXvomhKL
kHUfKUp2W9zOWWDlwSB65QuJ3wAQSCVs4g==
-----END CERTIFICATE-----`
