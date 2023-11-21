//go:build e2e

package edgeconnect

import (
	"fmt"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/test/helpers"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/edgeconnect"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/tenant"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

func Feature(t *testing.T) features.Feature {
	builder := features.New("install edgeconnect")
	builder.WithLabel("name", "edgeconnect-install")

	secretConfig := tenant.GetEdgeConnectTenantSecret(t)

	testEdgeConnect := *edgeconnect.New(
		// this name should match with tenant edge connect name
		edgeconnect.WithName(secretConfig.Name),
		edgeconnect.WithApiServer(secretConfig.ApiServer),
		edgeconnect.WithOAuthClientSecret(fmt.Sprintf("%s-client-secret", secretConfig.Name)),
		edgeconnect.WithOAuthEndpoint("https://sso-dev.dynatracelabs.com/sso/oauth2/token"),
		edgeconnect.WithOAuthResource(fmt.Sprintf("urn:dtenvironment:%s", secretConfig.TenantUid)),
	)

	// Register operator install
	edgeconnect.Install(builder, helpers.LevelAssess, &secretConfig, testEdgeConnect)
	builder.Teardown(edgeconnect.Delete(testEdgeConnect))
	return builder.Feature()
}
