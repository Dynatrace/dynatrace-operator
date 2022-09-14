//go:build e2e

package cloudnative

import (
	"context"
	"encoding/json"
	"os"
	"path"
	"strconv"
	"strings"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects/address"
	"github.com/Dynatrace/dynatrace-operator/test/bash"
	"github.com/Dynatrace/dynatrace-operator/test/csi"
	"github.com/Dynatrace/dynatrace-operator/test/kubeobjects/pod"
	"github.com/Dynatrace/dynatrace-operator/test/secrets"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
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

	codeModulesSecretsPath = "../testdata/secrets/cloudnative-codemodules.yaml"

	dataPath = "/data/"
)

type manifest struct {
	Version string `json:"version,omitempty"`
}

func codeModules(t *testing.T) features.Feature {
	currentWorkingDirectory, err := os.Getwd()
	require.NoError(t, err)

	secretPath := path.Join(currentWorkingDirectory, codeModulesSecretsPath)
	secretConfigs, err := secrets.ManyFromConfig(afero.NewOsFs(), secretPath)

	require.NoError(t, err)

	codeModulesInjection := features.New("codemodules injection")

	installAndDeploy(codeModulesInjection, secretConfigs[0], "../testdata/cloudnative/codemodules-deployment.yaml")

	assessDeployment(codeModulesInjection)

	codeModulesInjection.Assess("install dynakube", applyDynakube(secretConfigs[0].ApiUrl, codeModulesSpec()))

	assessDynakubeStartup(codeModulesInjection)
	assessOneAgentsAreRunning(codeModulesInjection)

	codeModulesInjection.Assess("csi driver did not crash", csiDriverIsAvailable)
	codeModulesInjection.Assess("codemodules have been downloaded", imageHasBeenDownloaded)
	codeModulesInjection.Assess("storage size has not increased", diskUsageDoesNotIncrease(secretConfigs[1]))

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
	client := environmentConfig.Client()
	resource := client.Resources()
	restConfig := client.RESTConfig()
	clientset, err := kubernetes.NewForConfig(resource.GetConfig())

	require.NoError(t, err)

	err = csi.ForEachPod(ctx, resource, func(podItem corev1.Pod) {
		var result *pod.ExecutionResult
		result, err = pod.
			NewExecutionQuery(podItem, "provisioner", bash.ListDirectory(dataPath)).
			Execute(clientset, restConfig)

		require.NoError(t, err)
		assert.Contains(t, result.StdOut.String(), "codemodules")

		result, err = pod.
			NewExecutionQuery(podItem, "provisioner", bash.ReadFile(getManifestPath())).
			Execute(clientset, restConfig)

		require.NoError(t, err)

		var codeModulesManifest manifest
		err = json.Unmarshal(result.StdOut.Bytes(), &codeModulesManifest)

		assert.NoError(t, err)
		assert.Equal(t, codeModulesVersion, codeModulesManifest.Version)
	})

	require.NoError(t, err)

	return ctx
}

func diskUsageDoesNotIncrease(secretConfig secrets.Secret) features.Func {
	return func(ctx context.Context, t *testing.T, environmentConfig *envconf.Config) context.Context {
		client := environmentConfig.Client()
		resource := client.Resources()
		restConfig := client.RESTConfig()
		clientset, err := kubernetes.NewForConfig(resource.GetConfig())

		require.NoError(t, err)

		storageMap := make(map[string]int)

		err = csi.ForEachPod(ctx, resource, func(podItem corev1.Pod) {
			var result *pod.ExecutionResult
			result, err = pod.
				NewExecutionQuery(podItem, "provisioner", bash.Pipe(
					bash.DiskUsageWithTotal(dataPath),
					bash.FilterLastLineOnly())).
				Execute(clientset, restConfig)

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
			dynakube, isDynakube := object.(*v1beta1.DynaKube)
			return isDynakube && dynakube.Status.Phase == v1beta1.Running
		})))

		err = csi.ForEachPod(ctx, resource, func(podItem corev1.Pod) {
			var result *pod.ExecutionResult
			result, err = pod.
				NewExecutionQuery(podItem, "provisioner", bash.Pipe(
					bash.DiskUsageWithTotal(dataPath),
					bash.FilterLastLineOnly())).
				Execute(clientset, restConfig)

			require.NoError(t, err)

			diskUsage, err := strconv.Atoi(strings.Split(result.StdOut.String(), "\t")[0])

			require.NoError(t, err)
			assert.Equal(t, storageMap[podItem.Name], diskUsage)
		})

		return ctx
	}
}

func getSecondTenantSecret(apiToken string) corev1.Secret {
	return corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "dynakube-2",
			Namespace: dynatraceNamespace,
		},
		Data: map[string][]byte{
			"apiToken": []byte(apiToken),
		},
	}
}

func getSecondTenantDynakube(apiUrl string) v1beta1.DynaKube {
	dynakube := v1beta1.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "dynakube-2",
			Namespace: dynatraceNamespace,
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
	dynakube.Spec.NamespaceSelector = metav1.LabelSelector{
		MatchLabels: map[string]string{
			"test-key": "test-value",
		},
	}

	return dynakube
}

func getManifestPath() string {
	return "/data/codemodules/" + codeModulesImageDigest + "/manifest.json"
}
