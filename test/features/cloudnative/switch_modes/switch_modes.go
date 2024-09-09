//go:build e2e

package switch_modes

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/features/cloudnative"
	"github.com/Dynatrace/dynatrace-operator/test/helpers"
	dynakubeComponents "github.com/Dynatrace/dynatrace-operator/test/helpers/components/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/sample"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/tenant"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

const (
	sampleAppsClassicName     = "sample-apps-classic"
	sampleAppsCloudNativeName = "sample-apps-cloud-native"
)

func Feature(t *testing.T) features.Feature {
	builder := features.New("cloudnative-to-classic")

	// build cloud native full stack dynakubeComponents
	secretConfig := tenant.GetSingleTenantSecret(t)
	commonOptions := []dynakubeComponents.Option{
		dynakubeComponents.WithApiUrl(secretConfig.ApiUrl),
	}
	dynakubeCloudNative := *dynakubeComponents.New(append(commonOptions, dynakubeComponents.WithCloudNativeSpec(&dynakube.CloudNativeFullStackSpec{}))...)
	sampleAppCloudNative := sample.NewApp(t, &dynakubeCloudNative,
		sample.AsDeployment(),
		sample.WithName(sampleAppsCloudNativeName),
	)
	builder.Assess("(cloudnative) create sample app namespace", sampleAppCloudNative.InstallNamespace())
	builder.Teardown(sampleAppCloudNative.Uninstall())

	// install operator and dynakubeComponents
	dynakubeComponents.Install(builder, helpers.LevelAssess, &secretConfig, dynakubeCloudNative)

	// apply sample apps
	builder.Assess("(cloudnative) install sample app", sampleAppCloudNative.Install())

	// run cloud native test here
	cloudnative.AssessSampleInitContainers(builder, sampleAppCloudNative)

	// switch to classic full stack
	dynakubeClassicFullStack := *dynakubeComponents.New(append(commonOptions, dynakubeComponents.WithClassicFullstackSpec(&dynakube.HostInjectSpec{}))...)
	sampleAppClassicFullStack := sample.NewApp(t, &dynakubeClassicFullStack,
		sample.AsDeployment(),
		sample.WithName(sampleAppsClassicName),
	)

	dynakubeComponents.Update(builder, helpers.LevelAssess, dynakubeClassicFullStack)

	// deploy sample apps
	builder.Assess("(classic) install sample app", sampleAppClassicFullStack.Install())
	builder.Teardown(sampleAppClassicFullStack.Uninstall())
	// tear down
	dynakubeComponents.Delete(builder, helpers.LevelTeardown, dynakubeCloudNative)

	return builder.Feature()
}
