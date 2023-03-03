//go:build e2e

package support_archive

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"strings"
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/cmd/support_archive"
	"github.com/Dynatrace/dynatrace-operator/src/functional"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/csi"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/operator"
	e2ewebhook "github.com/Dynatrace/dynatrace-operator/test/helpers/components/webhook"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/namespace"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/pod"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/replicaset"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/service"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/steps/assess"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/tenant"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/exp/slices"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

const testAppNameNotInjected = "application1"
const testAppNameInjected = "application2"

func supportArchiveExecution(t *testing.T) features.Feature {
	builder := features.New("support archive execution")
	secretConfig := tenant.GetSingleTenantSecret(t)

	dynakubeBuilder := dynakube.NewBuilder().
		WithDefaultObjectMeta().
		NamespaceSelector(metav1.LabelSelector{
			MatchLabels: map[string]string{
				"kubernetes.io/metadata.name": testAppNameInjected,
			},
		}).
		ApiUrl(secretConfig.ApiUrl).
		CloudNative(&dynatracev1beta1.CloudNativeFullStackSpec{})
	testDynakube := dynakubeBuilder.Build()

	// Register sample namespace creat and delete
	builder.Assess("create sample injected namespace", namespace.Create(namespace.NewBuilder(testAppNameInjected).Build()))
	builder.Assess("create sample not injected namespace", namespace.Create(namespace.NewBuilder(testAppNameNotInjected).Build()))
	builder.Teardown(namespace.Delete(testAppNameInjected))
	builder.Teardown(namespace.Delete(testAppNameNotInjected))

	// Register operator + dynakube install and teardown
	assess.InstallDynatraceWithTeardown(builder, &secretConfig, testDynakube)

	// Register actual test
	builder.Assess("support archive subcommand can be executed correctly", testSupportArchiveCommand(testDynakube))

	return builder.Feature()
}

func testSupportArchiveCommand(testDynakube dynatracev1beta1.DynaKube) features.Func {
	return func(ctx context.Context, t *testing.T, environmentConfig *envconf.Config) context.Context {
		result := executeSupportArchiveCommand(ctx, t, environmentConfig, "--stdout", testDynakube.Namespace)
		require.NotNil(t, result)

		zipReader, err := gzip.NewReader(result.StdOut)
		require.NoError(t, err)
		tarReader := tar.NewReader(zipReader)

		requiredFiles := collectRequiredFiles(t, ctx, environmentConfig.Client().Resources(), testDynakube)

		hdr, err := tarReader.Next()
		for err == nil {
			requiredFiles = assertFile(t, requiredFiles, *hdr)
			hdr, err = tarReader.Next()
		}

		require.Equal(t, io.EOF, err)

		assert.Emptyf(t, requiredFiles, "Support archive does not contain all expected files.")
		logMissingFiles(t, requiredFiles)
		return ctx
	}
}

func executeSupportArchiveCommand(ctx context.Context, t *testing.T, environmentConfig *envconf.Config, cmdLineArguments, namespace string) *pod.ExecutionResult { //nolint:revive
	resources := environmentConfig.Client().Resources()

	pods := pod.List(t, ctx, resources, namespace)
	require.NotNil(t, pods.Items)

	operatorPods := functional.Filter(pods.Items, func(podItem corev1.Pod) bool {
		return strings.Contains(podItem.Name, "dynatrace-operator")
	})

	require.Len(t, operatorPods, 1)

	executionQuery := pod.NewExecutionQuery(ctx, environmentConfig.Client().Resources(), operatorPods[0],
		operator.ContainerName,
		"/usr/local/bin/dynatrace-operator",
		"support-archive",
		cmdLineArguments)
	executionResult, err := executionQuery.Execute()
	require.NoError(t, err)

	return executionResult
}

func collectRequiredFiles(t *testing.T, ctx context.Context, resources *resources.Resources, testDynakube dynatracev1beta1.DynaKube) []string {
	namespace := testDynakube.Namespace
	requiredFiles := make([]string, 0)
	requiredFiles = append(requiredFiles, support_archive.OperatorVersionFileName)
	requiredFiles = append(requiredFiles, getRequiredPodFiles(t, ctx, resources, namespace)...)
	requiredFiles = append(requiredFiles, getRequiredReplicaSetFiles(t, ctx, resources, namespace)...)
	requiredFiles = append(requiredFiles, getRequiredServiceFiles(t, ctx, resources, namespace)...)
	requiredFiles = append(requiredFiles, getRequiredWorkloadFiles(namespace)...)
	requiredFiles = append(requiredFiles, getRequiredNamespaceFiles(namespace)...)
	requiredFiles = append(requiredFiles, getRequiredDynaKubeFiles(testDynakube)...)
	return requiredFiles
}

