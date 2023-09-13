//go:build e2e

package switch_modes

import (
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1/dynakube"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/src/webhook"
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

func SwitchModes(t *testing.T, name string) features.Feature {
	featureBuilder := features.New(name)

	// build classic full stack dynakube
	secretConfig := tenant.GetSingleTenantSecret(t)
	dynakubeBuilder := dynakube.NewBuilder().
		WithDefaultObjectMeta().
		ApiUrl(secretConfig.ApiUrl).
		ClassicFullstack(&dynatracev1beta1.HostInjectSpec{})
	dynakubeClassicFullStack := dynakubeBuilder.Build()

	sampleAppClassic := sampleapps.NewSampleDeployment(t, dynakubeClassicFullStack)
	sampleAppClassic.WithName(sampleAppsClassicName)
	featureBuilder.Assess("create sample app namespace", sampleAppClassic.InstallNamespace())

	// install operator and dynakube
	setup := setup.NewEnvironmentSetup(
		setup.CreateNamespaceWithoutTeardown(namespace.NewBuilder(dynakubeClassicFullStack.Namespace).Build()),
		setup.DeployOperatorViaMake(dynakubeClassicFullStack.NeedsCSIDriver()),
		setup.CreateDynakube(secretConfig, dynakubeClassicFullStack),
	)
	setup.CreateSetupSteps(featureBuilder)
	featureBuilder.Assess("install sample app", sampleAppClassic.Install())

	// change dynakube to cloud native
	dynakubeBuilder = dynakubeBuilder.ResetOneAgent().CloudNative(cloudnative.DefaultCloudNativeSpec())
	dynakubeCloudNative := dynakubeBuilder.Build()

	assess.DeleteDynakube(featureBuilder, dynakubeClassicFullStack)
	assess.AddClassicCleanUp(featureBuilder, dynakubeClassicFullStack)
	sampleAppCloudNative := sampleapps.NewSampleDeployment(t, dynakubeCloudNative)
	sampleAppCloudNative.WithName(sampleAppsCloudNativeName)
	sampleAppCloudNative.WithAnnotations(map[string]string{dtwebhook.AnnotationFailurePolicy: "fail"})
	featureBuilder.Assess("create sample app namespace", sampleAppCloudNative.InstallNamespace())

	assess.InstallDynakube(featureBuilder, &secretConfig, dynakubeCloudNative)

	// apply sample apps
	featureBuilder.Assess("install sample app", sampleAppCloudNative.Install())

	// run cloud native test here
	cloudnative.AssessSampleInitContainers(featureBuilder, sampleAppCloudNative)

	// teardown
	featureBuilder.Teardown(sampleAppCloudNative.Uninstall())
	featureBuilder.Teardown(sampleAppClassic.Uninstall())
	setup.CreateTeardownSteps(featureBuilder)

	return featureBuilder.Feature()
}
