//go:build e2e

package noinjection

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/exp"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/features/cloudnative"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers"
	dynakubeComponents "github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/components/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/kubernetes/objects/k8snamespace"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/sample"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/tenant"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

func Feature(t *testing.T) features.Feature {
	builder := features.New("cloudnative-disabled-auto-inject")

	secretConfig := tenant.GetSingleTenantSecret(t)
	testDynakube := *dynakubeComponents.New(
		dynakubeComponents.WithAnnotations(map[string]string{
			exp.InjectionAutomaticKey: "false",
		}),
		dynakubeComponents.WithAPIURL(secretConfig.APIURL),
		dynakubeComponents.WithCloudNativeSpec(cloudnative.DefaultCloudNativeSpec()),
	)

	// Register sample app install
	sampleNamespace := *k8snamespace.New("cloudnative-disabled-injection-sample")
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

		for _, pod := range pods.Items {
			if pod.DeletionTimestamp != nil {
				continue
			}

			require.Empty(t, pod.Spec.InitContainers)
		}

		return ctx
	}
}
