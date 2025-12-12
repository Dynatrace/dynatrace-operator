//go:build e2e

package switchmodes

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/oneagent"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/features/cloudnative"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers"
	dynakubeComponents "github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/components/dynakube"
	oneagenthelper "github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/components/oneagent"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/sample"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/tenant"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

const (
	sampleAppsClassicName     = "sample-apps-classic"
	sampleAppsCloudNativeName = "sample-apps-cloud-native"
)

func Feature(t *testing.T) features.Feature {
	builder := features.New("classic-to-cloudnative")

	// build classic full stack dynakube
	secretConfig := tenant.GetSingleTenantSecret(t)

	commonOptions := []dynakubeComponents.Option{
		dynakubeComponents.WithAPIURL(secretConfig.APIURL),
	}

	dynakubeClassicFullStack := *dynakubeComponents.New(
		append(commonOptions, dynakubeComponents.WithClassicFullstackSpec(&oneagent.HostInjectSpec{}))...,
	)

	sampleAppClassic := sample.NewApp(t, &dynakubeClassicFullStack,
		sample.AsDeployment(),
		sample.WithName(sampleAppsClassicName),
	)
	dynakubeComponents.Install(builder, helpers.LevelAssess, &secretConfig, dynakubeClassicFullStack)
	builder.Assess("install sample app", sampleAppClassic.Install())

	// change dynakube to cloud native
	dynakubeCloudNative := *dynakubeComponents.New(
		append(commonOptions, dynakubeComponents.WithCloudNativeSpec(cloudnative.DefaultCloudNativeSpec()))...,
	)

	dynakubeComponents.Delete(builder, helpers.LevelAssess, dynakubeClassicFullStack)
	oneagenthelper.RunClassicUninstall(builder, helpers.LevelAssess, dynakubeClassicFullStack)
	sampleAppCloudNative := sample.NewApp(t, &dynakubeCloudNative,
		sample.AsDeployment(),
		sample.WithName(sampleAppsCloudNativeName),
	)
	builder.Assess("create sample app namespace", sampleAppCloudNative.InstallNamespace())

	dynakubeComponents.Install(builder, helpers.LevelAssess, &secretConfig, dynakubeCloudNative)

	// apply sample apps
	builder.Assess("install sample app", sampleAppCloudNative.Install())

	// run cloud native test here
	cloudnative.AssessSampleInitContainers(builder, sampleAppCloudNative)

	// teardown
	builder.Teardown(sampleAppCloudNative.Uninstall())
	builder.Teardown(sampleAppClassic.Uninstall())
	dynakubeComponents.Delete(builder, helpers.LevelTeardown, dynakubeClassicFullStack)

	return builder.Feature()
}
