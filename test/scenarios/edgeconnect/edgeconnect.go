//go:build e2e

package edgeconnect

import (
	"fmt"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/edgeconnect"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/namespace"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/setup"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/tenant"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

func install(t *testing.T) features.Feature {
	builder := features.New("install edgeconnect")

	secretConfig := tenant.GetEdgeConnectTenantSecret(t)

	testEdgeConnect := edgeconnect.NewBuilder().
		// this name should match with tenant edge connect name
		Name(secretConfig.Name).
		ApiServer(secretConfig.ApiServer).
		OAuthClientSecret(fmt.Sprintf("%s-client-secret", secretConfig.Name)).
		OAuthEndpoint("https://sso-dev.dynatracelabs.com/sso/oauth2/token").
		OAuthResource(fmt.Sprintf("urn:dtenvironment:%s", secretConfig.TenantUid)).
		CustomPullSecret(fmt.Sprintf("%s-docker-pull-secret", secretConfig.Name)).
		Build()

	// Register operator install
	setup.CreateFeatureEnvironment(builder,
		setup.CreateNamespaceWithoutTeardown(namespace.NewBuilder(testEdgeConnect.Namespace).Build()),
		setup.DeployOperatorViaMake(false),
		setup.CreateEdgeConnect(secretConfig, testEdgeConnect),
	)
	return builder.Feature()
}
