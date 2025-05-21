//go:build e2e

package bootstrapper

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/exp"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/oneagent"
	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/common/volumes"
	"github.com/Dynatrace/dynatrace-operator/test/helpers"
	dynakubeComponents "github.com/Dynatrace/dynatrace-operator/test/helpers/components/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/sample"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/tenant"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

func NoCSI(t *testing.T) features.Feature {
	builder := features.New("node-image-pull-with-no-csi")
	secretConfig := tenant.GetSingleTenantSecret(t)
	dk := *dynakubeComponents.New(
		dynakubeComponents.WithApiUrl(secretConfig.ApiUrl),
		dynakubeComponents.WithApplicationMonitoringSpec(&oneagent.ApplicationMonitoringSpec{AppInjectionSpec: oneagent.AppInjectionSpec{CodeModulesImage: bootstrapperImage}}),
		dynakubeComponents.WithAnnotations(map[string]string{
			exp.OANodeImagePullKey:             "true",
			exp.OANodeImagePullTechnologiesKey: "php",
		}),
	)

	dynakubeComponents.Install(builder, helpers.LevelAssess, &secretConfig, dk)

	sampleApp := sample.NewApp(t, &dk, sample.AsDeployment())
	builder.Assess("install sample app", sampleApp.Install())
	builder.Assess("check injection of sample app", checkInjection(sampleApp))

	podSample := sample.NewApp(t, &dk,
		sample.WithName("only-pod-sample"),
	)
	builder.Assess("install additional pod", podSample.Install())
	builder.Assess("check injection of additional pod", checkInjection(podSample))

	randomUserSample := sample.NewApp(t, &dk,
		sample.WithName("random-user"),
		sample.AsDeployment(),
		sample.WithSecurityContext(corev1.PodSecurityContext{
			RunAsUser:  ptr.To[int64](1234),
			RunAsGroup: ptr.To[int64](1234),
		}),
	)
	builder.Assess("install sample app with random users set", randomUserSample.Install())
	builder.Assess("check injection of pods with random user", checkInjection(randomUserSample))

	builder.Teardown(sampleApp.Uninstall())
	builder.Teardown(podSample.Uninstall())
	builder.Teardown(randomUserSample.Uninstall())
	dynakubeComponents.Delete(builder, helpers.LevelTeardown, dk)

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

			args := item.Spec.InitContainers[0].Args
			// TODO use bootstrapper repo consts in the future
			require.Contains(t, args, "--source=/opt/dynatrace/oneagent")
			require.Contains(t, args, "--target=/mnt/bin")
			require.Contains(t, args, "--config-directory=/mnt/config")
			require.Contains(t, args, "--input-directory=/mnt/input")
			require.NotContains(t, args, "--work=")
			require.NotContains(t, args, "--debug")
			require.Contains(t, args, "--technology=php")
			require.Contains(t, args, "--suppress-error")

			expectedVolume := corev1.Volume{
				Name: volumes.InputVolumeName,
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName: consts.BootstrapperInitSecretName,
					},
				},
			}

			// require.Contains doesn't work, I tried
			found := false
			for _, v := range item.Spec.Volumes {
				if v.Name == expectedVolume.Name {
					require.NotNil(t, v.Secret)
					require.Equal(t, expectedVolume.Secret.SecretName, v.Secret.SecretName)
					found = true
				}
			}
			require.True(t, found)

			if item.Spec.SecurityContext != nil {
				item.Spec.InitContainers[0].SecurityContext.RunAsUser = item.Spec.SecurityContext.RunAsUser
				item.Spec.InitContainers[0].SecurityContext.RunAsGroup = item.Spec.SecurityContext.RunAsGroup
			}
		}

		return ctx
	}
}
