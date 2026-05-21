//go:build e2e

package bootstrapper

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-bootstrapper/pkg/configure/oneagent/pmc"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/exp"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/oneagent"
	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/namespace/bootstrapperconfig"
	webhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/volumes"
	dynakubeComponents "github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/components/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/platform"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/sample"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/tenant"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

func NoCSI(t *testing.T) features.Feature {
	builder := features.New("node-image-pull-with-no-csi")
	secretConfig := tenant.GetSingleTenantSecret(t)
	dk := *dynakubeComponents.New(
		dynakubeComponents.WithAPIURL(secretConfig.APIURL),
		dynakubeComponents.WithApplicationMonitoringSpec(&oneagent.ApplicationMonitoringSpec{AppInjectionSpec: oneagent.AppInjectionSpec{CodeModulesImage: bootstrapperImage}}),
		dynakubeComponents.WithAnnotations(map[string]string{
			exp.OANodeImagePullTechnologiesKey: "php",
		}),
	)

	dynakubeComponents.Install(builder, &secretConfig, dk)

	sampleApp := sample.NewApp(t, &dk,
		sample.AsDeployment(),
		sample.WithPodSecurityContext(corev1.PodSecurityContext{}),
		sample.WithoutClusterRole(),
	)
	builder.Assess("install sample app", sampleApp.Install())
	builder.Assess("check injection of sample app", checkInjection(sampleApp))
	builder.Assess("check bootstrapper secret has PMC and PGC data", checkBootstrapperSecret(sampleApp))

	podSample := sample.NewApp(t, &dk,
		sample.WithName("only-pod-sample"),
	)
	builder.Assess("install additional pod", podSample.Install())
	builder.Assess("check injection of additional pod", checkInjection(podSample))

	randomUserSample := sample.NewApp(t, &dk,
		sample.WithName("random-user"),
		sample.AsDeployment(),
		sample.WithPodSecurityContext(corev1.PodSecurityContext{
			RunAsUser:  new(int64(1234)),
			RunAsGroup: new(int64(1234)),
		}),
		sample.WithContainerSecurityContext(
			corev1.SecurityContext{
				SeccompProfile: &corev1.SeccompProfile{
					Type: corev1.SeccompProfileTypeRuntimeDefault,
				},
				AllowPrivilegeEscalation: new(false),
				RunAsNonRoot:             new(true),
				Capabilities: &corev1.Capabilities{
					Drop: []corev1.Capability{"ALL"},
				},
			}),
	)
	builder.Assess("install sample app with random users set", randomUserSample.Install())
	builder.Assess("check injection of pods with random user", checkInjection(randomUserSample))

	isOpenshift, err := platform.NewResolver().IsOpenshift()
	require.NoError(t, err)
	if isOpenshift {
		randomUserSampleFail := sample.NewApp(t, &dk,
			sample.WithName("random-user-fail"),
			sample.AsDeployment(),
			sample.WithPodSecurityContext(corev1.PodSecurityContext{
				RunAsUser:  new(int64(1234)),
				RunAsGroup: new(int64(1234)),
			}),
			sample.WithContainerSecurityContext(corev1.SecurityContext{
				SeccompProfile: &corev1.SeccompProfile{
					Type: corev1.SeccompProfileTypeRuntimeDefault,
				},
				AllowPrivilegeEscalation: new(false),
				RunAsNonRoot:             new(true),
				Capabilities: &corev1.Capabilities{
					Drop: []corev1.Capability{"ALL"},
				},
			}),
			sample.WithoutClusterRole(),
		)
		builder.Assess("try to install sample app with random users set", randomUserSampleFail.InstallFail())

		builder.Teardown(randomUserSampleFail.UninstallFail())
	}

	builder.Teardown(sampleApp.Uninstall())
	builder.Teardown(podSample.Uninstall())
	builder.Teardown(randomUserSample.Uninstall())

	return builder.Feature()
}

func checkInjection(deployment *sample.App) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		resource := envConfig.Client().Resources()
		samplePods := deployment.ListPods(ctx, t, resource)

		require.NotNil(t, samplePods)

		for _, item := range samplePods.Items {
			require.NotNil(t, item.Spec.InitContainers)
			require.Equal(t, webhook.InstallContainerName, item.Spec.InitContainers[0].Name)

			args := item.Spec.InitContainers[0].Args
			require.Contains(t, args, "--source=/opt/dynatrace/oneagent")
			require.Contains(t, args, "--target=/mnt/bin")
			require.Contains(t, args, "--config-directory=/mnt/config")
			require.Contains(t, args, "--input-directory=/mnt/input")
			require.NotContains(t, args, "--work=")
			require.NotContains(t, args, "--debug")
			require.Contains(t, args, "--technology=php")

			expectedVolume := corev1.Volume{
				Name: volumes.InputVolumeName,
				VolumeSource: corev1.VolumeSource{
					Projected: &corev1.ProjectedVolumeSource{
						Sources: []corev1.VolumeProjection{
							{
								Secret: &corev1.SecretProjection{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: consts.BootstrapperInitSecretName,
									},
									Optional: new(false),
								},
							},
							{
								Secret: &corev1.SecretProjection{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: consts.BootstrapperInitCertsSecretName,
									},
									Optional: new(true),
								},
							},
						},
					},
				},
			}

			// require.Contains doesn't work, I tried
			found := false
			for _, v := range item.Spec.Volumes {
				if v.Name == expectedVolume.Name {
					require.Equal(t, expectedVolume.Projected.Sources, v.Projected.Sources)
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

func checkBootstrapperSecret(app *sample.App) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		resource := envConfig.Client().Resources()
		samplePods := app.ListPods(ctx, t, resource)
		require.NotEmpty(t, samplePods.Items, "sample app pods should exist")

		namespace := samplePods.Items[0].Namespace

		var secret corev1.Secret
		require.NoError(t, resource.Get(ctx, consts.BootstrapperInitSecretName, namespace, &secret))

		require.NotEmpty(t, secret.Data[pmc.InputFileName], "PMC data should be present in bootstrapper secret")

		if pgcData, exists := secret.Data[bootstrapperconfig.DeclarativeInputFileName]; exists {
			require.NotEmpty(t, pgcData, "PGC data should not be empty if present")
		}

		return ctx
	}
}

func assertPodHasProjectedInputVolume(t *testing.T, pod *corev1.Pod) {
	for _, v := range pod.Spec.Volumes {
		if v.Name == volumes.InputVolumeName {
			require.NotNil(t, v.Projected, "input volume should be projected")
			require.Len(t, v.Projected.Sources, 2, "projected volume should have 2 sources")

			// Check bootstrapper config secret (required)
			require.NotNil(t, v.Projected.Sources[0].Secret, "first source should be secret")
			require.Equal(t, consts.BootstrapperInitSecretName, v.Projected.Sources[0].Secret.Name)
			require.False(t, *v.Projected.Sources[0].Secret.Optional)

			// Check bootstrapper certs secret (optional)
			require.NotNil(t, v.Projected.Sources[1].Secret, "second source should be secret")
			require.Equal(t, consts.BootstrapperInitCertsSecretName, v.Projected.Sources[1].Secret.Name)
			require.True(t, *v.Projected.Sources[1].Secret.Optional)

			return
		}
	}
	t.Error("pod should have input volume")
}
