//go:build e2e

package switch_modes

import (
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook"
	"github.com/Dynatrace/dynatrace-operator/test/features/cloudnative"
	"github.com/Dynatrace/dynatrace-operator/test/helpers"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/oneagent"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/sample"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/tenant"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

const (
	sampleAppsClassicName     = "sample-apps-classic"
	sampleAppsCloudNativeName = "sample-apps-cloud-native"
)

func Feature(t *testing.T) features.Feature {
	builder := features.New("switch from classic to cloudnative")
	builder.WithLabel("name", "classic-to-cloudnative")

	// build classic full stack dynakube
	secretConfig := tenant.GetSingleTenantSecret(t)

	commonOptions := []dynakube.Option{
		dynakube.WithApiUrl(secretConfig.ApiUrl),
	}

	dynakubeClassicFullStack := *dynakube.New(
		append(commonOptions, dynakube.WithClassicFullstackSpec(&dynatracev1beta1.HostInjectSpec{}))...,
	)

	sampleAppClassic := sample.NewApp(t, &dynakubeClassicFullStack,
		sample.AsDeployment(),
		sample.WithName(sampleAppsClassicName),
	)
	dynakube.Install(builder, helpers.LevelAssess, &secretConfig, dynakubeClassicFullStack)
	builder.Assess("install sample app", sampleAppClassic.Install())

	// change dynakube to cloud native
	dynakubeCloudNative := *dynakube.New(
		append(commonOptions, dynakube.WithCloudNativeSpec(cloudnative.DefaultCloudNativeSpec()))...,
	)

	dynakube.Delete(builder, helpers.LevelAssess, dynakubeClassicFullStack)
	oneagent.RunClassicUninstall(builder, helpers.LevelAssess, dynakubeClassicFullStack)
	sampleAppCloudNative := sample.NewApp(t, &dynakubeCloudNative,
		sample.AsDeployment(),
		sample.WithName(sampleAppsCloudNativeName),
		sample.WithAnnotations(map[string]string{dtwebhook.AnnotationFailurePolicy: "fail"}),
	)
	builder.Assess("create sample app namespace", sampleAppCloudNative.InstallNamespace())

	dynakube.Install(builder, helpers.LevelAssess, &secretConfig, dynakubeCloudNative)

	// apply sample apps
	builder.Assess("install sample app", sampleAppCloudNative.Install())

	// run cloud native test here
	cloudnative.AssessSampleInitContainers(builder, sampleAppCloudNative)

	// teardown
	builder.Teardown(sampleAppCloudNative.Uninstall())
	builder.Teardown(sampleAppClassic.Uninstall())
	dynakube.Delete(builder, helpers.LevelTeardown, dynakubeClassicFullStack)
	return builder.Feature()
}
