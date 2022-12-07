//go:build e2e

package cloudnative

import (
	"context"
	"path"
	"testing"
	"time"

	"github.com/Dynatrace/dynatrace-operator/test/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/kubeobjects/manifests"
	"github.com/Dynatrace/dynatrace-operator/test/kubeobjects/pod"
	"github.com/Dynatrace/dynatrace-operator/test/operator"
	"github.com/Dynatrace/dynatrace-operator/test/project"
	"github.com/Dynatrace/dynatrace-operator/test/sampleapps"
	"github.com/Dynatrace/dynatrace-operator/test/secrets"
	"github.com/Dynatrace/dynatrace-operator/test/setup"
	"github.com/Dynatrace/dynatrace-operator/test/shell"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

const (
	agentMountPath        = "/opt/dynatrace/oneagent-paas"
	sampleNSPath          = "cloudnative/test-namespace.yaml"
	deploymentPath        = "cloudnative/codemodules-deployment.yaml"
	ldPreloadError        = "ERROR: ld.so: object '/opt/dynatrace/oneagent-paas/agent/lib64/liboneagentproc.so' from LD_PRELOAD cannot be preloaded"
	podRestartTimeout     = 5 * time.Minute
	restartCountThreshold = int32(3)
	csiNetworkPolicy      = "network/csi-denial.yaml"
)

func NetworkProblems(t *testing.T) features.Feature {
	secretConfigs, err := secrets.DefaultMultiTenant(afero.NewOsFs())

	require.NoError(t, err)

	createNetworkProblems := features.New("creating network problems")
	createNetworkProblems.Setup(manifests.InstallFromFile(path.Join(project.TestDataDir(), csiNetworkPolicy)))
	createNetworkProblems.Setup(manifests.InstallFromFile(path.Join(project.TestDataDir(), sampleNSPath)))
	createNetworkProblems.Setup(secrets.ApplyDefault(secretConfigs[0]))
	createNetworkProblems.Setup(operator.InstallFromSource(true))

	setup.AssessOperatorDeployment(createNetworkProblems)

	createNetworkProblems.Assess("install dynakube", dynakube.Apply(
		dynakube.NewBuilder().
			WithDefaultObjectMeta().
			ApiUrl(secretConfigs[0].ApiUrl).
			CloudNative(codeModulesSpec()).
			Build()),
	)
	createNetworkProblems.Assess("dynakube phase changes to 'Running'", dynakube.WaitForDynakubePhase(dynakube.NewBuilder().WithDefaultObjectMeta().Build()))
	createNetworkProblems.Assess("install deployment", manifests.InstallFromFile(path.Join(project.TestDataDir(), deploymentPath)))
	createNetworkProblems.Assess("start sample apps and injection", sampleapps.Install)
	createNetworkProblems.Assess("check for dummy volume", checkForDummyVolume)
	createNetworkProblems.Assess("check pods after sleep", checkPodsAfterSleep)

	return createNetworkProblems.Feature()
}

func checkForDummyVolume(ctx context.Context, t *testing.T, environmentConfig *envconf.Config) context.Context {
	resources := environmentConfig.Client().Resources()
	restConfig := environmentConfig.Client().RESTConfig()
	pods := sampleapps.Get(ctx, t, resources)

	for _, podItem := range pods.Items {
		require.NotNil(t, podItem)
		require.NotNil(t, podItem.Spec)
		require.NotEmpty(t, podItem.Spec.InitContainers)

		var result *pod.ExecutionResult
		result, err := pod.
			NewExecutionQuery(podItem, sampleapps.Name,
				shell.ListDirectory(agentMountPath)...).
			Execute(restConfig)

		require.NoError(t, err)
		assert.Contains(t, result.StdErr.String(), ldPreloadError)
	}
	return ctx
}

func checkPodsAfterSleep(ctx context.Context, t *testing.T, environmentConfig *envconf.Config) context.Context {
	resources := environmentConfig.Client().Resources()
	samplePods := sampleapps.Get(ctx, t, resources)

	for _, podItem := range samplePods.Items {
		require.NotNil(t, podItem)
		require.NotNil(t, podItem.Spec)
		require.NotEmpty(t, podItem.Spec.InitContainers)

		assert.Equal(t, podItem.Status.Phase, corev1.PodPhase("Running"))
	}

	time.Sleep(podRestartTimeout)

	samplePods = sampleapps.Get(ctx, t, resources)
	for _, podItem := range samplePods.Items {
		require.NotNil(t, podItem)
		require.NotNil(t, podItem.Spec)
		require.NotEmpty(t, podItem.Spec.InitContainers)

		assert.Equal(t, podItem.Status.Phase, corev1.PodPhase("Running"))

		for _, containerStatus := range podItem.Status.ContainerStatuses {
			assert.Less(t, containerStatus.RestartCount, restartCountThreshold)
		}
	}

	return ctx
}