func getRequiredPodFiles(t *testing.T, ctx context.Context, resources *resources.Resources, namespace string) []string {
	pods := pod.List(t, ctx, resources, namespace)
	requiredFiles := make([]string, 0)

	operatorPods := functional.Filter(pods.Items, func(podItem corev1.Pod) bool {
		appNameLabel, ok := podItem.Labels[kubeobjects.AppNameLabel]
		return ok && appNameLabel == "dynatrace-operator"
	})

	for _, pod := range operatorPods {
		requiredFiles = append(requiredFiles,
			fmt.Sprintf("%s/%s/Pod/%s%s", support_archive.ManifestsDirectoryName, pod.Namespace, pod.Name, support_archive.ManifestsFileExtension))
		for _, container := range pod.Spec.Containers {
			requiredFiles = append(requiredFiles,
				fmt.Sprintf("%s/%s/%s.log", support_archive.LogsDirectoryName, pod.Name, container.Name))
		}
	}
	return requiredFiles
}

func getRequiredReplicaSetFiles(t *testing.T, ctx context.Context, resources *resources.Resources, namespace string) []string {
	replicaSets := replicaset.List(t, ctx, resources, namespace)
	requiredFiles := make([]string, 0)
	for _, replicaSet := range replicaSets.Items {
		requiredFiles = append(requiredFiles,
			fmt.Sprintf("%s/%s/ReplicaSet/%s%s", support_archive.ManifestsDirectoryName, replicaSet.Namespace, replicaSet.Name, support_archive.ManifestsFileExtension))
	}
	return requiredFiles
}

func getRequiredServiceFiles(t *testing.T, ctx context.Context, resources *resources.Resources, namespace string) []string {
	services := service.List(t, ctx, resources, namespace)
	requiredFiles := make([]string, 0)
	for _, service := range services.Items {
		requiredFiles = append(requiredFiles,
			fmt.Sprintf("%s/%s/Service/%s%s", support_archive.ManifestsDirectoryName, service.Namespace, service.Name, support_archive.ManifestsFileExtension))
	}
	return requiredFiles
}

func getRequiredWorkloadFiles(namespace string) []string {
	requiredFiles := make([]string, 0)
	requiredFiles = append(requiredFiles,
		fmt.Sprintf("%s/%s/%s/%s%s",
			support_archive.ManifestsDirectoryName,
			namespace,
			"Deployment",
			operator.DeploymentName,
			support_archive.ManifestsFileExtension))
	requiredFiles = append(requiredFiles,
		fmt.Sprintf("%s/%s/%s/%s%s",
			support_archive.ManifestsDirectoryName,
			namespace,
			"Deployment",
			e2ewebhook.DeploymentName,
			support_archive.ManifestsFileExtension))
	requiredFiles = append(requiredFiles,
		fmt.Sprintf("%s/%s/%s/%s%s",
			support_archive.ManifestsDirectoryName,
			namespace,
			"DaemonSet",
			csi.DaemonSetName,
			support_archive.ManifestsFileExtension))
	return requiredFiles
}

func getRequiredNamespaceFiles(namespace string) []string {
	requiredFiles := make([]string, 0)
	requiredFiles = append(requiredFiles,
		fmt.Sprintf("%s/%s/Namespace-%s%s",
			support_archive.ManifestsDirectoryName,
			namespace,
			namespace,
			support_archive.ManifestsFileExtension))
	requiredFiles = append(requiredFiles,
		fmt.Sprintf("%s/%s/Namespace-%s%s",
			support_archive.ManifestsDirectoryName,
			support_archive.InjectedNamespacesManifestsDirectoryName,
			testAppNameInjected,
			support_archive.ManifestsFileExtension))
	return requiredFiles
}

func getRequiredDynaKubeFiles(testDynakube dynatracev1beta1.DynaKube) []string {
	requiredFiles := make([]string, 0)
	requiredFiles = append(requiredFiles,
		fmt.Sprintf("%s/%s/%s/%s%s",
			support_archive.ManifestsDirectoryName,
			testDynakube.Namespace,
			"DynaKube",
			testDynakube.Name,
			support_archive.ManifestsFileExtension))

	return requiredFiles
}

func assertFile(t *testing.T, requiredFiles []string, hdr tar.Header) []string {
	index := slices.IndexFunc(requiredFiles, func(file string) bool { return file == hdr.Name })
	assert.NotEqualf(t, -1, index, "Found unexpected file %s.", hdr.Name)

	if index != -1 {
		requiredFiles = slices.Delete(requiredFiles, index, index+1)
	}

	assert.NotZerof(t, hdr.Size, "File %s is empty.", hdr.Name)

	return requiredFiles
}

func logMissingFiles(t *testing.T, requiredFiles []string) {
	if len(requiredFiles) > 0 {
		missingFilesLog := "Missing files:"
		for _, file := range requiredFiles {
			missingFilesLog = fmt.Sprintf("%s\n%s", missingFilesLog, file)
		}
		t.Log(missingFilesLog)
	}
}
