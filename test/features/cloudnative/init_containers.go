//go:build e2e

package cloudnative

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/pod"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/logs"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/sample"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/shell"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

func AssessSampleInitContainers(builder *features.FeatureBuilder, sampleApp *sample.App) {
	builder.Assess("sample apps have working init containers", checkInitContainers(sampleApp))
}

func checkInitContainers(sampleApp *sample.App) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		resources := envConfig.Client().Resources()

		pods := sampleApp.GetPods(ctx, t, resources)
		require.NotEmpty(t, pods.Items)

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
					oneAgentInstallContainer = &initContainer // loop breaks after assignment, memory aliasing is not a problem

					break
				}
			}
			require.NotNil(t, oneAgentInstallContainer, "'%s' pod - '%s' container not found", podItem.Name, webhook.InstallContainerName)

			assert.Equal(t, webhook.InstallContainerName, oneAgentInstallContainer.Name)

			logStream, err := clientset.CoreV1().Pods(podItem.Namespace).GetLogs(podItem.Name, &corev1.PodLogOptions{
				Container: webhook.InstallContainerName,
			}).Stream(ctx)

			require.NoError(t, err)
			logs.AssertContains(t, logStream, "init completed")

			ifNotEmptyCommand := shell.Shell(shell.CheckIfNotEmpty("/opt/dynatrace/oneagent-paas/log/php/"))
			executionResult, err := pod.Exec(ctx, resources, podItem, sampleApp.ContainerName(), ifNotEmptyCommand...)

			require.NoError(t, err)

			stdOut := executionResult.StdOut.String()
			stdErr := executionResult.StdErr.String()

			assert.Empty(t, stdOut)
			assert.Empty(t, stdErr)
		}

		return ctx
	}
}

func DefaultCloudNativeSpec() *dynakube.CloudNativeFullStackSpec {
	return &dynakube.CloudNativeFullStackSpec{
		HostInjectSpec: dynakube.HostInjectSpec{},
	}
}
