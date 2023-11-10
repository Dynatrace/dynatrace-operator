//go:build e2e

package network_problems

import (
	"context"
	"path"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/webhook"
	"github.com/Dynatrace/dynatrace-operator/test/features/cloudnative"
	"github.com/Dynatrace/dynatrace-operator/test/helpers"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/manifests"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/namespace"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/pod"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/sample"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/shell"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/tenant"
	"github.com/Dynatrace/dynatrace-operator/test/project"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

const (
	ldPreloadError = "ERROR: ld.so: object '/opt/dynatrace/oneagent-paas/agent/lib64/liboneagentproc.so' from LD_PRELOAD cannot be preloaded"
)

var (
	csiNetworkPolicy = path.Join(project.TestDataDir(), "network/csi-denial.yaml")
)

func ResilienceFeature(t *testing.T) features.Feature {
	builder := features.New("cloudnative resilience in case of network problems")
	builder.WithLabel("name", "cloudnative-csi-resilience")
	secretConfig := tenant.GetSingleTenantSecret(t)

	restrictCSI(builder)

	testDynakube := *dynakube.New(
		dynakube.WithApiUrl(secretConfig.ApiUrl),
		dynakube.WithCloudNativeSpec(cloudnative.DefaultCloudNativeSpec()),
		dynakube.WithAnnotations(map[string]string{
			"feature.dynatrace.com/max-csi-mount-attempts": "2",
		}),
	)

	sampleNamespace := *namespace.New("network-problem-sample")
	sampleApp := sample.NewApp(t, &testDynakube,
		sample.AsDeployment(),
		sample.WithNamespace(sampleNamespace),
	)
	builder.Assess("create sample namespace", sampleApp.InstallNamespace())

	dynakube.Install(builder, helpers.LevelAssess, &secretConfig, testDynakube)

	builder.Assess("install sample-apps", sampleApp.Install())
	builder.Assess("check for dummy volume", checkForDummyVolume(sampleApp))

	builder.Teardown(sampleApp.Uninstall())
	dynakube.Delete(builder, helpers.LevelTeardown, testDynakube)
	return builder.Feature()
}

func restrictCSI(builder *features.FeatureBuilder) {
	builder.Assess("restrict csi-driver", helpers.ToFeatureFunc(manifests.InstallFromFile(csiNetworkPolicy), true))
	builder.Teardown(helpers.ToFeatureFunc(manifests.UninstallFromFile(csiNetworkPolicy), true))
}

func checkForDummyVolume(sampleApp *sample.App) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		resources := envConfig.Client().Resources()
		pods := sampleApp.GetPods(ctx, t, resources)

		for _, podItem := range pods.Items {
			require.NotNil(t, podItem)
			require.NotNil(t, podItem.Spec)
			require.NotEmpty(t, podItem.Spec.InitContainers)

			listCommand := shell.ListDirectory(webhook.DefaultInstallPath)
			result, err := pod.Exec(ctx, resources, podItem, sampleApp.ContainerName(), listCommand...)

			require.NoError(t, err)
			assert.Contains(t, result.StdErr.String(), ldPreloadError)
		}
		return ctx
	}
}
