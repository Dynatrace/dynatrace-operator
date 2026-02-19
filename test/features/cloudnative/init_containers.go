//go:build e2e

package cloudnative

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-bootstrapper/cmd/k8sinit"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/oneagent"
	webhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubernetes/objects/k8spod"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/sample"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/shell"
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

		for _, pod := range pods.Items {
			if pod.DeletionTimestamp != nil {
				continue
			}

			require.NotEmpty(t, pod.Spec.InitContainers)

			var oneAgentInstallInitContainer *corev1.Container

			for _, initContainer := range pod.Spec.InitContainers {
				if initContainer.Name == webhook.InstallContainerName {
					oneAgentInstallInitContainer = &initContainer // loop breaks after assignment, memory aliasing is not a problem

					break
				}
			}
			require.NotNil(t, oneAgentInstallInitContainer, "init container not found in '%s' pod", pod.Name)

			if !sampleApp.CanInitError() {
				assert.Contains(t, oneAgentInstallInitContainer.Args, "--"+k8sinit.SuppressErrorsFlag, "errors may be suppressed, further checks are not useful")

				continue
			}

			assert.NotContains(t, oneAgentInstallInitContainer.Args, "--"+k8sinit.SuppressErrorsFlag, "in the tests the init-container should have no errors suppressed")

			ifNotEmptyCommand := shell.Shell(shell.CheckIfNotEmpty("/var/lib/dynatrace/oneagent/log/php/"))
			executionResult, err := k8spod.Exec(ctx, resources, pod, sampleApp.ContainerName(), ifNotEmptyCommand...)
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
