//go:build e2e

package token

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/oneagent"
	e2econst "github.com/Dynatrace/dynatrace-operator/test/e2e/features/consts"
	componentDynakube "github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/components/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/tenant"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

func FromPlatformToAPIToken(t *testing.T) features.Feature {
	if !tenant.UsePlatformToken() {
		t.Skip("skip test from platform to api token if default is api token")
	}
	builder := features.New("migrate-from-platform-to-api-token")

	secretConfig := tenant.GetSingleTenantSecret(t)

	options := []componentDynakube.Option{
		componentDynakube.WithAPIURL(secretConfig.APIURL),
		componentDynakube.WithHostMonitoringSpec(&oneagent.HostInjectSpec{}),
	}

	if tenant.UsePhase3Tenant() {
		options = append(options, componentDynakube.WithUsePublicRegistryFF())
	}

	testDynakube := *componentDynakube.New(options...)

	componentDynakube.Install(builder, &secretConfig, testDynakube)

	builder.Assess("update tenant secret to api token",
		tenant.CreateTenantSecret(secretConfig.ClassicTokens(), testDynakube.Name, testDynakube.Namespace))
	// trigger manually to not wait 15 minutes until next reconcile
	componentDynakube.TriggerReconciliation(builder, testDynakube)
	componentDynakube.VerifyStartup(builder, features.LevelAssess, testDynakube)
	componentDynakube.VerifyPlatformTokenStatus(builder, testDynakube, false)

	return builder.Feature()
}

func FromAPIToPlatformToken(t *testing.T) features.Feature {
	if tenant.UsePlatformToken() {
		t.Skip("skip test from api to platform token if default is platform token")
	}

	builder := features.New("migrate-from-api-to-platform-token")

	secretConfig := tenant.GetSingleTenantSecret(t)

	testDynakube := *componentDynakube.New(
		componentDynakube.WithAPIURL(secretConfig.APIURL),
		componentDynakube.WithHostMonitoringSpec(&oneagent.HostInjectSpec{}),
		componentDynakube.WithCustomPullSecret(e2econst.DevRegistryPullSecretName),
	)

	componentDynakube.Install(builder, &secretConfig, testDynakube)

	builder.Assess("update tenant secret to platform token",
		tenant.CreateTenantSecret(secretConfig.PlatformTokens(), testDynakube.Name, testDynakube.Namespace))
	// trigger manually to not wait 15 minutes until next reconcile
	componentDynakube.TriggerReconciliation(builder, testDynakube)
	componentDynakube.VerifyStartup(builder, features.LevelAssess, testDynakube)
	componentDynakube.VerifyPlatformTokenStatus(builder, testDynakube, true)

	return builder.Feature()
}
