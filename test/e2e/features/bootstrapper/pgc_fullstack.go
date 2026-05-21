//go:build e2e

package bootstrapper

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/oneagent"
	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/namespace/bootstrapperconfig"
	dynakubeComponents "github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/components/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/kubernetes/objects/k8snamespace"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/kubernetes/objects/k8spod"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/sample"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/shell"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/tenant"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

func PGCWithCloudNativeFullStack(t *testing.T) features.Feature {
	builder := features.New("pgc-with-fullstack")
	secretConfig := tenant.GetSingleTenantSecret(t)

	fullStackSpec := &oneagent.CloudNativeFullStackSpec{
		HostInjectSpec: oneagent.HostInjectSpec{
			Image: "",
		},
		AppInjectionSpec: oneagent.AppInjectionSpec{
			CodeModulesImage: bootstrapperImage,
		},
	}

	dk := *dynakubeComponents.New(
		dynakubeComponents.WithAPIURL(secretConfig.APIURL),
		dynakubeComponents.WithCloudNativeSpec(fullStackSpec),
	)

	sampleNamespace := *k8snamespace.New("pgc-fullstack-sample")
	sampleApp := sample.NewApp(t, &dk,
		sample.WithNamespace(sampleNamespace),
		sample.AsDeployment(),
	)

	builder.Assess("create sample namespace", sampleApp.InstallNamespace())
	dynakubeComponents.Install(builder, &secretConfig, dk)
	builder.Assess("install sample app", sampleApp.Install())
	builder.Assess("check bootstrapper secret has PGC data", checkBootstrapperSecret(sampleApp))
	builder.Assess("verify bootstrapper files mounted in pod", verifyBootstrapperFilesMounted(sampleApp))

	builder.Teardown(sampleApp.Uninstall())

	return builder.Feature()
}

func verifyBootstrapperFilesMounted(app *sample.App) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		resource := envConfig.Client().Resources()
		pod := app.GetPod(ctx, t, resource)

		listCommand := shell.ListDirectory("/var/lib/dynatrace/oneagent/agent/config")
		appResult, err := k8spod.Exec(ctx, resource, pod, app.ContainerName(), listCommand...)
		require.NoError(t, err)

		appOutput := appResult.StdOut.String()
		require.NotEmpty(t, appOutput, "app container should have config files after init container ran")
		require.Contains(t, appOutput, bootstrapperconfig.DeclarativeInputFileName, "PGC file should be present in config directory")

		return ctx
	}
}

func checkBootstrapperSecret(app *sample.App) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		resource := envConfig.Client().Resources()
		app.GetPod(ctx, t, resource)

		namespace := app.Namespace()

		var secret corev1.Secret
		require.NoError(t, resource.Get(ctx, consts.BootstrapperInitSecretName, namespace, &secret))

		if pgcData, exists := secret.Data[bootstrapperconfig.DeclarativeInputFileName]; exists {
			require.NotEmpty(t, pgcData, "PGC data should not be empty if present")
		}

		return ctx
	}
}
