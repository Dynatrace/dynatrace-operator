//go:build e2e

package switch_modes

import (
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/features/cloudnative"
	"github.com/Dynatrace/dynatrace-operator/test/helpers"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/sample"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/tenant"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

const (
	sampleAppsClassicName     = "sample-apps-classic"
	sampleAppsCloudNativeName = "sample-apps-cloud-native"
)

func Feature(t *testing.T) features.Feature {
	builder := features.New("switch from cloudnative to classic")
	builder.WithLabel("name", "cloudnative-to-classic")

	// build cloud native full stack dynakube
	secretConfig := tenant.GetSingleTenantSecret(t)
	commonOptions := []dynakube.Option{
		dynakube.WithApiUrl(secretConfig.ApiUrl),
	}
	dynakubeCloudNative := *dynakube.New(append(commonOptions, dynakube.WithCloudNativeSpec(&dynatracev1beta1.CloudNativeFullStackSpec{}))...)
	sampleAppCloudNative := sample.NewApp(t, &dynakubeCloudNative,
		sample.AsDeployment(),
		sample.WithName(sampleAppsCloudNativeName),
	)
	builder.Assess("(cloudnative) create sample app namespace", sampleAppCloudNative.InstallNamespace())
	builder.Teardown(sampleAppCloudNative.Uninstall())

	// install operator and dynakube
	dynakube.Install(builder, helpers.LevelAssess, &secretConfig, dynakubeCloudNative)

	// apply sample apps
	builder.Assess("(cloudnative) install sample app", sampleAppCloudNative.Install())

	// run cloud native test here
	cloudnative.AssessSampleInitContainers(builder, sampleAppCloudNative)

	// switch to classic full stack
	dynakubeClassicFullStack := *dynakube.New(append(commonOptions, dynakube.WithClassicFullstackSpec(&dynatracev1beta1.HostInjectSpec{}))...)
	sampleAppClassicFullStack := sample.NewApp(t, &dynakubeClassicFullStack,
		sample.AsDeployment(),
		sample.WithName(sampleAppsClassicName),
	)

	dynakube.Update(builder, helpers.LevelAssess, dynakubeClassicFullStack)

	// deploy sample apps
	builder.Assess("(classic) install sample app", sampleAppClassicFullStack.Install())
	builder.Teardown(sampleAppClassicFullStack.Uninstall())
	// tear down
	dynakube.Delete(builder, helpers.LevelTeardown, dynakubeCloudNative)
	return builder.Feature()
}
