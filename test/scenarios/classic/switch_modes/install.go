//go:build e2e

package switch_modes

import (
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1/dynakube"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/src/webhook"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/oneagent"
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

	// build classic full stack dynakube
	secretConfig := tenant.GetSingleTenantSecret(t)
	dynakubeBuilder := dynakube.NewBuilder().
		WithDefaultObjectMeta().
		ApiUrl(secretConfig.ApiUrl).
		ClassicFullstack(&dynatracev1beta1.HostInjectSpec{})
	dynakubeClassicFullstack := dynakubeBuilder.Build()

	// install operator and dynakube
	assess.InstallDynatrace(featureBuilder, &secretConfig, dynakubeClassicFullstack)

	sampleAppClassic := sampleapps.NewSampleDeployment(t, dynakubeClassicFullstack)
	sampleAppClassic.WithName(sampleAppsClassicName)

	featureBuilder.Assess("install sample app", sampleAppClassic.Install())

	// change dynakube to cloud native
	dynakubeBuilder = dynakubeBuilder.ResetOneAgent().CloudNative(cloudnative.DefaultCloudNativeSpec())
	dynakubeCloudNative := dynakubeBuilder.Build()

	assess.InstallOperatorFromSource(featureBuilder, dynakubeCloudNative)
	assess.UpdateDynakube(featureBuilder, dynakubeCloudNative)

	// wait for oneagent daemonset to be ready
	featureBuilder.Assess("wait for dynakube to be reconciled", dynakube.WaitForDynakubePhase(dynakubeCloudNative, dynatracev1beta1.Deploying))
	featureBuilder.Assess("wait for daemonset to be ready", oneagent.WaitForDaemonset(dynakubeCloudNative))

	// apply sample apps
	sampleAppCloudNative := sampleapps.NewSampleDeployment(t, dynakubeCloudNative)
	sampleAppCloudNative.WithName(sampleAppsCloudNativeName)
	sampleAppCloudNative.WithAnnotations(map[string]string{dtwebhook.AnnotationFailurePolicy: "fail"})
	featureBuilder.Assess("install sample app", sampleAppCloudNative.Install())

	// run cloud native test here
	cloudnative.AssessSampleInitContainers(featureBuilder, sampleAppCloudNative)

	// teardown
	featureBuilder.Teardown(sampleAppCloudNative.Uninstall())
	featureBuilder.Teardown(sampleAppClassic.Uninstall())
	teardown.UninstallDynatrace(featureBuilder, dynakubeCloudNative)

	return featureBuilder.Feature()
}
