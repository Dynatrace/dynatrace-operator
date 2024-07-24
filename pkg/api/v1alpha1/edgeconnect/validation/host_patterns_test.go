package validation

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha1/edgeconnect"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestHostPatternsRequired(t *testing.T) {
	t.Run(`hostPatters optional - no error when provisioner false`, func(t *testing.T) {
		ec := &edgeconnect.EdgeConnect{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testName,
				Namespace: testNamespace,
			},
			Spec: edgeconnect.EdgeConnectSpec{
				ApiServer: "tenantid-test.dev.apps.dynatracelabs.com",
				OAuth: edgeconnect.OAuthSpec{
					ClientSecret: "secret",
					Endpoint:     "endpoint",
					Resource:     "resource",
				},
				ServiceAccountName: testServiceAccountName,
			},
		}
		response := handleRequest(t, ec, prepareTestServiceAccount(testServiceAccountName, testNamespace))
		assert.True(t, response.Allowed)
		assert.Empty(t, response.Warnings)
	})

	t.Run(`hostPatters is required - error when provisioner true`, func(t *testing.T) {
		ec := &edgeconnect.EdgeConnect{
			Spec: edgeconnect.EdgeConnectSpec{
				ApiServer: "tenantid-test.dev.apps.dynatracelabs.com",
				OAuth: edgeconnect.OAuthSpec{
					ClientSecret: "secret",
					Endpoint:     "endpoint",
					Resource:     "resource",
					Provisioner:  true,
				},
			},
		}
		assertDeniedResponse(t, []string{errorHostPattersIsRequired}, ec)
	})
}
