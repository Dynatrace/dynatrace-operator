//go:build e2e

package applicationmonitoring

import (
	"context"
	"fmt"
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/config"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects/address"
	"github.com/Dynatrace/dynatrace-operator/src/webhook"
	"github.com/Dynatrace/dynatrace-operator/src/webhook/mutation/pod_mutator/oneagent_mutation"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/codemodules"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/deployment"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/pod"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/sampleapps"
	sample "github.com/Dynatrace/dynatrace-operator/test/helpers/sampleapps/base"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/shell"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/steps/assess"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/steps/teardown"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/tenant"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

var readOnlyInjection = map[string]string{dynatracev1beta1.AnnotationFeatureReadOnlyCsiVolume: "true"}

func readOnlyCSIVolume(t *testing.T) features.Feature {
	builder := features.New("read only csi volume")
	secretConfig := tenant.GetSingleTenantSecret(t)
	testDynakube := dynakube.NewBuilder().
		WithDefaultObjectMeta().
		WithAnnotations(readOnlyInjection).
		ApiUrl(secretConfig.ApiUrl).
		ApplicationMonitoring(&dynatracev1beta1.ApplicationMonitoringSpec{
			UseCSIDriver: address.Of(true),
		}).Build()
	sampleDeployment := sampleapps.NewSampleDeployment(t, testDynakube)

	assess.InstallDynatrace(builder, &secretConfig, testDynakube)

	builder.Assess("install sample deployment and wait till ready", sampleDeployment.Install())
	builder.Assess("check init container env var", checkInitContainerEnvVar(sampleDeployment))
	builder.Assess("check mounted volumes", checkMountedVolumes(sampleDeployment))
	builder.Assess(fmt.Sprintf("check %s has no conn info", codemodules.RuxitAgentProcFile), codemodules.CheckRuxitAgentProcFileHasNoConnInfo(testDynakube))

	builder.WithTeardown("removing sample namespace", sampleDeployment.UninstallNamespace())

	teardown.UninstallDynatrace(builder, testDynakube)

	return builder.Feature()
}

func checkInitContainerEnvVar(sampleApp sample.App) features.Func {
	return func(ctx context.Context, t *testing.T, environmentConfig *envconf.Config) context.Context {
		resources := environmentConfig.Client().Resources()
		pods := sampleApp.GetPods(ctx, t, resources)

		for _, podItem := range pods.Items {
			for _, initContainer := range podItem.Spec.InitContainers {
				require.NotEmpty(t, initContainer)
				if initContainer.Name == webhook.InstallContainerName {
					require.Equal(t, "true", kubeobjects.FindEnvVar(initContainer.Env, config.AgentReadonlyCSI).Value)
				}
			}
		}
		return ctx
	}
}

func checkMountedVolumes(sampleApp sample.App) features.Func {
	return func(ctx context.Context, t *testing.T, environmentConfig *envconf.Config) context.Context {
		resources := environmentConfig.Client().Resources()

		err := deployment.NewQuery(ctx, resources, client.ObjectKey{
			Name:      sampleApp.Name(),
			Namespace: sampleApp.Namespace().Name,
		}).ForEachPod(func(podItem corev1.Pod) {
			listCommand := shell.ListDirectory(oneagent_mutation.OneAgentConfMountPath)
			result, err := pod.Exec(ctx, resources, podItem, sampleApp.Namespace().Name, listCommand...)

			require.NoError(t, err)
			assert.Contains(t, result.StdOut.String(), codemodules.RuxitAgentProcFile)
		})
		require.NoError(t, err)
		return ctx
	}
}
