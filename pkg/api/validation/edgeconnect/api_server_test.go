package validation

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha2/edgeconnect"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	testName      = "test-name"
	testNamespace = "test-namespace"
)

func TestApiServer(t *testing.T) {
	t.Run("happy apiServer", func(t *testing.T) {
		for _, suffix := range allowedSuffix {
			ec := &edgeconnect.EdgeConnect{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testName,
					Namespace: testNamespace,
				},
				Spec: edgeconnect.EdgeConnectSpec{
					APIServer: "tenantid" + suffix,
					OAuth: edgeconnect.OAuthSpec{
						ClientSecret: "secret",
						Endpoint:     "endpoint",
						Resource:     "resource",
					},
				},
			}
			assertAllowed(t, ec, prepareTestServiceAccount(testServiceAccountName, testNamespace))
		}
	})

	t.Run("invalid apiServer (missing tenant)", func(t *testing.T) {
		assertDenied(t, []string{errorMissingAllowedSuffixAPIServer}, &edgeconnect.EdgeConnect{
			Spec: edgeconnect.EdgeConnectSpec{
				APIServer: allowedSuffix[0],
				OAuth: edgeconnect.OAuthSpec{
					ClientSecret: "secret",
					Endpoint:     "endpoint",
					Resource:     "resource",
				},
			},
		})
	})

	t.Run("invalid apiServer (wrong suffix)", func(t *testing.T) {
		assertDenied(t, []string{errorMissingAllowedSuffixAPIServer}, &edgeconnect.EdgeConnect{
			Spec: edgeconnect.EdgeConnectSpec{
				APIServer: "doma.in",
				OAuth: edgeconnect.OAuthSpec{
					ClientSecret: "secret",
					Endpoint:     "endpoint",
					Resource:     "resource",
				},
			},
		})
	})

	t.Run("invalid apiServer url should NOT be empty string", func(t *testing.T) {
		assertDenied(t, []string{errorMissingAllowedSuffixAPIServer}, &edgeconnect.EdgeConnect{
			Spec: edgeconnect.EdgeConnectSpec{
				APIServer: "",
				OAuth: edgeconnect.OAuthSpec{
					ClientSecret: "secret",
					Endpoint:     "endpoint",
					Resource:     "resource",
				},
			},
		})
	})

	t.Run("invalid apiServer url should not start with numbers", func(t *testing.T) {
		assertDenied(t, []string{errorInvalidAPIServer}, &edgeconnect.EdgeConnect{
			Spec: edgeconnect.EdgeConnectSpec{
				// adding not-allowed symbols to API server
				APIServer: string([]byte{0}) + allowedSuffix[0],
				OAuth: edgeconnect.OAuthSpec{
					ClientSecret: "secret",
					Endpoint:     "endpoint",
					Resource:     "resource",
				},
			},
		})
	})

	t.Run("invalid apiServer (includes http protocol)", func(t *testing.T) {
		assertDenied(t, []string{errorProtocolIsNotAllowedAPIServer}, &edgeconnect.EdgeConnect{
			Spec: edgeconnect.EdgeConnectSpec{
				APIServer: "http://doma.in",
				OAuth: edgeconnect.OAuthSpec{
					ClientSecret: "secret",
					Endpoint:     "endpoint",
					Resource:     "resource",
				},
			},
		})
	})

	t.Run("invalid apiServer (includes https protocol)", func(t *testing.T) {
		assertDenied(t, []string{errorProtocolIsNotAllowedAPIServer}, &edgeconnect.EdgeConnect{
			Spec: edgeconnect.EdgeConnectSpec{
				APIServer: "https://doma.in",
				OAuth: edgeconnect.OAuthSpec{
					ClientSecret: "secret",
					Endpoint:     "endpoint",
					Resource:     "resource",
				},
			},
		})
	})
}
