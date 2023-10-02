package network_zones

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/namespace"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/sampleapps"
	sample "github.com/Dynatrace/dynatrace-operator/test/helpers/sampleapps/base"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/setup"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/steps/assess"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/steps/teardown"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/tenant"
	"github.com/Dynatrace/dynatrace-operator/test/scenarios/cloudnative"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

const testNetworkZone = "testzone"
const annotationInjected = "oneagent.dynatrace.com/injected"
const annotationReason = "oneagent.dynatrace.com/reason"

func networkZones(t *testing.T) features.Feature {
	builder := features.New("dynakube in network zone")
	secretConfig := tenant.GetSingleTenantSecret(t)

	builder.Assess("create network zone before hand",
		tenant.CreateNetworkZone(secretConfig, testNetworkZone, []string{}, tenant.FallbackNone))

	// intentionally no ActiveGate, to block OA rollout and codemodules injection
	dynakubeBuilder := dynakube.NewBuilder().
		WithDefaultObjectMeta().
		ApiUrl(secretConfig.ApiUrl).
		NetworkZone(testNetworkZone).
		CloudNative(cloudnative.DefaultCloudNativeSpec())

	testDynakube := dynakubeBuilder.Build()

	// Register operator install
	operatorNamespaceBuilder := namespace.NewBuilder(testDynakube.Namespace)

	steps := setup.NewEnvironmentSetup(
		setup.CreateNamespaceWithoutTeardown(operatorNamespaceBuilder.Build()),
		setup.DeployOperatorViaMake(testDynakube.NeedsCSIDriver()))
	steps.CreateSetupSteps(builder)

	// Register dynakube install
	assess.InstallDynakube(builder, &secretConfig, testDynakube)

	// Register sample app install
	namespaceBuilder := namespace.NewBuilder("cloudnative-network-zone")

	sampleNamespace := namespaceBuilder.Build()
	sampleApp := sampleapps.NewSampleDeployment(t, testDynakube)
	sampleApp.WithNamespace(sampleNamespace)

	builder.Assess("create sample namespace", sampleApp.InstallNamespace())
	builder.Assess("install sample app", sampleApp.Install())

	// Register actual tests
	builder.Assess("check injection annotations on sample app pods", checkInjectionAnnotations(sampleApp))
	//	builder.Assess("check that OneAgent rollout is postponed", checkOneAgentRollout())

	// Register sample, DynaKube and operator uninstall
	builder.Teardown(sampleApp.UninstallNamespace())
	teardown.DeleteDynakube(builder, testDynakube)
	steps.CreateTeardownSteps(builder)

	return builder.Feature()
}

func checkOneAgentRollout() features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		return ctx
	}
}

func checkInjectionAnnotations(sampleApp sample.App) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		resource := envConfig.Client().Resources()
		samplePods := sampleApp.GetPods(ctx, t, resource)

		require.NotNil(t, samplePods)

		for _, pod := range samplePods.Items {
			require.NotNil(t, pod.Annotations)

			require.Contains(t, pod.Annotations, annotationInjected)
			assert.Equal(t, "true", pod.Annotations[annotationInjected])

			require.Contains(t, pod.Annotations, annotationReason)
			assert.Equal(t, "EmptyConnectionInfo", pod.Annotations[annotationReason])
		}
		return ctx
	}
}
