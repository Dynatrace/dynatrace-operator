//go:build e2e

package cloudnative

import (
	"context"
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/webhook"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/istio"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/namespace"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/pod"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/logs"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/sampleapps"
	sample "github.com/Dynatrace/dynatrace-operator/test/helpers/sampleapps/base"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/shell"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/steps/assess"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/steps/teardown"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/tenant"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

func Install(t *testing.T, istioEnabled bool) features.Feature {
	builder := features.New("default installation")
	t.Logf("istio enabled: %v", istioEnabled)
	secretConfig := tenant.GetSingleTenantSecret(t)

	dynakubeBuilder := dynakube.NewBuilder().
		WithDefaultObjectMeta().
		ApiUrl(secretConfig.ApiUrl).
		CloudNative(defaultCloudNativeSpec())
	if istioEnabled {
		dynakubeBuilder = dynakubeBuilder.WithIstio()
	}
	testDynakube := dynakubeBuilder.Build()

	// Register operator install
	operatorNamespaceBuilder := namespace.NewBuilder(testDynakube.Namespace)
	if istioEnabled {
		operatorNamespaceBuilder = operatorNamespaceBuilder.WithLabels(istio.InjectionLabel)
	}
	assess.InstallOperatorFromSourceWithCustomNamespace(builder, operatorNamespaceBuilder.Build(), testDynakube)

	// Register sample app install
	namespaceBuilder := namespace.NewBuilder("cloudnative-sample")
	if istioEnabled {
		namespaceBuilder = namespaceBuilder.WithLabels(istio.InjectionLabel)
	}
	sampleNamespace := namespaceBuilder.Build()
	sampleApp := sampleapps.NewSampleDeployment(t, testDynakube)
	sampleApp.WithNamespace(sampleNamespace)

	// Register dynakube install
	assess.InstallDynakube(builder, &secretConfig, testDynakube)

	// Register sample app install
	builder.Assess("install sample app", sampleApp.Install())

	// Register actual test
	assessSampleInitContainers(builder, sampleApp)
	if istioEnabled {
		istio.AssessIstio(builder, testDynakube, sampleApp)
	}

	// Register sample, dynakube and operator uninstall
	builder.Teardown(sampleApp.UninstallNamespace())
	teardown.UninstallDynatrace(builder, testDynakube)

	return builder.Feature()
}

func assessSampleAppsRestartHalf(builder *features.FeatureBuilder, sampleApp sample.App) {
	builder.Assess("restart half of sample apps", sampleApp.RestartHalf)
}

func assessSampleInitContainers(builder *features.FeatureBuilder, sampleApp sample.App) {
	builder.Assess("sample apps have working init containers", checkInitContainers(sampleApp))
}

func checkInitContainers(sampleApp sample.App) features.Func {
	return func(ctx context.Context, t *testing.T, environmentConfig *envconf.Config) context.Context {
		resources := environmentConfig.Client().Resources()
		pods := sampleApp.GetPods(ctx, t, resources)
		clientset, err := kubernetes.NewForConfig(resources.GetConfig())

		require.NoError(t, err)

		for _, podItem := range pods.Items {
			if podItem.DeletionTimestamp != nil {
				continue
			}

			require.NotNil(t, podItem)
			require.NotNil(t, podItem.Spec)
			require.NotEmpty(t, podItem.Spec.InitContainers)

			var oneAgentInstallContainer *corev1.Container

			for _, initContainer := range podItem.Spec.InitContainers {
				if initContainer.Name == webhook.InstallContainerName {
					oneAgentInstallContainer = &initContainer //nolint:gosec // loop breaks after assignment, memory aliasing is not a problem
					break
				}
			}
			require.NotNil(t, oneAgentInstallContainer, "'%s' pod - '%s' container not found", podItem.Name, webhook.InstallContainerName)

			assert.Equal(t, webhook.InstallContainerName, oneAgentInstallContainer.Name)

			logStream, err := clientset.CoreV1().Pods(podItem.Namespace).GetLogs(podItem.Name, &corev1.PodLogOptions{
				Container: webhook.InstallContainerName,
			}).Stream(ctx)

			require.NoError(t, err)
			logs.AssertContains(t, logStream, "standalone agent init completed")

			ifEmptyCommand := shell.CheckIfEmpty("/opt/dynatrace/oneagent-paas/log/php/")
			executionResult, err := pod.Exec(ctx, resources, podItem, sampleApp.ContainerName(), ifEmptyCommand...)

			require.NoError(t, err)

			stdOut := executionResult.StdOut.String()
			stdErr := executionResult.StdErr.String()

			assert.Empty(t, stdOut)
			assert.Empty(t, stdErr)
		}

		return ctx
	}
}

func defaultCloudNativeSpec() *dynatracev1beta1.CloudNativeFullStackSpec {
	return &dynatracev1beta1.CloudNativeFullStackSpec{
		HostInjectSpec: dynatracev1beta1.HostInjectSpec{},
	}
}
