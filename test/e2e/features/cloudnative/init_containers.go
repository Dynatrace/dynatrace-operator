//go:build e2e

package cloudnative

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-bootstrapper/cmd"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/oneagent"
	webhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/kubeobjects/pod"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/sample"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/shell"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
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

		for _, podItem := range pods.Items {
			if podItem.DeletionTimestamp != nil {
				continue
			}

			require.NotNil(t, podItem)
			require.NotNil(t, podItem.Spec)
			require.NotEmpty(t, podItem.Spec.InitContainers)

			var oneAgentInstallInitContainer *corev1.Container

			for _, initContainer := range podItem.Spec.InitContainers {
				if initContainer.Name == webhook.InstallContainerName {
					oneAgentInstallInitContainer = &initContainer // loop breaks after assignment, memory aliasing is not a problem

					break
				}
			}
			require.NotNil(t, oneAgentInstallInitContainer, "init container not found in '%s' pod", podItem.Name)

			if !sampleApp.CanInitError() {
				assert.Contains(t, oneAgentInstallInitContainer.Args, "--"+cmd.SuppressErrorsFlag, "errors may be suppressed, further checks are not useful")

				continue
			}

			assert.NotContains(t, oneAgentInstallInitContainer.Args, "--"+cmd.SuppressErrorsFlag, "in the tests the init-container should have no errors suppressed")

			ifNotEmptyCommand := shell.Shell(shell.CheckIfNotEmpty("/var/lib/dynatrace/oneagent/log/php/"))
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

func DefaultCloudNativeSpec() *oneagent.CloudNativeFullStackSpec {
	return &oneagent.CloudNativeFullStackSpec{
		HostInjectSpec: oneagent.HostInjectSpec{},
	}
}
