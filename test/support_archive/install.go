//go:build e2e

package support_archive

import (
	"context"
	"github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/test/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/kubeobjects/pod"
	"github.com/Dynatrace/dynatrace-operator/test/operator"
	"github.com/Dynatrace/dynatrace-operator/test/secrets"
	"github.com/Dynatrace/dynatrace-operator/test/setup"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"strings"
	"testing"

	"sigs.k8s.io/e2e-framework/pkg/features"
)

func installOperatorAndDynakube(t *testing.T) features.Feature {
	secretConfig := getSecretConfig(t)

	defaultInstallation := features.New("default installation")
	setup.InstallOperator(defaultInstallation, secretConfig)
	setup.AssessOperatorDeployment(defaultInstallation)

	defaultInstallation.Assess("dynakube applied", dynakube.Apply(
		dynakube.NewBuilder().
			WithDefaultObjectMeta().
			ApiUrl(secretConfig.ApiUrl).
			CloudNative(&v1beta1.CloudNativeFullStackSpec{}).
			Build()))

	setup.AssessDynakubeStartup(defaultInstallation)

	defaultInstallation.Assess("execute troubleshoot", executeTroubleshoot)

	//	assessOneAgentsAreRunning(defaultInstallation)

	return defaultInstallation.Feature()
}

func getSecretConfig(t *testing.T) secrets.Secret {
	secretConfig, err := secrets.DefaultSingleTenant(afero.NewOsFs())
	require.NoError(t, err)
	return secretConfig
}

func executeTroubleshoot(ctx context.Context, t *testing.T, environmentConfig *envconf.Config) context.Context {
	resources := environmentConfig.Client().Resources()
	pods := operator.Get(t, ctx, resources)
	for _, podItem := range pods.Items {
		require.NotNil(t, podItem)

		if strings.Contains(podItem.Name, "dynatrace-operator") {
			executionQuery := pod.NewExecutionQuery(podItem, "dynatrace-operator", "/usr/local/bin/dynatrace-operator troubleshoot")
			executionResult, err := executionQuery.Execute(environmentConfig.Client().RESTConfig())
			require.NoError(t, err)

			stdOut := executionResult.StdOut.String()
			stdErr := executionResult.StdErr.String()

			t.Logf("Troubleshoot execution STDOUT:\n%s", stdOut)
			t.Logf("Troubleshoot execution STDERR:\n%s", stdErr)

			return ctx
		}
	}
	t.Errorf("troubleshoot command hasn't been executed")
	return ctx
}
