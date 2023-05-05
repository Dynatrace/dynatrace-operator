//go:build e2e

package classic

import (
	"context"
	"strings"
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/pod"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/sampleapps"
	sample "github.com/Dynatrace/dynatrace-operator/test/helpers/sampleapps/base"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/shell"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/steps/assess"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/tenant"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/klient/wait"
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

	assess.InstallDynatraceWithTeardown(builder, &secretConfig, testDynakube)

	// Register actual test
	builder.Assess("install sample app", sampleApp.Install())
	builder.Teardown(sampleApp.UninstallNamespace())
	builder.Assess("sample apps are injected", isAgentInjected(sampleApp))

	return builder.Feature()
}

func isAgentInjected(sampleApp sample.App) features.Func {
	return func(ctx context.Context, t *testing.T, environmentConfig *envconf.Config) context.Context {
		resources := environmentConfig.Client().Resources()
		pods := sampleApp.GetPods(ctx, t, resources)
		require.NoError(t, wait.For(classicInjected(ctx, resources, sampleApp, pods.Items)))
		return ctx
	}
}

func classicInjected(ctx context.Context, resources *resources.Resources, sampleApp sample.App, pods []corev1.Pod) func() (done bool, err error) {
	return func() (done bool, err error) {
		if pods == nil {
			return false, nil
		}
		for _, podItem := range pods {
			listCommand := shell.ListDirectory("/var/lib")
			executionResult, err := pod.Exec(ctx, resources, podItem, sampleApp.ContainerName(), listCommand...)
			if err != nil {
				return false, err
			}

			stdOut := executionResult.StdOut.String()
			if !strings.Contains(stdOut, "dynatrace") {
				return false, nil
			}
		}
		return true, nil
	}
}
