//go:build e2e

package bootstrapper

import (
	"context"
	"testing"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta4/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta4/dynakube/oneagent"
	"github.com/Dynatrace/dynatrace-operator/test/features/cloudnative/codemodules"
	"github.com/Dynatrace/dynatrace-operator/test/helpers"
	dynakubeComponents "github.com/Dynatrace/dynatrace-operator/test/helpers/components/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/job"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/namespace"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/pod"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/sample"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/tenant"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/e2e-framework/klient/k8s"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

const (
	bootstrapperImage = "quay.io/dynatrace/dynatrace-bootstrapper:snapshot"
)

func InstallWithCSI(t *testing.T) features.Feature {
	builder := features.New("remote-image-download-with-csi")
	secretConfig := tenant.GetSingleTenantSecret(t)

	appMonDynakube := *dynakubeComponents.New(
		dynakubeComponents.WithName("app-codemodules"),
		dynakubeComponents.WithApplicationMonitoringSpec(&oneagent.ApplicationMonitoringSpec{AppInjectionSpec: oneagent.AppInjectionSpec{CodeModulesImage: bootstrapperImage}}),
		dynakubeComponents.WithAnnotations(map[string]string{dynakube.AnnotationFeatureRemoteImageDownload: "true"}),
		dynakubeComponents.WithApiUrl(secretConfig.ApiUrl),
	)

	sampleNamespace := *namespace.New("codemodules-sample-remote-image-download")
	sampleApp := sample.NewApp(t, &appMonDynakube,
		sample.AsDeployment(),
		sample.WithNamespace(sampleNamespace),
	)

	builder.Assess("create sample namespace", sampleApp.InstallNamespace())

	dynakubeComponents.Install(builder, helpers.LevelAssess, &secretConfig, appMonDynakube)

	builder.Assess("check if jobs completed", jobsAreCompleted(appMonDynakube))

	builder.Assess("check if jobs got cleaned up", jobsAreCleanedUp(appMonDynakube))

	builder.Assess("install sample app", sampleApp.Install())

	builder.Assess("codemodules have been downloaded", codemodules.ImageHasBeenDownloaded(appMonDynakube))
	builder.Assess("volumes are mounted correctly", codemodules.VolumesAreMountedCorrectly(*sampleApp))

	builder.Teardown(sampleApp.Uninstall())
	dynakubeComponents.Delete(builder, helpers.LevelTeardown, appMonDynakube)

	return builder.Feature()
}

func jobsAreCompleted(dk dynakube.DynaKube) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		resource := envConfig.Client().Resources()

		jobList := job.GetJobsForOwner(ctx, t, resource, dk.Name, dk.Namespace)

		for _, job := range jobList.Items {
			t.Logf("waiting for job to be completed: %s", job.Name)
			pods := pod.GetPodsForOwner(ctx, t, resource, job.Name, job.Namespace)
			for _, currentPod := range pods.Items {
				// let's wait not only for the pods to be deleted but rather exactly check if they really reached the state Completed
				ctx = pod.WaitForCondition(currentPod.Name, currentPod.Namespace, func(object k8s.Object) bool {
					p, isPod := object.(*corev1.Pod)

					return isPod && len(p.Status.ContainerStatuses) > 0 && p.Status.ContainerStatuses[0].State.Terminated != nil && p.Status.ContainerStatuses[0].State.Terminated.Reason == "Completed"
				}, time.Minute*5)(ctx, t, envConfig)

				ctx = pod.WaitForPodsDeletionWithOwner(job.Name, job.Namespace)(ctx, t, envConfig)
			}
		}

		return ctx
	}
}

func jobsAreCleanedUp(dk dynakube.DynaKube) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		resource := envConfig.Client().Resources()

		jobList := job.GetJobsForOwner(ctx, t, resource, dk.Name, dk.Namespace)
		require.Empty(t, jobList.Items)

		return ctx
	}
}
