//go:build e2e

package applicationmonitoring

import (
	"context"
	"fmt"
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/address"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/env"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod_mutator/oneagent_mutation"
	"github.com/Dynatrace/dynatrace-operator/test/helpers"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/codemodules"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/deployment"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/pod"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/sample"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/shell"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/tenant"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

var readOnlyInjection = map[string]string{dynatracev1beta1.AnnotationFeatureReadOnlyCsiVolume: "true"}

func ReadOnlyCSIVolume(t *testing.T) features.Feature {
	builder := features.New("read only csi volume")
	builder.WithLabel("name", "app-read-only-csi-volume")
	secretConfig := tenant.GetSingleTenantSecret(t)
	testDynakube := *dynakube.New(
		dynakube.WithAnnotations(readOnlyInjection),
		dynakube.WithApiUrl(secretConfig.ApiUrl),
		dynakube.WithApplicationMonitoringSpec(&dynatracev1beta1.ApplicationMonitoringSpec{
			UseCSIDriver: address.Of(true),
		}),
	)
	sampleDeployment := sample.NewApp(t, &testDynakube, sample.AsDeployment())
	builder.Assess("install sample deployment namespace", sampleDeployment.InstallNamespace())

	dynakube.Install(builder, helpers.LevelAssess, &secretConfig, testDynakube)

	builder.Assess("install sample deployment and wait till ready", sampleDeployment.Install())
	builder.Assess("check init container env var", checkInitContainerEnvVar(sampleDeployment))
	builder.Assess("check mounted volumes", checkMountedVolumes(sampleDeployment))
	builder.Assess(fmt.Sprintf("check %s has no conn info", codemodules.RuxitAgentProcFile), codemodules.CheckRuxitAgentProcFileHasNoConnInfo(testDynakube))

	builder.WithTeardown("removing sample namespace", sampleDeployment.Uninstall())

	dynakube.Delete(builder, helpers.LevelTeardown, testDynakube)
	return builder.Feature()
}

func checkInitContainerEnvVar(sampleApp *sample.App) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		resources := envConfig.Client().Resources()
		pods := sampleApp.GetPods(ctx, t, resources)

		for _, podItem := range pods.Items {
			for _, initContainer := range podItem.Spec.InitContainers {
				require.NotEmpty(t, initContainer)
				if initContainer.Name == webhook.InstallContainerName {
					require.Equal(t, "true", env.FindEnvVar(initContainer.Env, consts.AgentReadonlyCSI).Value)
				}
			}
		}
		return ctx
	}
}

func checkMountedVolumes(sampleApp *sample.App) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		resources := envConfig.Client().Resources()

		err := deployment.NewQuery(ctx, resources, client.ObjectKey{
			Name:      sampleApp.Name(),
			Namespace: sampleApp.Namespace(),
		}).ForEachPod(func(podItem corev1.Pod) {
			listCommand := shell.ListDirectory(oneagent_mutation.OneAgentConfMountPath)
			result, err := pod.Exec(ctx, resources, podItem, sampleApp.ContainerName(), listCommand...)

			require.NoError(t, err)
			assert.Contains(t, result.StdOut.String(), codemodules.RuxitAgentProcFile)
		})
		require.NoError(t, err)
		return ctx
	}
}
