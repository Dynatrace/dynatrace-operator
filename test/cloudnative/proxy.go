//go:build e2e

package cloudnative

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/test/bash"
	"github.com/Dynatrace/dynatrace-operator/test/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/kubeobjects/deployment"
	"github.com/Dynatrace/dynatrace-operator/test/kubeobjects/manifests"
	"github.com/Dynatrace/dynatrace-operator/test/kubeobjects/pod"
	"github.com/Dynatrace/dynatrace-operator/test/logs"
	"github.com/Dynatrace/dynatrace-operator/test/proxy"
	"github.com/Dynatrace/dynatrace-operator/test/sampleapps"
	"github.com/Dynatrace/dynatrace-operator/test/secrets"
	"github.com/Dynatrace/dynatrace-operator/test/setup"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

const (
	dynatraceNetworkPolicy = "../testdata/network/dynatrace-denial.yaml"
	timeoutError           = "dial tcp 54.88.45.104:443: i/o timeout"
	secretUnchanged        = "secret unchanged"
	sampleNSNetworkPolicy  = "../testdata/network/sample-ns-denial.yaml"
	sampleNS               = "../testdata/cloudnative/test-namespace.yaml"
	DtProxy                = "DT_PROXY"
	agentPath              = "/opt/dynatrace/oneagent-paas"
)

func WithProxy(t *testing.T, proxySpec *v1beta1.DynaKubeProxy) features.Feature {
	secretConfig, err := secrets.DefaultSingleTenant(afero.NewOsFs())

	require.NoError(t, err)

	cloudNativeWithProxy := features.New("cloudNative with proxy")
	cloudNativeWithProxy.Setup(manifests.InstallFromFile(sampleNS))
	cloudNativeWithProxy.Setup(manifests.InstallFromFile(dynatraceNetworkPolicy))
	cloudNativeWithProxy.Setup(manifests.InstallFromFile(sampleNSNetworkPolicy))

	setup.InstallAndDeploy(cloudNativeWithProxy, secretConfig, "../testdata/cloudnative/sample-deployment.yaml")
	setup.AssessDeployment(cloudNativeWithProxy)

	proxy.InstallProxy(cloudNativeWithProxy, proxySpec)

	cloudNativeWithProxy.Assess("install dynakube", dynakube.Apply(
		dynakube.NewBuilder().
			WithDefaultObjectMeta().
			ApiUrl(secretConfig.ApiUrl).
			CloudNative(&v1beta1.CloudNativeFullStackSpec{}).
			Proxy(proxySpec).
			Build()),
	)
	setup.AssessDynakubeStartup(cloudNativeWithProxy)

	cloudNativeWithProxy.Assess("restart sample apps", sampleapps.Restart)
	cloudNativeWithProxy.Assess("check existing init container and env vars", checkSampleInitContainer)
	cloudNativeWithProxy.Assess("check logs", checkLogs)

	return cloudNativeWithProxy.Feature()
}

func checkLogs(ctx context.Context, t *testing.T, environmentConfig *envconf.Config) context.Context {
	resources := environmentConfig.Client().Resources()
	clientset, err := kubernetes.NewForConfig(resources.GetConfig())

	require.NoError(t, err)

	err = deployment.NewQuery(ctx, resources, client.ObjectKey{
		Name:      "dynatrace-operator",
		Namespace: "dynatrace",
	}).ForEachPod(func(podItem v1.Pod) {
		logStream, err := clientset.CoreV1().Pods(podItem.Namespace).GetLogs(podItem.Name, &v1.PodLogOptions{}).Stream(ctx)
		require.NoError(t, err)
		logs.AssertLogContains(t, logStream, timeoutError)

		logStream, err = clientset.CoreV1().Pods(podItem.Namespace).GetLogs(podItem.Name, &v1.PodLogOptions{}).Stream(ctx)
		require.NoError(t, err)
		logs.AssertLogContains(t, logStream, secretUnchanged)
	})

	require.NoError(t, err)

	err = deployment.NewQuery(ctx, resources, client.ObjectKey{
		Name:      "myapp",
		Namespace: "test-namespace-1",
	}).ForEachPod(func(podItem v1.Pod) {
		logStream, err := clientset.CoreV1().Pods(podItem.Namespace).GetLogs(podItem.Name, &v1.PodLogOptions{
			Container: "install-oneagent",
		}).Stream(ctx)
		require.NoError(t, err)
		logs.AssertLogContains(t, logStream, proxy.ProxySpec.Value)
	})

	require.NoError(t, err)
	return ctx
}

func checkSampleInitContainer(ctx context.Context, t *testing.T, environmentConfig *envconf.Config) context.Context {
	resources := environmentConfig.Client().Resources()
	restConfig := environmentConfig.Client().RESTConfig()
	pods := sampleapps.Get(t, ctx, resources)

	for _, podItem := range pods.Items {
		require.NotNil(t, podItem)
		require.NotNil(t, podItem.Spec)
		require.NotNil(t, podItem.Spec.InitContainers)

		for _, container := range podItem.Spec.Containers {
			if container.Name == "myapp" {
				require.NotNil(t, container.Env)
				require.True(t, containsDtProxyEnvVar(container.Env, DtProxy))
				for _, env := range container.Env {
					if env.Name == DtProxy {
						require.NotNil(t, env.Value)
					}
					break
				}
				break
			}
		}

		var result *pod.ExecutionResult
		result, err := pod.
			NewExecutionQuery(podItem, sampleapps.Name,
				bash.ListDirectory(agentPath)).
			Execute(restConfig)

		require.NoError(t, err)
		assert.Contains(t, result.StdOut.String(), "agent")
		assert.Empty(t, result.StdErr.String())
	}
	return ctx
}

func containsDtProxyEnvVar(envs []v1.EnvVar, dtproxy string) bool {
	for _, env := range envs {
		if env.Name == dtproxy {
			return true
		}
	}

	return false
}
