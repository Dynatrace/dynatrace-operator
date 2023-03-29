//go:build e2e

package cloudnativeproxy

import (
	"context"
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/daemonset"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/namespace"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/proxy"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/sampleapps"
	sample "github.com/Dynatrace/dynatrace-operator/test/helpers/sampleapps/base"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/steps/assess"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/steps/teardown"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/tenant"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

const (
	httpsProxy = "https_proxy"
	dtProxy    = "DT_PROXY"
)

func withProxy(t *testing.T, proxySpec *dynatracev1beta1.DynaKubeProxy) features.Feature {
	builder := features.New("cloudNative with proxy")
	secretConfig := tenant.GetSingleTenantSecret(t)

	testDynakube := dynakube.NewBuilder().
		WithDefaultObjectMeta().
		WithDynakubeNamespaceSelector().
		ApiUrl(secretConfig.ApiUrl).
		CloudNative(&dynatracev1beta1.CloudNativeFullStackSpec{}).
		Proxy(proxySpec).
		Build()

	sampleNamespace := namespace.NewBuilder("proxy-sample").WithLabels(testDynakube.NamespaceSelector().MatchLabels).Build()
	sampleApp := sampleapps.NewSampleDeployment(t, testDynakube)
	sampleApp.WithLabels(testDynakube.NamespaceSelector().MatchLabels)
	sampleApp.WithNamespace(sampleNamespace)

	// Register sample namespace create and delete
	builder.Assess("create sample namespace", namespace.Create(sampleNamespace))
	builder.Teardown(sampleApp.UninstallNamespace())

	// Register operator install
	assess.InstallOperatorFromSource(builder, testDynakube)

	// Register proxy create and delete
	proxy.SetupProxyWithTeardown(builder, testDynakube)
	proxy.CutOffDynatraceNamespace(builder, proxySpec)
	proxy.CutOffSampleNamespace(builder, proxySpec)

	// Register actual test
	assess.InstallDynakube(builder, &secretConfig, testDynakube)
	builder.Assess("check env variables of oneagent pods", checkOneAgentEnvVars(testDynakube))
	builder.Assess("install sample app", sampleApp.Install())
	builder.Assess("check existing init container and env var", checkSampleInitContainerEnvVars(sampleApp))

	// Register operator and dynakube uninstall
	teardown.UninstallDynatrace(builder, testDynakube)

	return builder.Feature()
}

func checkOneAgentEnvVars(dynakube dynatracev1beta1.DynaKube) features.Func {
	return func(ctx context.Context, t *testing.T, environmentConfig *envconf.Config) context.Context {
		resources := environmentConfig.Client().Resources()
		err := daemonset.NewQuery(ctx, resources, client.ObjectKey{
			Name:      dynakube.OneAgentDaemonsetName(),
			Namespace: dynakube.Namespace,
		}).ForEachPod(func(podItem corev1.Pod) {
			require.NotNil(t, podItem)
			require.NotNil(t, podItem.Spec)

			checkEnvVarsInContainer(t, podItem, dynakube.OneAgentDaemonsetName(), httpsProxy)
		})

		require.NoError(t, err)
		return ctx
	}
}

func checkSampleInitContainerEnvVars(sampleApp sample.App) features.Func {
	return func(ctx context.Context, t *testing.T, environmentConfig *envconf.Config) context.Context {
		resources := environmentConfig.Client().Resources()
		pods := sampleApp.GetPods(ctx, t, resources)

		for _, podItem := range pods.Items {
			require.NotNil(t, podItem)
			require.NotNil(t, podItem.Spec)
			require.NotNil(t, podItem.Spec.InitContainers)

			checkEnvVarsInContainer(t, podItem, sampleApp.ContainerName(), dtProxy)
		}
		return ctx
	}
}

func checkEnvVarsInContainer(t *testing.T, podItem corev1.Pod, containerName string, envVar string) {
	for _, container := range podItem.Spec.Containers {
		if container.Name == containerName {
			require.NotNil(t, container.Env)
			require.True(t, kubeobjects.EnvVarIsIn(container.Env, envVar))
			for _, env := range container.Env {
				if env.Name == envVar {
					require.NotNil(t, env.Value)
				}
			}
		}
	}
}
