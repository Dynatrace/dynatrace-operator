package edgeconnect

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha1/edgeconnect"
	"github.com/stretchr/testify/assert"
)

func TestHostPatternsRequired(t *testing.T) {
	t.Run(`hostPatters optional - no error when provisioner false`, func(t *testing.T) {
		edgeConnect := &edgeconnect.EdgeConnect{
			Spec: edgeconnect.EdgeConnectSpec{
				ApiServer: "tenantid-test.dev.apps.dynatracelabs.com",
				OAuth: edgeconnect.OAuthSpec{
					ClientSecret: "secret",
					Endpoint:     "endpoint",
					Resource:     "resource",
				},
			},
		}
		response := handleRequest(t, edgeConnect)
		assert.True(t, response.Allowed)
		assert.Equal(t, 0, len(response.Warnings))
	})

	t.Run(`hostPatters is required - error when provisioner true`, func(t *testing.T) {
		edgeConnect := &edgeconnect.EdgeConnect{
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
		assertDeniedResponse(t, []string{errorHostPattersIsRequired}, edgeConnect)
	})
}
