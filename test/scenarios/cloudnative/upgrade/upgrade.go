//go:build e2e

package upgrade

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/namespace"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/sampleapps"
	sample "github.com/Dynatrace/dynatrace-operator/test/helpers/sampleapps/base"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/setup"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/tenant"
	"github.com/Dynatrace/dynatrace-operator/test/scenarios/cloudnative"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

func upgrade(t *testing.T) features.Feature {
	builder := features.New("upgrade_installation")
	secretConfig := tenant.GetSingleTenantSecret(t)
	dynakubeBuilder := dynakube.NewBuilder().
		WithDefaultObjectMeta().
		ApiUrl(secretConfig.ApiUrl).
		CloudNative(cloudnative.DefaultCloudNativeSpec())
	testDynakube := dynakubeBuilder.Build()

	sampleNamespace := namespace.NewBuilder("upgrade-sample").Build()
	sampleApp := sampleapps.NewSampleDeployment(t, testDynakube)
	sampleApp.WithNamespace(sampleNamespace)
	builder.Assess("create sample namespace", sampleApp.InstallNamespace())

	steps := setup.NewEnvironmentSetup(
		setup.CreateDefaultDynatraceNamespace(),
		setup.DeployOperatorViaHelm("v0.10.4", true),
		setup.CreateDynakube(secretConfig, testDynakube))
	steps.CreateSetupSteps(builder)

	// Register sample app install
	builder.Assess("install sample app", sampleApp.Install())

	// Register actual test
	cloudnative.AssessSampleInitContainers(builder, sampleApp)
	// update to snapshot
	setup.DeployOperatorViaMake(testDynakube.NeedsCSIDriver()).AddSetupSetup(builder)
	assessSampleAppsRestartHalf(builder, sampleApp)
	cloudnative.AssessSampleInitContainers(builder, sampleApp)
	builder.Teardown(sampleApp.UninstallNamespace())
	steps.CreateTeardownSteps(builder)
	return builder.Feature()
}

func assessSampleAppsRestartHalf(builder *features.FeatureBuilder, sampleApp sample.App) {
	builder.Assess("restart half of sample apps", sampleApp.RestartHalf)
}
