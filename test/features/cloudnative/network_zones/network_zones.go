//go:build e2e

package network_zones

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/features/cloudnative"
	"github.com/Dynatrace/dynatrace-operator/test/helpers"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/activegate"
	dynakubeComponents "github.com/Dynatrace/dynatrace-operator/test/helpers/components/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/namespace"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/rand"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/sample"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/tenant"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/e2e-framework/klient/k8s"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/klient/wait/conditions"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

const (
	annotationInjected = "oneagent.dynatrace.com/injected"
	annotationReason   = "oneagent.dynatrace.com/reason"
	timeout            = 2 * time.Minute
)

// Feature defines the overall e2e test for testing OneAgent
// injection behavior when Dynatrace is configured with a network zone.
//
// It does the following to cover the scenario of ensuring OneAgent injection is properly
// blocked when no ActiveGate is available, and enabled once one is added:
//   - Creates test a network zone via the tenant helper (can be highly destructive)
//   - Configures a DynaKube custom resource without an ActiveGate => no activegate == no networkzone communication
//   - Installs a sample application
//   - Verifies the sample app pods do NOT have OneAgent injected, validated via pod annotations
//   - Updates the DynaKube to add an ActiveGate => so now networkzone communication is possible
//   - Restarts the sample app pods
//   - Verifies the sample app pods now DO have OneAgent injected
//
// Prerequisites:
// Have a tenant that has no activegates bound to it.
func Feature(t *testing.T) features.Feature {
	builder := features.New("dynakube in network zone")
	builder.WithLabel("name", "cloudnative-network-zone")
	secretConfig := tenant.GetSingleTenantSecret(t)

	networkZone, err := rand.GetRandomName(rand.WithPrefix("op-e2e-"), rand.WithLength(8))
	require.NoError(t, err)

	builder.Assess("create network zone before hand",
		tenant.CreateNetworkZone(secretConfig, networkZone, []string{}, tenant.FallbackNone))

	builder.Assess("wait for network zone to be ready",
		tenant.WaitForNetworkZone(secretConfig, networkZone, tenant.FallbackNone))

	// intentionally no ActiveGate, to block OA rollout and codemodules injection
	options := []dynakubeComponents.Option{
		dynakubeComponents.WithApiUrl(secretConfig.ApiUrl),
		dynakubeComponents.WithNetworkZone(networkZone),
		dynakubeComponents.WithCloudNativeSpec(cloudnative.DefaultCloudNativeSpec()),
	}

	testDynakube := *dynakubeComponents.New(options...)

	// Register sample app install
	sampleNamespace := *namespace.New("cloudnative-network-zone")
	sampleApp := sample.NewApp(t, &testDynakube,
		sample.AsDeployment(),
		sample.WithNamespace(sampleNamespace),
	)

	builder.Assess("create sample namespace", sampleApp.InstallNamespace())

	// Register dynakubeComponents install, do not wait for OneAgents to start up, because them not to is expected in this scenario
	dynakubeComponents.Create(builder, helpers.LevelAssess, &secretConfig, testDynakube)
	builder.Assess(
		fmt.Sprintf("'%s' dynakubeComponents phase changes to 'Running'", testDynakube.Name),
		dynakubeComponents.WaitForPhase(testDynakube, status.Deploying))
	builder.Assess("install sample app", sampleApp.Install())

	// Register actual tests
	builder.Assess("check injection annotations on sample app pods", checkInjectionAnnotations(sampleApp, "false", "EmptyConnectionInfo"))
	builder.Assess("make sure that OneAgent pods do not yet start up", checkOneAgentPodsDoNotStart(testDynakube, timeout))

	// update DynaKube to start AG, which should than enable OA rollout
	options = append(options, dynakubeComponents.WithActiveGate())
	testDynaKubeWithAG := *dynakubeComponents.New(options...)
	dynakubeComponents.Update(builder, helpers.LevelAssess, testDynaKubeWithAG)
	dynakubeComponents.VerifyStartup(builder, helpers.LevelAssess, testDynaKubeWithAG)

	builder.Assess("Restart sample app pods", sampleApp.Restart())
	builder.Assess("check injection annotations on sample app pods", checkInjectionAnnotations(sampleApp, "true", ""))

	// Register sample, DynaKube and operator uninstall
	builder.Teardown(sampleApp.Uninstall())
	dynakubeComponents.Delete(builder, helpers.LevelTeardown, testDynakube)

	builder.Teardown(activegate.WaitForStatefulSetPodsDeletion(&testDynakube, "activegate"))
	builder.Teardown(tenant.WaitForNetworkZoneDeletion(secretConfig, networkZone))

	return builder.Feature()
}

func checkInjectionAnnotations(sampleApp *sample.App, injected string, reason string) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		resource := envConfig.Client().Resources()
		samplePods := sampleApp.GetPods(ctx, t, resource)

		require.NotNil(t, samplePods)

		for _, pod := range samplePods.Items {
			require.NotNil(t, pod.Annotations)

			require.Contains(t, pod.Annotations, annotationInjected)
			assert.Equal(t, injected, pod.Annotations[annotationInjected])

			if injected == "false" && pod.Annotations[annotationInjected] == "false" {
				require.Contains(t, pod.Annotations, annotationReason)
				assert.Equal(t, reason, pod.Annotations[annotationReason])
			}
		}

		return ctx
	}
}

func checkOneAgentPodsDoNotStart(dk dynakube.DynaKube, timeout time.Duration) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		resources := envConfig.Client().Resources()
		err := wait.For(conditions.New(resources).ResourceMatch(&appsv1.DaemonSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      dk.OneAgentDaemonsetName(),
				Namespace: dk.Namespace,
			},
		}, func(object k8s.Object) bool {
			daemonset, isDaemonset := object.(*appsv1.DaemonSet)

			return isDaemonset && daemonset.Status.DesiredNumberScheduled == daemonset.Status.UpdatedNumberScheduled &&
				daemonset.Status.DesiredNumberScheduled == daemonset.Status.NumberReady
		}), wait.WithTimeout(timeout))

		require.Error(t, err)

		return ctx
	}
}
