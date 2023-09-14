//go:build e2e

package switch_modes

import (
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/namespace"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/sampleapps"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/setup"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/steps/assess"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/tenant"
	"github.com/Dynatrace/dynatrace-operator/test/scenarios/cloudnative"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

const (
	sampleAppsClassicName     = "sample-apps-classic"
	sampleAppsCloudNativeName = "sample-apps-cloud-native"
)

func switchModes(t *testing.T, name string) features.Feature {
	featureBuilder := features.New(name)

	// build cloud native full stack dynakube
	secretConfig := tenant.GetSingleTenantSecret(t)
	cloudNativeDynakubeBuilder := dynakube.NewBuilder().
		WithDefaultObjectMeta().
		ApiUrl(secretConfig.ApiUrl).
		CloudNative(&dynatracev1beta1.CloudNativeFullStackSpec{})
	dynakubeCloudNative := cloudNativeDynakubeBuilder.Build()
	sampleAppCloudNative := sampleapps.NewSampleDeployment(t, dynakubeCloudNative)
	sampleAppCloudNative.WithName(sampleAppsCloudNativeName)
	featureBuilder.Assess("(cloudnative) create sample app namespace", sampleAppCloudNative.InstallNamespace())
	featureBuilder.Teardown(sampleAppCloudNative.Uninstall())

	// install operator and dynakube
	steps := setup.NewEnvironmentSetup(
		setup.CreateNamespaceWithoutTeardown(namespace.NewBuilder(dynakubeCloudNative.Namespace).Build()),
		setup.DeployOperatorViaMake(dynakubeCloudNative.NeedsCSIDriver()),
		setup.CreateDynakube(secretConfig, dynakubeCloudNative),
	)
	steps.CreateSetupSteps(featureBuilder)
	// apply sample apps
	featureBuilder.Assess("(cloudnative) install sample app", sampleAppCloudNative.Install())

	// run cloud native test here
	cloudnative.AssessSampleInitContainers(featureBuilder, sampleAppCloudNative)

	// switch to classic full stack
	classicDynakubeBuilder := cloudNativeDynakubeBuilder.ResetOneAgent().ClassicFullstack(&dynatracev1beta1.HostInjectSpec{})
	dynakubeClassicFullStack := classicDynakubeBuilder.Build()
	sampleAppClassicFullStack := sampleapps.NewSampleDeployment(t, dynakubeClassicFullStack)
	sampleAppClassicFullStack.WithName(sampleAppsClassicName)
	featureBuilder.Assess("(classic) create sample app namespace", sampleAppClassicFullStack.InstallNamespace())
	assess.UpdateDynakube(featureBuilder, dynakubeClassicFullStack)

	// deploy sample apps
	featureBuilder.Assess("(classic) install sample app", sampleAppClassicFullStack.Install())
	featureBuilder.Teardown(sampleAppClassicFullStack.Uninstall())
	// tear down
	steps.CreateTeardownSteps(featureBuilder)
	return featureBuilder.Feature()
}
