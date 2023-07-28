//go:build e2e

package disabled_auto_injection

import (
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/namespace"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/sampleapps"
	sample "github.com/Dynatrace/dynatrace-operator/test/helpers/sampleapps/base"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/steps/assess"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/steps/teardown"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/tenant"
	"github.com/Dynatrace/dynatrace-operator/test/scenarios/cloudnative"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/context"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

func automaticInjectionDisabled(t *testing.T) features.Feature {
	builder := features.New("default installation")

	secretConfig := tenant.GetSingleTenantSecret(t)

	dynakubeBuilder := dynakube.NewBuilder().
		WithDefaultObjectMeta().
		WithAnnotations(map[string]string{
			dynatracev1beta1.AnnotationFeatureAutomaticInjection: "false",
		}).
		ApiUrl(secretConfig.ApiUrl).
		CloudNative(cloudnative.DefaultCloudNativeSpec())

	testDynakube := dynakubeBuilder.Build()

	// Register operator install
	operatorNamespaceBuilder := namespace.NewBuilder(testDynakube.Namespace)

	assess.InstallOperatorFromSourceWithCustomNamespace(builder, operatorNamespaceBuilder.Build(), testDynakube)

	// Register sample app install
	namespaceBuilder := namespace.NewBuilder("cloudnative-disabled-injection-sample")
	sampleNamespace := namespaceBuilder.Build()
	sampleApp := sampleapps.NewSampleDeployment(t, testDynakube)
	sampleApp.WithNamespace(sampleNamespace)
	builder.Assess("create sample namespace", sampleApp.InstallNamespace())

	// Register dynakube install
	assess.InstallDynakube(builder, &secretConfig, testDynakube)

	// Register sample app install
	builder.Assess("install sample app", sampleApp.Install())

	// Register actual test
	assessSampleInitContainersDisabled(builder, sampleApp)

	// Register sample, dynakube and operator uninstall
	builder.Teardown(sampleApp.UninstallNamespace())
	teardown.UninstallDynatrace(builder, testDynakube)

	return builder.Feature()
}

func assessSampleInitContainersDisabled(builder *features.FeatureBuilder, sampleApp sample.App) {
	builder.Assess("sample apps don't have init containers", checkInitContainersNotInjected(sampleApp))
}

func checkInitContainersNotInjected(sampleApp sample.App) features.Func {
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
