//go:build e2e

package network_problems

import (
	"bytes"
	"context"
	"io"
	"path"
	"strings"
	"testing"
	"time"

	"github.com/Dynatrace/dynatrace-operator/test/features/cloudnative"
	"github.com/Dynatrace/dynatrace-operator/test/helpers"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/manifests"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/namespace"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/sample"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/tenant"
	"github.com/Dynatrace/dynatrace-operator/test/project"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

const (
	ldPreloadError = "ERROR: ld.so: object '/opt/dynatrace/oneagent-paas/agent/lib64/liboneagentproc.so' from LD_PRELOAD cannot be preloaded"
)

var (
	csiNetworkPolicy = path.Join(project.TestDataDir(), "network/csi-denial.yaml")
)

// Prerequisites: istio service mesh
//
// Setup: CloudNative deployment with CSI driver
//
// Verification that the CSI driver is able to recover from network issues, when
// using cloudNative and code modules image.
//
// Connectivity for csi driver pods is restricted to the local k8s cluster (no
// outside connections allowed) and sample application is installed. The test
// checks if init container was attached, run successfully and that the sample
// pods are up and running.
func ResilienceFeature(t *testing.T) features.Feature {
	builder := features.New("cloudnative-csi-resilience")
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
		clientset, err := kubernetes.NewForConfig(resources.GetConfig())
		require.NoError(t, err)

		pods := sampleApp.GetPods(ctx, t, resources)

		for _, podItem := range pods.Items {
			require.NotNil(t, podItem)
			require.NotNil(t, podItem.Spec)
			require.NotEmpty(t, podItem.Spec.InitContainers)

			err = wait.For(func(ctx context.Context) (done bool, err error) {
				logStream, err := clientset.CoreV1().Pods(podItem.Namespace).GetLogs(podItem.Name, &corev1.PodLogOptions{
					Container: sampleApp.ContainerName(),
				}).Stream(ctx)
				require.NoError(t, err)
				buffer := new(bytes.Buffer)
				_, err = io.Copy(buffer, logStream)

				return strings.Contains(buffer.String(), ldPreloadError), err
			}, wait.WithTimeout(2*time.Minute))

			require.NoError(t, err)
		}

		return ctx
	}
}
