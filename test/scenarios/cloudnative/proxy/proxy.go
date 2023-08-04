//go:build e2e

package cloudnativeproxy

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1/dynakube"
	"github.com/Dynatrace/dynatrace-operator/src/installer/common"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	"github.com/Dynatrace/dynatrace-operator/src/webhook/mutation/pod_mutator/oneagent_mutation"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/istio"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/daemonset"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/deployment"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/namespace"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/pod"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/proxy"
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

const (
	httpsProxy = "https_proxy"
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

	sampleLabels := kubeobjects.MergeMap(testDynakube.NamespaceSelector().MatchLabels, istio.InjectionLabel)
	sampleNamespace := namespace.NewBuilder("proxy-sample").WithLabels(sampleLabels).Build()
	sampleApp := sampleapps.NewSampleDeployment(t, testDynakube)
	sampleApp.WithLabels(sampleLabels)
	sampleApp.WithNamespace(sampleNamespace)

	// Register sample namespace create and delete
	builder.Assess("create sample namespace", namespace.Create(sampleNamespace))
	builder.Teardown(sampleApp.UninstallNamespace())

	// Register operator install
	operatorNamespaceBuilder := namespace.NewBuilder(testDynakube.Namespace)
	if proxySpec != nil {
		operatorNamespaceBuilder = operatorNamespaceBuilder.WithLabels(istio.InjectionLabel)
	}
	assess.InstallOperatorFromSourceWithCustomNamespace(builder, operatorNamespaceBuilder.Build(), testDynakube)

	// Register proxy create and delete
	proxy.SetupProxyWithTeardown(t, builder, testDynakube)
	proxy.CutOffDynatraceNamespace(builder, proxySpec)
	proxy.IsDynatraceNamespaceCutOff(builder, testDynakube)

	// Register dynakube install
	assess.InstallDynakube(builder, &secretConfig, testDynakube)
	// Register sample app install
	builder.Assess("install sample app", sampleApp.Install())

	// Register actual test
	builder.Assess("check env variables of oneagent pods", checkOneAgentEnvVars(testDynakube))
	builder.Assess("check proxy settings in ruxitagentproc.conf", checkRuxitAgentProcFileHasProxySetting(sampleApp, proxySpec))

	// Register operator and dynakube uninstall
	teardown.UninstallDynatrace(builder, testDynakube)

	return builder.Feature()
}

func checkOneAgentEnvVars(dynakube dynatracev1beta1.DynaKube) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		resources := envConfig.Client().Resources()
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

func checkRuxitAgentProcFileHasProxySetting(sampleApp sample.App, proxySpec *dynatracev1beta1.DynaKubeProxy) features.Func {
	return func(ctx context.Context, t *testing.T, e *envconf.Config) context.Context {
		resources := e.Client().Resources()

		err := deployment.NewQuery(ctx, resources, client.ObjectKey{
			Name:      sampleApp.Name(),
			Namespace: sampleApp.Namespace().Name,
		}).ForEachPod(func(podItem corev1.Pod) {
			dir := filepath.Join(oneagent_mutation.OneAgentConfMountPath, common.RuxitConfFileName)
			readFileCommand := shell.ReadFile(dir)
			result, err := pod.Exec(ctx, resources, podItem, "sample-dynakube", readFileCommand...)
			assert.Contains(t, result.StdOut.String(), fmt.Sprintf("proxy %s", proxySpec.Value))
			require.NoError(t, err)
		})

		require.NoError(t, err)
		return ctx
	}
}
