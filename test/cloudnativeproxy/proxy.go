//go:build e2e

package cloudnativeproxy

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	"github.com/Dynatrace/dynatrace-operator/test/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/kubeobjects/daemonset"
	"github.com/Dynatrace/dynatrace-operator/test/kubeobjects/manifests"
	"github.com/Dynatrace/dynatrace-operator/test/kubeobjects/namespace"
	"github.com/Dynatrace/dynatrace-operator/test/oneagent"
	"github.com/Dynatrace/dynatrace-operator/test/operator"
	"github.com/Dynatrace/dynatrace-operator/test/proxy"
	"github.com/Dynatrace/dynatrace-operator/test/sampleapps"
	"github.com/Dynatrace/dynatrace-operator/test/secrets"
	"github.com/Dynatrace/dynatrace-operator/test/setup"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

const (
	dynatraceNetworkPolicy       = "../testdata/network/dynatrace-denial.yaml"
	httpsProxy                   = "https_proxy"
	sampleNamespaceNetworkPolicy = "../testdata/network/sample-ns-denial.yaml"
	sampleNamespace              = "test-namespace-1"
	DtProxy                      = "DT_PROXY"
	agentPath                    = "/opt/dynatrace/oneagent-paas"
	sampleAppDeployment          = "../testdata/cloudnative/sample-deployment.yaml"
)

var injectionLabel = map[string]string{
	"inject": "dynakube",
}

func WithProxy(t *testing.T, proxySpec *v1beta1.DynaKubeProxy) features.Feature {
	secretConfig, err := secrets.DefaultSingleTenant(afero.NewOsFs())

	require.NoError(t, err)

	cloudNativeWithProxy := features.New("cloudNative with proxy")
	cloudNativeWithProxy.Setup(namespace.Create(
		namespace.NewBuilder(sampleNamespace).
			WithLabels(injectionLabel).
			Build()),
	)
	cloudNativeWithProxy.Setup(secrets.ApplyDefault(secretConfig))
	cloudNativeWithProxy.Setup(operator.InstallAllForKubernetes())
	setup.AssessDeployment(cloudNativeWithProxy)

	proxy.InstallProxy(cloudNativeWithProxy, proxySpec)

	cloudNativeWithProxy.Assess("install dynakube", dynakube.Apply(
		dynakube.NewBuilder().
			WithDefaultObjectMeta().
			WithDynakubeNamespaceSelector().
			ApiUrl(secretConfig.ApiUrl).
			CloudNative(&v1beta1.CloudNativeFullStackSpec{}).
			Proxy(proxySpec).
			Build()),
	)
	setup.AssessDynakubeStartup(cloudNativeWithProxy)

	cloudNativeWithProxy.Assess("osAgent can connect", oneagent.OSAgentCanConnect())
	cloudNativeWithProxy.Assess("cut off dynatrace namespace", manifests.InstallFromFile(dynatraceNetworkPolicy))
	cloudNativeWithProxy.Assess("cut off sample namespace", manifests.InstallFromFile(sampleNamespaceNetworkPolicy))
	cloudNativeWithProxy.Assess("check env variables of oneagent pods", checkOneAgentEnvVars)
	cloudNativeWithProxy.Assess("install deployment", manifests.InstallFromFile(sampleAppDeployment))
	cloudNativeWithProxy.Assess("check existing init container and env var", checkSampleInitContainerEnvVar)

	return cloudNativeWithProxy.Feature()
}

func checkOneAgentEnvVars(ctx context.Context, t *testing.T, environmentConfig *envconf.Config) context.Context {
	resources := environmentConfig.Client().Resources()

	err := daemonset.NewQuery(ctx, resources, client.ObjectKey{
		Name:      "dynakube-oneagent",
		Namespace: "dynatrace",
	}).ForEachPod(func(podItem v1.Pod) {
		require.NotNil(t, podItem)
		require.NotNil(t, podItem.Spec)

		for _, container := range podItem.Spec.Containers {
			if container.Name == "dynatrace-oneagent" {
				require.NotNil(t, container.Env)
				require.True(t, kubeobjects.EnvVarIsIn(container.Env, httpsProxy))
				for _, env := range container.Env {
					if env.Name == httpsProxy {
						require.NotNil(t, env.Value)
					}
				}
			}
		}
	})

	require.NoError(t, err)
	return ctx
}

func checkSampleInitContainerEnvVar(ctx context.Context, t *testing.T, environmentConfig *envconf.Config) context.Context {
	resources := environmentConfig.Client().Resources()
	pods := sampleapps.Get(t, ctx, resources)

	for _, podItem := range pods.Items {
		require.NotNil(t, podItem)
		require.NotNil(t, podItem.Spec)
		require.NotNil(t, podItem.Spec.InitContainers)

		for _, container := range podItem.Spec.Containers {
			if container.Name == sampleapps.Name {
				require.NotNil(t, container.Env)
				require.True(t, kubeobjects.EnvVarIsIn(container.Env, DtProxy))
				for _, env := range container.Env {
					if env.Name == DtProxy {
						require.NotNil(t, env.Value)
					}
				}
			}
		}
	}
	return ctx
}
