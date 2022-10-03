//go:build e2e

package cloudnative

import (
	"bytes"
	"context"
	"io"
	"os"
	"path"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/test/activegate"
	"github.com/Dynatrace/dynatrace-operator/test/csi"
	"github.com/Dynatrace/dynatrace-operator/test/kubeobjects/environment"
	"github.com/Dynatrace/dynatrace-operator/test/kubeobjects/manifests"
	"github.com/Dynatrace/dynatrace-operator/test/kubeobjects/namespace"
	"github.com/Dynatrace/dynatrace-operator/test/kubeobjects/pod"
	"github.com/Dynatrace/dynatrace-operator/test/oneagent"
	"github.com/Dynatrace/dynatrace-operator/test/operator"
	"github.com/Dynatrace/dynatrace-operator/test/secrets"
	"github.com/Dynatrace/dynatrace-operator/test/webhook"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/e2e-framework/pkg/env"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

var testEnvironment env.Environment

func TestMain(m *testing.M) {
	testEnvironment = environment.Get()
	testEnvironment.BeforeEachTest(deleteDynakubeIfExists())
	testEnvironment.BeforeEachTest(oneagent.WaitForDaemonSetPodsDeletion())
	testEnvironment.BeforeEachTest(namespace.DeleteIfExists(sampleAppsNamespace))
	testEnvironment.BeforeEachTest(namespace.Recreate(dynatraceNamespace))

	testEnvironment.AfterEachTest(deleteDynakubeIfExists())
	testEnvironment.AfterEachTest(oneagent.WaitForDaemonSetPodsDeletion())
	testEnvironment.AfterEachTest(namespace.Delete(sampleAppsNamespace))
	testEnvironment.AfterEachTest(namespace.Delete(dynatraceNamespace))

	testEnvironment.Run(m)
}

func TestCloudNative(t *testing.T) {
	testEnvironment.Test(t, install(t))
	//testEnvironment.Test(t, codeModules(t))
}

func installAndDeploy(builder *features.FeatureBuilder, secretConfig secrets.Secret, deploymentPath string) {
	builder.Setup(secrets.ApplyDefault(secretConfig))
	builder.Setup(operator.InstallForKubernetes())
	builder.Setup(manifests.InstallFromFile(deploymentPath))
}

func assessDeployment(builder *features.FeatureBuilder) {
	builder.Assess("operator started", operator.WaitForDeployment())
	builder.Assess("webhook started", webhook.WaitForDeployment())
	builder.Assess("csi driver started", csi.WaitForDaemonset())
}

func assessDynakubeStartup(builder *features.FeatureBuilder) {
	builder.Assess("activegate started", activegate.WaitForStatefulSet())
	builder.Assess("oneagent started", oneagent.WaitForDaemonset())
	builder.Assess("dynakube phase changes to 'Running'", waitForDynakubePhase())
}

func assessOneAgentsAreRunning(builder *features.FeatureBuilder) {
	builder.Assess("restart sample apps", restartSampleApps)
	builder.Assess("sample apps have working init containers", checkInitContainers)
	builder.Assess("osAgent can connect", osAgentsCanConnect())
}

func getSecretConfig(t *testing.T) secrets.Secret {
	currentWorkingDirectory, err := os.Getwd()
	require.NoError(t, err)

	secretPath := path.Join(currentWorkingDirectory, installSecretsPath)
	secretConfig, err := secrets.NewFromConfig(afero.NewOsFs(), secretPath)

	require.NoError(t, err)

	return secretConfig
}

func osAgentsCanConnect() features.Func {
	return func(ctx context.Context, t *testing.T, environmentConfig *envconf.Config) context.Context {
		resource := environmentConfig.Client().Resources()
		clientset, err := kubernetes.NewForConfig(resource.GetConfig())

		require.NoError(t, err)
		require.NoError(t, oneagent.ForEachPod(ctx, resource, func(pod corev1.Pod) {
			logStream, err := clientset.CoreV1().Pods(pod.Namespace).GetLogs(pod.Name, &corev1.PodLogOptions{}).Stream(ctx)

			require.NoError(t, err)

			assertLogContains(t, logStream, "[oneagentos] [PingReceiver] Ping received: Healthy(0)")
		}))

		return ctx
	}
}

func checkInitContainers(ctx context.Context, t *testing.T, environmentConfig *envconf.Config) context.Context {
	var pods corev1.PodList
	resources := environmentConfig.Client().Resources()

	require.NoError(t, resources.WithNamespace(sampleAppsNamespace).List(ctx, &pods))

	clientset, err := kubernetes.NewForConfig(resources.GetConfig())

	require.NoError(t, err)

	for _, podItem := range pods.Items {
		require.NotNil(t, podItem)
		require.NotNil(t, podItem.Spec)
		require.NotEmpty(t, podItem.Spec.InitContainers)

		oneAgentInstallContainer := podItem.Spec.InitContainers[0]

		assert.Equal(t, oneAgentInstallContainerName, oneAgentInstallContainer.Name)

		logStream, err := clientset.CoreV1().Pods(podItem.Namespace).GetLogs(podItem.Name, &corev1.PodLogOptions{
			Container: oneAgentInstallContainerName,
		}).Stream(ctx)

		require.NoError(t, err)
		assertLogContains(t, logStream, "standalone agent init completed")

		executionQuery := pod.NewExecutionQuery(podItem, sampleAppsName, "cat /opt/dynatrace/oneagent-paas/log/nginx/ruxitagent_nginx_myapp-__bootstrap_1.0.log")
		executionResult, err := executionQuery.Execute(environmentConfig.Client().RESTConfig())

		require.NoError(t, err)

		stdOut := executionResult.StdOut.String()
		stdErr := executionResult.StdErr.String()

		assert.NotEmpty(t, stdOut)
		assert.Empty(t, stdErr)
		assert.Contains(t, stdOut, "info    [native] Communicating via https://")
	}

	return ctx
}

func assertLogContains(t *testing.T, logStream io.ReadCloser, contains string) {
	buffer := new(bytes.Buffer)
	copied, err := io.Copy(buffer, logStream)

	require.NoError(t, err)
	require.Equal(t, int64(buffer.Len()), copied)
	assert.Contains(t, buffer.String(), contains)
}
