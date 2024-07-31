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
	t.Run(`happy apiServer`, func(t *testing.T) {
		for _, suffix := range allowedSuffix {
			ec := &edgeconnect.EdgeConnect{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testName,
					Namespace: testNamespace,
				},
				Spec: edgeconnect.EdgeConnectSpec{
					ApiServer: "tenantid" + suffix,
					OAuth: edgeconnect.OAuthSpec{
						ClientSecret: "secret",
						Endpoint:     "endpoint",
						Resource:     "resource",
					},
					ServiceAccountName: testServiceAccountName,
				},
			}
			assertAllowed(t, ec, prepareTestServiceAccount(testServiceAccountName, testNamespace))
		}
	})

	t.Run(`invalid apiServer (missing tenant)`, func(t *testing.T) {
		assertDenied(t, []string{errorInvalidApiServer}, &edgeconnect.EdgeConnect{
			Spec: edgeconnect.EdgeConnectSpec{
				ApiServer: allowedSuffix[0],
				OAuth: edgeconnect.OAuthSpec{
					ClientSecret: "secret",
					Endpoint:     "endpoint",
					Resource:     "resource",
				},
			},
		})
	})

	t.Run(`invalid apiServer (wrong suffix)`, func(t *testing.T) {
		assertDenied(t, []string{errorInvalidApiServer}, &edgeconnect.EdgeConnect{
			Spec: edgeconnect.EdgeConnectSpec{
				ApiServer: "doma.in",
				OAuth: edgeconnect.OAuthSpec{
					ClientSecret: "secret",
					Endpoint:     "endpoint",
					Resource:     "resource",
				},
			},
		})
	})
}
