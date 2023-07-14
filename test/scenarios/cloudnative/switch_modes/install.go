//go:build e2e

package switch_modes

import (
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/sampleapps"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/steps/assess"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/steps/teardown"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/tenant"
	"github.com/Dynatrace/dynatrace-operator/test/scenarios/cloudnative"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

const (
	sampleAppsClassicName     = "sample-apps-classic"
	sampleAppsCloudNativeName = "sample-apps-cloud-native"
)

func Install(t *testing.T, name string) features.Feature {
	featureBuilder := features.New(name)

	// build cloud native full stack dynakube
	secretConfig := tenant.GetSingleTenantSecret(t)
	cloudNativeDynakubeBuilder := dynakube.NewBuilder().
		WithDefaultObjectMeta().
		ApiUrl(secretConfig.ApiUrl).
		CloudNative(&dynatracev1beta1.CloudNativeFullStackSpec{})

	// install operator and dynakube
	assess.InstallDynatrace(featureBuilder, &secretConfig, cloudNativeDynakubeBuilder.Build())

	// apply sample apps
	sampleAppCloudNative := sampleapps.NewSampleDeployment(t, cloudNativeDynakubeBuilder.Build())
	sampleAppCloudNative.WithName(sampleAppsCloudNativeName)
	featureBuilder.Assess("install sample app", sampleAppCloudNative.Install())

	// run cloud native test here
	cloudnative.AssessSampleInitContainers(featureBuilder, sampleAppCloudNative)

	// switch to classic full stack
	classicDynakubeBuilder := cloudNativeDynakubeBuilder.ResetOneAgent().ClassicFullstack(&dynatracev1beta1.HostInjectSpec{})
	assess.UpdateDynakube(featureBuilder, classicDynakubeBuilder.Build())

	// deploy sample apps
	sampleAppClassicFullStack := sampleapps.NewSampleDeployment(t, classicDynakubeBuilder.Build())
	sampleAppClassicFullStack.WithName(sampleAppsClassicName)
	featureBuilder.Assess("install sample app", sampleAppClassicFullStack.Install())

	// tear down
	featureBuilder.Teardown(sampleAppCloudNative.Uninstall())
	featureBuilder.Teardown(sampleAppClassicFullStack.Uninstall())
	teardown.UninstallDynatrace(featureBuilder, cloudNativeDynakubeBuilder.Build())

	return featureBuilder.Feature()
}
