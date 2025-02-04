//go:build e2e

package disabled_auto_injection

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/features/cloudnative"
	"github.com/Dynatrace/dynatrace-operator/test/helpers"
	dynakubeComponents "github.com/Dynatrace/dynatrace-operator/test/helpers/components/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/namespace"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/sample"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/tenant"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/context"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

func Feature(t *testing.T) features.Feature {
	builder := features.New("cloudnative-disabled-auto-inject")

	secretConfig := tenant.GetSingleTenantSecret(t)
	testDynakube := *dynakubeComponents.New(
		dynakubeComponents.WithAnnotations(map[string]string{
			dynakube.AnnotationFeatureAutomaticInjection: "false",
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

	// Register actual test
	assessSampleInitContainersDisabled(builder, sampleApp)

	// Register sample, dynakubeComponents and operator uninstall
	builder.Teardown(sampleApp.Uninstall())
	dynakubeComponents.Delete(builder, helpers.LevelTeardown, testDynakube)

	return builder.Feature()
}

func assessSampleInitContainersDisabled(builder *features.FeatureBuilder, sampleApp *sample.App) {
	builder.Assess("sample apps don't have init containers", checkInitContainersNotInjected(sampleApp))
}

func checkInitContainersNotInjected(sampleApp *sample.App) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		resources := envConfig.Client().Resources()

		pods := sampleApp.GetPods(ctx, t, resources)
		require.NotEmpty(t, pods.Items)

		for _, podItem := range pods.Items {
			if podItem.DeletionTimestamp != nil {
				continue
			}

			require.NotNil(t, podItem)
			require.NotNil(t, podItem.Spec)
			require.Empty(t, podItem.Spec.InitContainers)
		}

		return ctx
	}
}
