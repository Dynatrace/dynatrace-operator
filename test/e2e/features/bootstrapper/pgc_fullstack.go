//go:build e2e

package bootstrapper

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-bootstrapper/pkg/configure/oneagent/pmc"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/oneagent"
	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/namespace/bootstrapperconfig"
	dynakubeComponents "github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/components/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/kubernetes/objects/k8snamespace"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/kubernetes/objects/k8spod"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/sample"
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
		samplePods := app.ListPods(ctx, t, resource)
		require.NotEmpty(t, samplePods.Items, "sample app pods should exist")

		pod := samplePods.Items[0]

		// Check init container has input files
		lsInitCommand := []string{"ls", "-la", "/mnt/input/"}
		initResult, err := k8spod.Exec(ctx, resource, pod, "install", lsInitCommand...)
		require.NoError(t, err)

		initOutput := initResult.StdOut.String()
		require.Contains(t, initOutput, pmc.InputFileName, "PMC file should be available in init container")
		require.Contains(t, initOutput, bootstrapperconfig.DeclarativeInputFileName, "PGC file should be available in init container")

		// Check app container has bootstrapper files after init container ran
		lsAppCommand := []string{"ls", "-la", "/opt/dynatrace/oneagent/"}
		appResult, err := k8spod.Exec(ctx, resource, pod, app.ContainerName(), lsAppCommand...)
		require.NoError(t, err)

		appOutput := appResult.StdOut.String()
		require.NotEmpty(t, appOutput, "app container should have oneagent files after init container ran")

		return ctx
	}
}

func checkBootstrapperSecret(app *sample.App) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		resource := envConfig.Client().Resources()
		samplePods := app.ListPods(ctx, t, resource)
		require.NotEmpty(t, samplePods.Items, "sample app pods should exist")

		namespace := samplePods.Items[0].Namespace

		var secret corev1.Secret
		require.NoError(t, resource.Get(ctx, consts.BootstrapperInitSecretName, namespace, &secret))

		if pgcData, exists := secret.Data[bootstrapperconfig.DeclarativeInputFileName]; exists {
			require.NotEmpty(t, pgcData, "PGC data should not be empty if present")
		}

		return ctx
	}
}
