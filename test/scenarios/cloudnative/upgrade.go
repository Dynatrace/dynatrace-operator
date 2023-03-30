//go:build e2e

package cloudnative

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/namespace"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/sampleapps"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/steps/assess"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/steps/teardown"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/tenant"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

func Upgrade(t *testing.T) features.Feature {
	builder := features.New("upgrade_installation")
	secretConfig := tenant.GetSingleTenantSecret(t)
	dynakubeBuilder := dynakube.NewBuilder().
		WithDefaultObjectMeta().
		ApiUrl(secretConfig.ApiUrl).
		CloudNative(defaultCloudNativeSpec())
	testDynakube := dynakubeBuilder.Build()

	sampleNamespace := namespace.NewBuilder("upgrade-sample").Build()
	sampleApp := sampleapps.NewSampleDeployment(t, testDynakube)
	sampleApp.WithNamespace(sampleNamespace)

	assess.InstallOperatorFromRelease(builder, testDynakube, "v0.9.1")
	builder.Assess("install sample app", sampleApp.Install())

	assess.InstallDynakube(builder, &secretConfig, testDynakube)
	assessSampleAppsRestart(builder, sampleApp)
	assessSampleInitContainers(builder, sampleApp)
	// update to snapshot
	assess.UpgradeOperatorFromSource(builder, testDynakube)
	assessSampleAppsRestartHalf(builder, sampleApp)
	assessSampleInitContainers(builder, sampleApp)
	builder.Teardown(sampleApp.UninstallNamespace())
	teardown.UninstallDynatrace(builder, testDynakube)

	return builder.Feature()
}
