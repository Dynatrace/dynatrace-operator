//go:build e2e

package classic

import (
	"context"
	"path"
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/test/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/kubeobjects/pod"
	"github.com/Dynatrace/dynatrace-operator/test/oneagent"
	"github.com/Dynatrace/dynatrace-operator/test/operator"
	"github.com/Dynatrace/dynatrace-operator/test/project"
	"github.com/Dynatrace/dynatrace-operator/test/sampleapps"
	"github.com/Dynatrace/dynatrace-operator/test/setup"
	"github.com/Dynatrace/dynatrace-operator/test/shell"
	"github.com/Dynatrace/dynatrace-operator/test/webhook"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

func install(t *testing.T) features.Feature {
	installClassicFullStack := features.New("install classic fullstack")
	secretConfig := getSecretConfig(t)

	setup.InstallDynatraceFromSource(installClassicFullStack, &secretConfig)
	setup.DeploySampleApps(installClassicFullStack, path.Join(project.TestDataDir(), "classic-fullstack/sample-deployment.yaml"))

	installClassicFullStack.Assess("operator started", operator.WaitForDeployment())
	installClassicFullStack.Assess("webhook started", webhook.WaitForDeployment())
	installClassicFullStack.Assess("install dynakube", dynakube.Apply(
		dynakube.NewBuilder().
			WithDefaultObjectMeta().
			ApiUrl(secretConfig.ApiUrl).
			ClassicFullstack(&dynatracev1beta1.HostInjectSpec{
				Env: []v1.EnvVar{
					{
						Name:  "ONEAGENT_ENABLE_VOLUME_STORAGE",
						Value: "true",
					},
				},
			}).
			Build()),
	)

	setup.AssessDynakubeStartup(installClassicFullStack)

	installClassicFullStack.Assess("os agent can connect", oneagent.OSAgentCanConnect())
	installClassicFullStack.Assess("restart sample apps", sampleapps.Restart)
	installClassicFullStack.Assess("sample apps are injected", isAgentInjected)

	return installClassicFullStack.Feature()
}

func isAgentInjected(ctx context.Context, t *testing.T, environmentConfig *envconf.Config) context.Context {
	resources := environmentConfig.Client().Resources()
	pods := pod.List(t, ctx, resources, sampleapps.Namespace)

	for _, podItem := range pods.Items {
		require.NotNil(t, podItem)

		executionQuery := pod.NewExecutionQuery(podItem, sampleapps.Name, shell.ListDirectory("/var/lib/dynatrace")...)
		executionResult, err := executionQuery.Execute(environmentConfig.Client().RESTConfig())

		require.NoError(t, err)

		stdOut := executionResult.StdOut.String()
		stdErr := executionResult.StdErr.String()

		assert.NotEmpty(t, stdOut)
		assert.Empty(t, stdErr)
		assert.Contains(t, stdOut, "oneagent")
	}
	return ctx
}
