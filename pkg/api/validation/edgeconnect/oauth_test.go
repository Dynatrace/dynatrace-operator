package validation

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha2/edgeconnect"
)

const (
	testAPIServer          = "tenantid.apps.dynatrace.com"
	testValidOAuthEndpoint = "https://sso.dynatrace.com/endpoint"
)

func TestOauthEndpoint(t *testing.T) {
	t.Run("happy apiServer", func(t *testing.T) {
		for _, domain := range allowedSSODomains {
			ec := &edgeconnect.EdgeConnect{
				Spec: edgeconnect.EdgeConnectSpec{
					APIServer: testAPIServer,
					OAuth: edgeconnect.OAuthSpec{
						ClientSecret: "secret",
						Endpoint:     "https://" + domain + "/endpoint",
						Resource:     "resource",
					},
				},
			}
			assertAllowed(t, ec)
		}
	})

	t.Run("invalid ouath.endpoint (missing protocol)", func(t *testing.T) {
		for _, domain := range allowedSSODomains {
			ec := &edgeconnect.EdgeConnect{
				Spec: edgeconnect.EdgeConnectSpec{
					APIServer: testAPIServer,
					OAuth: edgeconnect.OAuthSpec{
						ClientSecret: "secret",
						Endpoint:     domain + "/endpoint",
						Resource:     "resource",
					},
				},
			}

			assertDenied(t, []string{errorProtocolIsMissingOauthEndpoint}, ec)
		}
	})

	t.Run("invalid oauth.endpoint (wrong protocol)", func(t *testing.T) {
		assertDenied(t, []string{errorProtocolIsMissingOauthEndpoint}, &edgeconnect.EdgeConnect{
			Spec: edgeconnect.EdgeConnectSpec{
				APIServer: testAPIServer,
				OAuth: edgeconnect.OAuthSpec{
					ClientSecret: "secret",
					Endpoint:     "http://sso.dynatrace.com/endpoint",
					Resource:     "resource",
				},
			},
		})
	})

	t.Run("invalid oauth.endpoint (wrong domain)", func(t *testing.T) {
		assertDenied(t, []string{errorUnknownSSOServer}, &edgeconnect.EdgeConnect{
			Spec: edgeconnect.EdgeConnectSpec{
				APIServer: testAPIServer,
				OAuth: edgeconnect.OAuthSpec{
					ClientSecret: "secret",
					Endpoint:     "https://my-sso-server/endpoint",
					Resource:     "resource",
				},
			},
		})
	})

	t.Run("invalid oauth.endpoint (url should NOT be empty string)", func(t *testing.T) {
		assertDenied(t, []string{errorProtocolIsMissingOauthEndpoint, errorUnknownSSOServer}, &edgeconnect.EdgeConnect{
			Spec: edgeconnect.EdgeConnectSpec{
				APIServer: testAPIServer,
				OAuth: edgeconnect.OAuthSpec{
					ClientSecret: "secret",
					Endpoint:     "",
					Resource:     "resource",
				},
			},
		})
	})

	t.Run("invalid oauth.endpoint (url.Parse fails)", func(t *testing.T) {
		assertDenied(t, []string{errorInvalidOauthEndpoint}, &edgeconnect.EdgeConnect{
			Spec: edgeconnect.EdgeConnectSpec{
				APIServer: testAPIServer,
				OAuth: edgeconnect.OAuthSpec{
					ClientSecret: "secret",
					Endpoint:     "https://sso-dev.dynatracelabs.com( )/sso/oauth2/token",
					Resource:     "resource",
				},
			},
		})
	})

	t.Run("invalid oauth.endpoint (protocol and domain)", func(t *testing.T) {
		assertDenied(t, []string{errorProtocolIsMissingOauthEndpoint, errorUnknownSSOServer}, &edgeconnect.EdgeConnect{
			Spec: edgeconnect.EdgeConnectSpec{
				APIServer: testAPIServer,
				OAuth: edgeconnect.OAuthSpec{
					ClientSecret: "secret",
					Endpoint:     "http://my-sso-dev.dynatracelabs.com/sso/oauth2/token",
					Resource:     "resource",
				},
			},
		})
	})
}
