//go:build e2e

package disabled_auto_injection

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/exp"
	"github.com/Dynatrace/dynatrace-operator/test/features/cloudnative"
	"github.com/Dynatrace/dynatrace-operator/test/helpers"
	dynakubeComponents "github.com/Dynatrace/dynatrace-operator/test/helpers/components/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/namespace"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/sample"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/tenant"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

func FeatureEnabledPod(t *testing.T) features.Feature {
	builder := features.New("cloudnative-disabled-auto-inject-enabled-pod")

	secretConfig := tenant.GetSingleTenantSecret(t)
	testDynakube := *dynakubeComponents.New(
		dynakubeComponents.WithAnnotations(map[string]string{
			exp.InjectionAutomaticKey: "false",
		}),
		dynakubeComponents.WithApiUrl(secretConfig.ApiUrl),
		dynakubeComponents.WithCloudNativeSpec(cloudnative.DefaultCloudNativeSpec()),
	)

	// Register sample app install
	sampleNamespace := *namespace.New("cloudnative-disabled-injection-sample")
	sampleApp := sample.NewApp(t, &testDynakube,
		sample.AsDeployment(),
		sample.WithNamespace(sampleNamespace),
	)
	builder.Assess("create sample namespace", sampleApp.InstallNamespace())

	// Register dynakube install
	dynakubeComponents.Install(builder, helpers.LevelAssess, &secretConfig, testDynakube)

	// Register sample app install
	builder.Assess("install sample app", sampleApp.Install())

	sampleAppEnabled := sample.NewApp(t, &testDynakube,
		sample.WithName("sample-app-enabled"),
		sample.AsDeployment(),
		sample.WithAnnotations(map[string]string{"dynatrace.com/inject": "true"}),
		sample.WithNamespace(sampleNamespace),
	)

	builder.Assess("install sample app with enabled pod", sampleAppEnabled.Install())

	// Register actual test
	assessSampleInitContainersDisabled(builder, sampleApp)
	cloudnative.AssessSampleInitContainers(builder, sampleAppEnabled)

	// Register sample, dynakubeComponents and operator uninstall
	builder.Teardown(sampleApp.Uninstall())
	builder.Teardown(sampleAppEnabled.Uninstall())
	builder.Teardown(namespace.Delete(sampleNamespace.Name))
	dynakubeComponents.Delete(builder, helpers.LevelTeardown, testDynakube)

	return builder.Feature()
}
