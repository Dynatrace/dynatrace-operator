//go:build e2e

package cloudnative

import (
	"context"
	"encoding/json"
	"path"
	"strconv"
	"strings"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	dtcsi "github.com/Dynatrace/dynatrace-operator/src/controllers/csi"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects/address"
	"github.com/Dynatrace/dynatrace-operator/src/webhook"
	"github.com/Dynatrace/dynatrace-operator/src/webhook/mutation/pod_mutator/oneagent_mutation"
	"github.com/Dynatrace/dynatrace-operator/test/csi"
	"github.com/Dynatrace/dynatrace-operator/test/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/istiosetup"
	"github.com/Dynatrace/dynatrace-operator/test/kubeobjects/deployment"
	"github.com/Dynatrace/dynatrace-operator/test/kubeobjects/manifests"
	"github.com/Dynatrace/dynatrace-operator/test/kubeobjects/pod"
	"github.com/Dynatrace/dynatrace-operator/test/project"
	"github.com/Dynatrace/dynatrace-operator/test/sampleapps"
	"github.com/Dynatrace/dynatrace-operator/test/secrets"
	"github.com/Dynatrace/dynatrace-operator/test/setup"
	"github.com/Dynatrace/dynatrace-operator/test/shell"
	"github.com/pkg/errors"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/e2e-framework/klient/k8s"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/klient/wait/conditions"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

const (
	codeModulesVersion     = "1.246.0.20220627-183412"
	codeModulesImage       = "quay.io/dynatrace/codemodules:" + codeModulesVersion
	codeModulesImageDigest = "7ece13a07a20c77a31cc36906a10ebc90bd47970905ee61e8ed491b7f4c5d62f"

	dataPath = "/data/"
)

var (
	codeModulesDeploymentConfig = path.Join(project.TestDataDir(), "cloudnative/codemodules-deployment.yaml")
)

type manifest struct {
	Version string `json:"version,omitempty"`
}

func CodeModules(t *testing.T, istioEnabled bool) features.Feature {
	secretConfigs, err := secrets.DefaultMultiTenant(afero.NewOsFs())

	require.NoError(t, err)

	codeModulesInjection := features.New("codemodules injection")

	if istioEnabled {
		codeModulesInjection.Setup(manifests.InstallFromFile(istioTestNamespaceConfig))
	} else {
		codeModulesInjection.Setup(manifests.InstallFromFile(testNamespaceConfig))
	}
	setup.InstallDynatraceFromSource(codeModulesInjection, &secretConfigs[0])
	setup.AssessOperatorDeployment(codeModulesInjection)

	setup.DeploySampleApps(codeModulesInjection, codeModulesDeploymentConfig)

	dynakubeBuilder := dynakube.NewBuilder().
		WithDefaultObjectMeta().
		ApiUrl(secretConfigs[0].ApiUrl).
		CloudNative(codeModulesSpec())
	if istioEnabled {
		dynakubeBuilder = dynakubeBuilder.WithIstio()
	}

	codeModulesInjection.Assess("install dynakube", dynakube.Apply(dynakubeBuilder.Build()))

	setup.AssessDynakubeStartup(codeModulesInjection)
	assessSampleAppsRestart(codeModulesInjection)
	assessOneAgentsAreRunning(codeModulesInjection)

	if istioEnabled {
		istiosetup.AssessIstio(codeModulesInjection)
	}

	codeModulesInjection.Assess("csi driver did not crash", csiDriverIsAvailable)
	codeModulesInjection.Assess("codemodules have been downloaded", imageHasBeenDownloaded)
	codeModulesInjection.Assess("storage size has not increased", diskUsageDoesNotIncrease(secretConfigs[0]))
	codeModulesInjection.Assess("volumes are mounted correctly", volumesAreMountedCorrectly())

	return codeModulesInjection.Feature()
}

func codeModulesSpec() *v1beta1.CloudNativeFullStackSpec {
	return &v1beta1.CloudNativeFullStackSpec{
		HostInjectSpec: v1beta1.HostInjectSpec{
			NodeSelector: map[string]string{
				"inject": "dynakube",
			},
		},
		AppInjectionSpec: v1beta1.AppInjectionSpec{
			CodeModulesImage: codeModulesImage,
		},
	}
}

func csiDriverIsAvailable(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
	resource := envConfig.Client().Resources()
	daemonset, err := csi.Get(ctx, resource)

	require.NoError(t, err)
	assert.Equal(t, daemonset.Status.DesiredNumberScheduled, daemonset.Status.NumberReady)

	return ctx
}

func imageHasBeenDownloaded(ctx context.Context, t *testing.T, environmentConfig *envconf.Config) context.Context {
	resource := environmentConfig.Client().Resources()
	restConfig := environmentConfig.Client().RESTConfig()

	err := csi.ForEachPod(ctx, resource, func(podItem corev1.Pod) {
		var result *pod.ExecutionResult
		result, err := pod.
			NewExecutionQuery(podItem, "provisioner", shell.ListDirectory(dataPath)...).
			Execute(restConfig)

		require.NoError(t, err)
		assert.Contains(t, result.StdOut.String(), "codemodules")

		result, err = pod.
			NewExecutionQuery(podItem, "provisioner", shell.Shell(shell.ReadFile(getManifestPath()))...).
			Execute(restConfig)

		require.NoError(t, err)

		var codeModulesManifest manifest
		err = json.Unmarshal(result.StdOut.Bytes(), &codeModulesManifest)
		if err != nil {
			err = errors.WithMessagef(err, "json:\n%s", result.StdOut)
		}
		require.NoError(t, err)

		assert.Equal(t, codeModulesVersion, codeModulesManifest.Version)
	})

	require.NoError(t, err)

	return ctx
}

func diskUsageDoesNotIncrease(secretConfig secrets.Secret) features.Func {
	return func(ctx context.Context, t *testing.T, environmentConfig *envconf.Config) context.Context {
		resource := environmentConfig.Client().Resources()
		restConfig := environmentConfig.Client().RESTConfig()
		storageMap := make(map[string]int)
		err := csi.ForEachPod(ctx, resource, func(podItem corev1.Pod) {
			var result *pod.ExecutionResult
			result, err := pod.
				NewExecutionQuery(podItem, "provisioner", shell.Shell(shell.Pipe(
					shell.DiskUsageWithTotal(dataPath),
					shell.FilterLastLineOnly()))...).
				Execute(restConfig)

			require.NoError(t, err)

			diskUsage, err := strconv.Atoi(strings.Split(result.StdOut.String(), "\t")[0])

			require.NoError(t, err)

			storageMap[podItem.Name] = diskUsage
		})

		secondTenantSecret := getSecondTenantSecret(secretConfig.ApiToken)
		secondTenant := getSecondTenantDynakube(secretConfig.ApiUrl)

		require.NoError(t, err)
		require.NoError(t, resource.Create(ctx, &secondTenantSecret))
		require.NoError(t, resource.Create(ctx, &secondTenant))

		require.NoError(t, wait.For(conditions.New(resource).ResourceMatch(&secondTenant, func(object k8s.Object) bool {
			dynakubeInstance, isDynakube := object.(*v1beta1.DynaKube)
			return isDynakube && dynakubeInstance.Status.Phase == v1beta1.Running
		})))

		err = csi.ForEachPod(ctx, resource, func(podItem corev1.Pod) {
			var result *pod.ExecutionResult
			result, err = pod.
				NewExecutionQuery(podItem, "provisioner", shell.Shell(shell.Pipe(
					shell.DiskUsageWithTotal(dataPath),
					shell.FilterLastLineOnly()))...).
				Execute(restConfig)

			require.NoError(t, err)

			diskUsage, err := strconv.Atoi(strings.Split(result.StdOut.String(), "\t")[0])

			require.NoError(t, err)
			// Dividing it by 1000 so the sizes do not need to be exactly the same down to the byte
			assert.Equal(t, storageMap[podItem.Name]/1000, diskUsage/1000)
		})

		return ctx
	}
}

func volumesAreMountedCorrectly() features.Func {
	return func(ctx context.Context, t *testing.T, environmentConfig *envconf.Config) context.Context {
		resource := environmentConfig.Client().Resources()
		err := deployment.NewQuery(ctx, resource, client.ObjectKey{
			Name:      sampleapps.Name,
			Namespace: sampleapps.Namespace,
		}).ForEachPod(func(podItem corev1.Pod) {
			volumes := podItem.Spec.Volumes
			volumeMounts := podItem.Spec.Containers[0].VolumeMounts

			assert.True(t, isVolumeAttached(t, volumes, oneagent_mutation.OneAgentBinVolumeName))
			assert.True(t, isVolumeMounted(t, volumeMounts, oneagent_mutation.OneAgentBinVolumeName))

			executionResult, err := pod.
				NewExecutionQuery(podItem, sampleapps.Name, shell.ListDirectory(webhook.DefaultInstallPath)...).
				Execute(environmentConfig.Client().RESTConfig())

			require.NoError(t, err)
			assert.NotEmpty(t, executionResult.StdOut.String())

			executionResult, err = pod.
				NewExecutionQuery(podItem, sampleapps.Name, shell.Shell(shell.Pipe(
					shell.DiskUsageWithTotal(webhook.DefaultInstallPath),
					shell.FilterLastLineOnly()))...).
				Execute(environmentConfig.Client().RESTConfig())

			require.NoError(t, err)
			require.Contains(t, executionResult.StdOut.String(), "total")

			diskUsage, err := strconv.Atoi(strings.Split(executionResult.StdOut.String(), "\t")[0])

			require.NoError(t, err)
			assert.Greater(t, diskUsage, 0)
		})

		require.NoError(t, err)
		return ctx
	}
}

func isVolumeMounted(t *testing.T, volumeMounts []corev1.VolumeMount, volumeMountName string) bool {
	result := false
	for _, volumeMount := range volumeMounts {
		if volumeMount.Name == volumeMountName {
			result = true

			assert.Equal(t, webhook.DefaultInstallPath, volumeMount.MountPath)
			assert.False(t, volumeMount.ReadOnly)
		}
	}
	return result
}

func isVolumeAttached(t *testing.T, volumes []corev1.Volume, volumeName string) bool {
	result := false
	for _, volume := range volumes {
		if volume.Name == volumeName {
			result = true

			assert.NotNil(t, volume.CSI)
			assert.Equal(t, dtcsi.DriverName, volume.CSI.Driver)

			if volume.CSI.ReadOnly != nil {
				assert.False(t, *volume.CSI.ReadOnly)
			}
		}
	}
	return result
}

func getSecondTenantSecret(apiToken string) corev1.Secret {
	return corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "dynakube-2",
			Namespace: dynakube.Namespace,
		},
		Data: map[string][]byte{
			"apiToken": []byte(apiToken),
		},
	}
}

func getSecondTenantDynakube(apiUrl string) v1beta1.DynaKube {
	dynakubeInstance := v1beta1.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "dynakube-2",
			Namespace: dynakube.Namespace,
		},
		Spec: v1beta1.DynaKubeSpec{
			APIURL: apiUrl,
			OneAgent: v1beta1.OneAgentSpec{
				ApplicationMonitoring: &v1beta1.ApplicationMonitoringSpec{
					UseCSIDriver: address.Of(true),
					AppInjectionSpec: v1beta1.AppInjectionSpec{
						CodeModulesImage: codeModulesImage,
					},
				},
			},
		},
	}
	dynakubeInstance.Spec.NamespaceSelector = metav1.LabelSelector{
		MatchLabels: map[string]string{
			"test-key": "test-value",
		},
	}

	return dynakubeInstance
}

func getManifestPath() string {
	return "/data/codemodules/" + codeModulesImageDigest + "/manifest.json"
}
