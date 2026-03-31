package edgeconnect

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClient(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		client, err := NewClient(
			WithContext(t.Context()),
			WithClientID("dummy_client_id"),
			WithClientSecret("dummy_client_secret"),
			WithOAuthScopes([]string{"test"}),
			WithBaseURL("http://test.com"),
			WithTokenURL("http://test.com/token"),
			WithCustomCA([]byte(customCA)),
		)
		require.NoError(t, err)
		assert.NotNil(t, client)
		assert.Implements(t, (*APIClient)(nil), client)
	})

	t.Run("invalid cert", func(t *testing.T) {
		client, err := NewClient(
			WithContext(t.Context()),
			WithClientID("dummy_client_id"),
			WithClientSecret("dummy_client_secret"),
			WithOAuthScopes([]string{"test"}),
			WithBaseURL("http://test.com"),
			WithTokenURL("http://test.com/token"),
			WithCustomCA([]byte("invalid")),
		)
		require.Error(t, err)
		assert.Nil(t, client)
	})

	t.Run("invalid base URL", func(t *testing.T) {
		client, err := NewClient(
			WithContext(t.Context()),
			WithBaseURL("://invalid"),
		)
		require.Error(t, err)
		assert.Nil(t, client)
	})
}

func TestNewClientFromAPIClient(t *testing.T) {
	t.Run("create client from core client", func(t *testing.T) {
		client := NewClientFromAPIClient(nil)
		assert.NotNil(t, client)
		assert.Implements(t, (*APIClient)(nil), client)
	})
}

// // Generated with:
// // openssl genrsa -out ca.key 2048
// // openssl req -x509 -new -nodes -key ca.key -sha256 -days 3650 -out ca.crt -subj '/CN=Test CA/C=AT/ST=UA/L=Linz/O=Dynatrace/OU=Operator' -extensions v3_ca
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
