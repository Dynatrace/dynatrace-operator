//go:build e2e

package classic

import (
	"context"
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/pod"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/sampleapps"
	sample "github.com/Dynatrace/dynatrace-operator/test/helpers/sampleapps/base"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/shell"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/steps/assess"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/tenant"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

func install(t *testing.T) features.Feature {
	builder := features.New("install classic fullstack")
	secretConfig := tenant.GetSingleTenantSecret(t)
	testDynakube := dynakube.NewBuilder().
		WithDefaultObjectMeta().
		ApiUrl(secretConfig.ApiUrl).
		ClassicFullstack(&dynatracev1beta1.HostInjectSpec{}).
		Build()

	sampleApp := sampleapps.NewSampleDeployment(t, testDynakube)

	// Register sample apps install and teardown
	builder.Assess("install sample app", sampleApp.Install())
	builder.Teardown(sampleApp.UninstallNamespace())

	assess.InstallDynatraceWithTeardown(builder, &secretConfig, testDynakube)

	// Register actual test
	builder.Assess("restart sample apps", sampleApp.Restart)
	builder.Assess("sample apps are injected", isAgentInjected(sampleApp))

	return builder.Feature()
}

func isAgentInjected(sampleApp sample.App) features.Func {
	return func(ctx context.Context, t *testing.T, environmentConfig *envconf.Config) context.Context {
		resources := environmentConfig.Client().Resources()
		pods := sampleApp.GetPods(ctx, t, resources)

		for _, podItem := range pods.Items {
			require.NotNil(t, podItem)

			listCommand := shell.ListDirectory("/var/lib/dynatrace")
			executionResult, err := pod.Exec(ctx, resources, podItem, sampleApp.ContainerName(), listCommand...)

			require.NoError(t, err)

			stdOut := executionResult.StdOut.String()
			stdErr := executionResult.StdErr.String()

			assert.NotEmpty(t, stdOut)
			assert.Empty(t, stdErr)
			assert.Contains(t, stdOut, "oneagent")
		}
		return ctx
	}
}
