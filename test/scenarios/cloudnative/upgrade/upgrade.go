//go:build e2e

package cloudnative

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/namespace"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/sampleapps"
	sample "github.com/Dynatrace/dynatrace-operator/test/helpers/sampleapps/base"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/steps/assess"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/steps/teardown"
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
	builder.Assess("create sample namespace", namespace.Create(sampleNamespace))
	sampleApp := sampleapps.NewSampleDeployment(t, testDynakube)
	sampleApp.WithNamespace(sampleNamespace)

	assess.InstallOperatorFromRelease(builder, testDynakube, "v0.9.1")

	// Register dynakube install
	assess.InstallDynakube(builder, &secretConfig, testDynakube)

	// Register sample app install
	builder.Assess("install sample app", sampleApp.Install())

	// Register actual test
	cloudnative.AssessSampleInitContainers(builder, sampleApp)
	// update to snapshot
	assess.UpgradeOperatorFromSource(builder, testDynakube)
	assessSampleAppsRestartHalf(builder, sampleApp)
	cloudnative.AssessSampleInitContainers(builder, sampleApp)
	builder.Teardown(sampleApp.UninstallNamespace())
	teardown.UninstallDynatrace(builder, testDynakube)

	return builder.Feature()
}

func assessSampleAppsRestartHalf(builder *features.FeatureBuilder, sampleApp sample.App) {
	builder.Assess("restart half of sample apps", sampleApp.RestartHalf)
}
