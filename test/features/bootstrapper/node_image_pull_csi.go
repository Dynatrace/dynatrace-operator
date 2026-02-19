//go:build e2e

package bootstrapper

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/exp"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/oneagent"
	"github.com/Dynatrace/dynatrace-operator/test/features/cloudnative/codemodules"
	"github.com/Dynatrace/dynatrace-operator/test/helpers"
	dynakubeComponents "github.com/Dynatrace/dynatrace-operator/test/helpers/components/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubernetes/objects/k8sjob"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubernetes/objects/k8snamespace"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubernetes/objects/k8spod"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/sample"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/tenant"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

const (
	bootstrapperImage = "ghcr.io/dynatrace/dynatrace-bootstrapper:snapshot"
)

func InstallWithCSI(t *testing.T) features.Feature {
	builder := features.New("node-image-pull-with-csi")
	secretConfig := tenant.GetSingleTenantSecret(t)

	appMonDynakube := *dynakubeComponents.New(
		dynakubeComponents.WithName("app-codemodules"),
		dynakubeComponents.WithApplicationMonitoringSpec(&oneagent.ApplicationMonitoringSpec{AppInjectionSpec: oneagent.AppInjectionSpec{CodeModulesImage: bootstrapperImage}}),
		dynakubeComponents.WithAnnotations(map[string]string{exp.OANodeImagePullKey: "true"}),
		dynakubeComponents.WithAPIURL(secretConfig.APIURL),
	)

	sampleNamespace := *k8snamespace.New("codemodules-sample-node-image-pull")
	sampleApp := sample.NewApp(t, &appMonDynakube,
		sample.AsDeployment(),
		sample.WithNamespace(sampleNamespace),
	)

	builder.Assess("create sample namespace", sampleApp.InstallNamespace())

	dynakubeComponents.Install(builder, helpers.LevelAssess, &secretConfig, appMonDynakube)

	builder.Assess("check if jobs completed", jobsAreCompleted(appMonDynakube))

	builder.Assess("check if jobs got cleaned up", k8sjob.WaitForDeletionWithOwner(appMonDynakube.Name, appMonDynakube.Namespace))

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

		jobList := k8sjob.ListForOwner(ctx, t, resource, dk.Name, dk.Namespace)
		require.NotEmpty(t, jobList.Items)

		for _, job := range jobList.Items {
			t.Logf("waiting for job to be completed: %s", job.Name)
			ctx = k8spod.WaitForDeletionWithOwner(job.Name, job.Namespace)(ctx, t, envConfig)
		}

		return ctx
	}
}
