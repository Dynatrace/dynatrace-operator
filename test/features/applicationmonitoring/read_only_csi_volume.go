//go:build e2e

package applicationmonitoring

import (
	"context"
	"fmt"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	oamutation "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/oneagent"
	"github.com/Dynatrace/dynatrace-operator/test/helpers"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/codemodules"
	dynakubeComponents "github.com/Dynatrace/dynatrace-operator/test/helpers/components/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/deployment"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/pod"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/sample"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/shell"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/tenant"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

var readOnlyInjection = map[string]string{dynakube.AnnotationFeatureReadOnlyCsiVolume: "true"}

func ReadOnlyCSIVolume(t *testing.T) features.Feature {
	builder := features.New("app-read-only-csi-volume")
	secretConfig := tenant.GetSingleTenantSecret(t)
	testDynakube := *dynakubeComponents.New(
		dynakubeComponents.WithAnnotations(readOnlyInjection),
		dynakubeComponents.WithApiUrl(secretConfig.ApiUrl),
		dynakubeComponents.WithApplicationMonitoringSpec(&dynakube.ApplicationMonitoringSpec{
			UseCSIDriver: true,
		}),
	)
	sampleDeployment := sample.NewApp(t, &testDynakube, sample.AsDeployment())
	builder.Assess("install sample deployment namespace", sampleDeployment.InstallNamespace())

	dynakubeComponents.Install(builder, helpers.LevelAssess, &secretConfig, testDynakube)

	builder.Assess("install sample deployment and wait till ready", sampleDeployment.Install())
	builder.Assess("check mounted volumes", checkMountedVolumes(sampleDeployment))
	builder.Assess(fmt.Sprintf("check %s has no conn info", codemodules.RuxitAgentProcFile), codemodules.CheckRuxitAgentProcFileHasNoConnInfo(testDynakube))

	builder.WithTeardown("removing sample namespace", sampleDeployment.Uninstall())

	dynakubeComponents.Delete(builder, helpers.LevelTeardown, testDynakube)

	return builder.Feature()
}

func checkMountedVolumes(sampleApp *sample.App) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		resources := envConfig.Client().Resources()

		err := deployment.NewQuery(ctx, resources, client.ObjectKey{
			Name:      sampleApp.Name(),
			Namespace: sampleApp.Namespace(),
		}).ForEachPod(func(podItem corev1.Pod) {
			listCommand := shell.ListDirectory(oamutation.OneAgentConfMountPath)
			result, err := pod.Exec(ctx, resources, podItem, sampleApp.ContainerName(), listCommand...)

			require.NoError(t, err)
			assert.Contains(t, result.StdOut.String(), codemodules.RuxitAgentProcFile)
		})
		require.NoError(t, err)

		return ctx
	}
}
