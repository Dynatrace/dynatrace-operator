//go:build e2e

package applicationmonitoring

import (
	"context"
	"testing"

	dynatracev1beta2 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta2/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/address"
	"github.com/Dynatrace/dynatrace-operator/test/helpers"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/sample"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/tenant"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

// ApplicationMonitoring deployment without CSI driver
func WithoutCSI(t *testing.T) features.Feature {
	builder := features.New("application monitoring without csi driver enabled")
	builder.WithLabel("name", "app-without-csi")
	secretConfig := tenant.GetSingleTenantSecret(t)
	appOnlyDynakube := *dynakube.New(
		dynakube.WithApiUrl(secretConfig.ApiUrl),
		dynakube.WithApplicationMonitoringSpec(&dynatracev1beta2.ApplicationMonitoringSpec{
			UseCSIDriver: false,
		}),
	)

	dynakube.Install(builder, helpers.LevelAssess, &secretConfig, appOnlyDynakube, false)

	sampleApp := sample.NewApp(t, &appOnlyDynakube, sample.AsDeployment())
	builder.Assess("install sample app", sampleApp.Install())
	builder.Assess("check injection of additional pod", checkInjection(sampleApp))

	podSample := sample.NewApp(t, &appOnlyDynakube,
		sample.WithName("only-pod-sample"),
	)
	builder.Assess("install additional pod", podSample.Install())
	builder.Assess("check injection of additional pod", checkInjection(podSample))

	alreadyInjectedSample := sample.NewApp(t, &appOnlyDynakube,
		sample.WithName("already-injected"),
		sample.AsDeployment(),
		sample.WithAnnotations(map[string]string{"oneagent.dynatrace.com/injected": "true"}),
	)
	builder.Assess("install already injected sample app", alreadyInjectedSample.Install())
	builder.Assess("check if pods with already injection annotation are not injected again", checkAlreadyInjected(alreadyInjectedSample))

	randomUserSample := sample.NewApp(t, &appOnlyDynakube,
		sample.WithName("random-user"),
		sample.AsDeployment(),
		sample.WithSecurityContext(corev1.PodSecurityContext{
			RunAsUser:  address.Of[int64](1234),
			RunAsGroup: address.Of[int64](1234),
		}),
	)
	builder.Assess("install sample app with random users set", randomUserSample.Install())
	builder.Assess("check injection of pods with random user", checkInjection(randomUserSample))

	builder.Teardown(sampleApp.Uninstall())
	builder.Teardown(podSample.Uninstall())
	builder.Teardown(alreadyInjectedSample.Uninstall())
	builder.Teardown(randomUserSample.Uninstall())
	dynakube.Delete(builder, helpers.LevelTeardown, appOnlyDynakube)

	return builder.Feature()
}

func checkInjection(deployment *sample.App) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		resource := envConfig.Client().Resources()
		samplePods := deployment.GetPods(ctx, t, resource)

		require.NotNil(t, samplePods)

		for _, item := range samplePods.Items {
			require.NotNil(t, item.Spec.InitContainers)
			require.Equal(t, "install-oneagent", item.Spec.InitContainers[0].Name)
		}

		return ctx
	}
}

func checkAlreadyInjected(deployment *sample.App) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		resource := envConfig.Client().Resources()
		samplePods := deployment.GetPods(ctx, t, resource)

		require.NotNil(t, samplePods)

		for _, item := range samplePods.Items {
			require.Nil(t, item.Spec.InitContainers)
		}

		return ctx
	}
}
