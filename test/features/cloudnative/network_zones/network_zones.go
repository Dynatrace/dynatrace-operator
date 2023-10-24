//go:build e2e

package network_zones

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	dynakubev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/features/cloudnative"
	"github.com/Dynatrace/dynatrace-operator/test/helpers"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/activegate"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/namespace"
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

const testNetworkZone = "testzone"
const annotationInjected = "oneagent.dynatrace.com/injected"
const annotationReason = "oneagent.dynatrace.com/reason"

func Feature(t *testing.T) features.Feature {
	builder := features.New("dynakube in network zone")
	secretConfig := tenant.GetSingleTenantSecret(t)

	builder.Assess("create network zone before hand",
		tenant.CreateNetworkZone(secretConfig, testNetworkZone, []string{}, tenant.FallbackNone))

	// intentionally no ActiveGate, to block OA rollout and codemodules injection
	options := []dynakube.Option{
		dynakube.WithApiUrl(secretConfig.ApiUrl),
		dynakube.WithNetworkZone(testNetworkZone),
		dynakube.WithCloudNativeSpec(cloudnative.DefaultCloudNativeSpec()),
	}

	testDynakube := *dynakube.New(options...)

	// Register sample app install
	sampleNamespace := *namespace.New("cloudnative-network-zone")
	sampleApp := sample.NewApp(t, &testDynakube,
		sample.AsDeployment(),
		sample.WithNamespace(sampleNamespace),
	)

	builder.Assess("create sample namespace", sampleApp.InstallNamespace())

	// Register dynakube install, do not wait for OneAgents to start up, because them not to is expected in this scenario
	dynakube.Create(builder, helpers.LevelAssess, &secretConfig, testDynakube)
	builder.Assess(
		fmt.Sprintf("'%s' dynakube phase changes to 'Running'", testDynakube.Name),
		dynakube.WaitForPhase(testDynakube, status.Deploying))
	builder.Assess("install sample app", sampleApp.Install())

	// Register actual tests
	builder.Assess("check injection annotations on sample app pods", checkInjectionAnnotations(sampleApp, "false", "EmptyConnectionInfo"))
	builder.Assess("make sure that OneAgent pods do not yet start up", checkOneAgentPodsDoNotStart(testDynakube, 2*time.Minute))

	// update DynaKube to start AG, which should than enable OA rollout
	options = append(options, dynakube.WithActiveGate())
	testDynaKubeWithAG := *dynakube.New(options...)
	dynakube.Update(builder, helpers.LevelAssess, testDynaKubeWithAG)
	dynakube.VerifyStartup(builder, helpers.LevelAssess, testDynaKubeWithAG)

	builder.Assess("Restart sample app pods", sampleApp.Restart())
	builder.Assess("check injection annotations on sample app pods", checkInjectionAnnotations(sampleApp, "true", ""))

	// Register sample, DynaKube and operator uninstall
	builder.Teardown(sampleApp.Uninstall())
	dynakube.Delete(builder, helpers.LevelTeardown, testDynakube)

	builder.Teardown(activegate.WaitForStatefulSetPodsDeletion(&testDynakube, "activegate"))
	builder.Teardown(tenant.WaitForNetworkZoneDeletion(secretConfig, testNetworkZone))
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

			if injected == "false" {
				require.Contains(t, pod.Annotations, annotationReason)
				assert.Equal(t, reason, pod.Annotations[annotationReason])
			}
		}
		return ctx
	}
}

func checkOneAgentPodsDoNotStart(testDynakube dynakubev1beta1.DynaKube, timeout time.Duration) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		resources := envConfig.Client().Resources()
		err := wait.For(conditions.New(resources).ResourceMatch(&appsv1.DaemonSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testDynakube.OneAgentDaemonsetName(),
				Namespace: testDynakube.Namespace,
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
