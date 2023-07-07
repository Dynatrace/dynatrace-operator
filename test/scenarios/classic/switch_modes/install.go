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

func Install(t *testing.T, name string) features.Feature {
	featureBuilder := features.New(name)

	// build classic full stack dynakube
	secretConfig := tenant.GetSingleTenantSecret(t)
	dynakubeBuilder := dynakube.NewBuilder().
		WithDefaultObjectMeta().
		ApiUrl(secretConfig.ApiUrl).
		ClassicFullstack(&dynatracev1beta1.HostInjectSpec{})

	// install operator and dynakube
	assess.InstallDynatrace(featureBuilder, &secretConfig, dynakubeBuilder.Build())

	sampleAppClassic := sampleapps.NewSampleDeployment(t, dynakubeBuilder.Build())
	sampleAppClassic.WithName("sample-apps-classic")

	featureBuilder.Assess("install sample app", sampleAppClassic.Install())

	// change dynakube to cloud native
	dynakubeBuilder = dynakubeBuilder.ResetOneAgent().CloudNative(cloudnative.DefaultCloudNativeSpec())

	assess.InstallOperatorFromSource(featureBuilder, dynakubeBuilder.Build())
	assess.UpdateDynakube(featureBuilder, dynakubeBuilder.Build())

	featureBuilder.Assess("oneagent status instances are ready", dynakube.WaitForOneAgentInstances(dynakubeBuilder.Build()))

	// apply sample apps
	sampleAppCloudNative := sampleapps.NewSampleDeployment(t, dynakubeBuilder.Build())
	featureBuilder.Assess("install sample app", sampleAppCloudNative.Install())

	// run cloud native test here
	cloudnative.AssessSampleInitContainers(featureBuilder, sampleAppCloudNative)

	// teardown
	featureBuilder.Teardown(sampleAppCloudNative.Uninstall())
	featureBuilder.Teardown(sampleAppClassic.Uninstall())
	teardown.UninstallDynatrace(featureBuilder, dynakubeBuilder.Build())

	return featureBuilder.Feature()
}
