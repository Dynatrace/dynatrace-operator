//go:build e2e

package applicationmonitoring

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/oneagent"
	webhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers"
	dynakubeComponents "github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/components/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/sample"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/tenant"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

// ApplicationMonitoring deployment without CSI driver
func WithoutCSI(t *testing.T) features.Feature {
	builder := features.New("app-monitoring-without-csi")
	secretConfig := tenant.GetSingleTenantSecret(t)
	appOnlyDynakube := *dynakubeComponents.New(
		dynakubeComponents.WithAPIURL(secretConfig.APIURL),
		dynakubeComponents.WithApplicationMonitoringSpec(&oneagent.ApplicationMonitoringSpec{}),
	)

	dynakubeComponents.Install(builder, helpers.LevelAssess, &secretConfig, appOnlyDynakube)

	sampleApp := sample.NewApp(t, &appOnlyDynakube, sample.AsDeployment())
	builder.Assess("install sample app", sampleApp.Install())
	builder.Assess("check injection of additional pod", checkInjection(sampleApp))

	podSample := sample.NewApp(t, &appOnlyDynakube,
		sample.WithName("only-pod-sample"),
	)
	builder.Assess("install additional pod", podSample.Install())
	builder.Assess("check injection of additional pod", checkInjection(podSample))

	randomUserSample := sample.NewApp(t, &appOnlyDynakube,
		sample.WithName("random-user"),
		sample.AsDeployment(),
		sample.WithPodSecurityContext(corev1.PodSecurityContext{
			RunAsUser:  ptr.To[int64](1234),
			RunAsGroup: ptr.To[int64](1234),
		}),
	)
	builder.Assess("install sample app with random users set", randomUserSample.Install())
	builder.Assess("check injection of pods with random user", checkInjection(randomUserSample))

	builder.Teardown(sampleApp.Uninstall())
	builder.Teardown(podSample.Uninstall())
	builder.Teardown(randomUserSample.Uninstall())
	dynakubeComponents.Delete(builder, helpers.LevelTeardown, appOnlyDynakube)

	return builder.Feature()
}

func checkInjection(deployment *sample.App) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		resource := envConfig.Client().Resources()
		samplePods := deployment.GetPods(ctx, t, resource)

		require.NotNil(t, samplePods)

		for _, item := range samplePods.Items {
			require.NotNil(t, item.Spec.InitContainers)
			require.Equal(t, webhook.InstallContainerName, item.Spec.InitContainers[0].Name)
		}

		return ctx
	}
}
