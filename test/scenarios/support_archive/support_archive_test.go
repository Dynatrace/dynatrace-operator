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

	"github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/cmd/support_archive"
	"github.com/Dynatrace/dynatrace-operator/src/functional"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	"github.com/Dynatrace/dynatrace-operator/test/csi"
	"github.com/Dynatrace/dynatrace-operator/test/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/kubeobjects/environment"
	"github.com/Dynatrace/dynatrace-operator/test/kubeobjects/namespace"
	"github.com/Dynatrace/dynatrace-operator/test/kubeobjects/pod"
	"github.com/Dynatrace/dynatrace-operator/test/kubeobjects/replicaset"
	"github.com/Dynatrace/dynatrace-operator/test/kubeobjects/service"
	"github.com/Dynatrace/dynatrace-operator/test/oneagent"
	"github.com/Dynatrace/dynatrace-operator/test/operator"
	"github.com/Dynatrace/dynatrace-operator/test/setup"
	e2ewebhook "github.com/Dynatrace/dynatrace-operator/test/webhook"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/exp/slices"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/e2e-framework/klient/conf"
	"sigs.k8s.io/e2e-framework/pkg/env"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

var testEnvironment env.Environment

const testAppNameNotInjected = "application1"
const testAppNameInjected = "application2"

func TestMain(m *testing.M) {
	testEnvironment = environment.Get()
	testEnvironment.BeforeEachTest(dynakube.DeleteIfExists(dynakube.NewBuilder().WithDefaultObjectMeta().Build()))
	testEnvironment.BeforeEachTest(oneagent.WaitForDaemonSetPodsDeletion())
	testEnvironment.BeforeEachTest(namespace.Recreate(namespace.NewBuilder(dynakube.Namespace).Build()))
	testEnvironment.BeforeEachTest(namespace.Recreate(namespace.NewBuilder(testAppNameNotInjected).Build()))
	testEnvironment.BeforeEachTest(namespace.Recreate(namespace.NewBuilder(testAppNameInjected).Build()))

	testEnvironment.AfterEachTest(namespace.Delete(testAppNameInjected))
	testEnvironment.AfterEachTest(namespace.Delete(testAppNameNotInjected))
	testEnvironment.AfterEachTest(dynakube.DeleteIfExists(dynakube.NewBuilder().WithDefaultObjectMeta().Build()))
	testEnvironment.AfterEachTest(oneagent.WaitForDaemonSetPodsDeletion())
	testEnvironment.AfterEachTest(namespace.Delete(dynakube.Namespace))

	testEnvironment.Run(m)
}

func TestSupportArchive(t *testing.T) {
	testEnvironment.Test(t, SupportArchiveExecution(t))
}

func SupportArchiveExecution(t *testing.T) features.Feature {
	secretConfig := getSecretConfig(t)

	supportArchiveExecution := features.New("support archive execution")
	setup.InstallDynatraceFromSource(supportArchiveExecution, &secretConfig)
	setup.AssessOperatorDeployment(supportArchiveExecution)

	dynakubeBuilder := dynakube.NewBuilder().
		WithDefaultObjectMeta().
		NamespaceSelector(v1.LabelSelector{
			MatchLabels: map[string]string{
				"kubernetes.io/metadata.name": testAppNameInjected,
			},
		}).
		ApiUrl(secretConfig.ApiUrl).
		CloudNative(&v1beta1.CloudNativeFullStackSpec{})
	supportArchiveExecution.Assess("dynakube applied", dynakube.Apply(dynakubeBuilder.Build()))
	setup.AssessDynakubeStartup(supportArchiveExecution)

	supportArchiveExecution.Assess("support archive subcommand can be executed correctly", testSupportArchiveCommand())
	return supportArchiveExecution.Feature()
}

func testSupportArchiveCommand() features.Func {
	return func(ctx context.Context, t *testing.T, environmentConfig *envconf.Config) context.Context {
		result := executeSupportArchiveCommand(ctx, t, environmentConfig, "--stdout")
		require.NotNil(t, result)

		zipReader, err := gzip.NewReader(result.StdOut)
		require.NoError(t, err)
		tarReader := tar.NewReader(zipReader)

		requiredFiles := collectRequiredFiles(t, ctx, environmentConfig)

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

func executeSupportArchiveCommand(ctx context.Context, t *testing.T, environmentConfig *envconf.Config, cmdLineArguments string) *pod.ExecutionResult {
	resources := environmentConfig.Client().Resources()

	pods := pod.List(t, ctx, resources, operator.Namespace)
	require.NotNil(t, pods.Items)

	operatorPods := functional.Filter(pods.Items, func(podItem corev1.Pod) bool {
		return strings.Contains(podItem.Name, "dynatrace-operator")
	})

	require.Len(t, operatorPods, 1)

	executionQuery := pod.NewExecutionQuery(operatorPods[0],
		"dynatrace-operator",
		"/usr/local/bin/dynatrace-operator",
		"support-archive",
		cmdLineArguments)
	executionResult, err := executionQuery.Execute(environmentConfig.Client().RESTConfig())
	require.NoError(t, err)

	return executionResult
}

func collectRequiredFiles(t *testing.T, ctx context.Context, environmentConfig *envconf.Config) []string {
	requiredFiles := make([]string, 0)
	requiredFiles = append(requiredFiles, support_archive.OperatorVersionFileName)
	requiredFiles = append(requiredFiles, getRequiredPodFiles(t, ctx, environmentConfig)...)
	requiredFiles = append(requiredFiles, getRequiredReplicasetFiles(t, ctx, environmentConfig)...)
	requiredFiles = append(requiredFiles, getRequiredServiceFiles(t, ctx, environmentConfig)...)
	requiredFiles = append(requiredFiles, getRequiredWorkloadFiles()...)
	requiredFiles = append(requiredFiles, getRequiredNamespaceFiles()...)
	requiredFiles = append(requiredFiles, getRequiredDynaKubeFiles()...)
	return requiredFiles
}

func getRequiredPodFiles(t *testing.T, ctx context.Context, environmentConfig *envconf.Config) []string {
	pods := pod.List(t, ctx, environmentConfig.Client().Resources(), operator.Namespace)
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

func getRequiredReplicasetFiles(t *testing.T, ctx context.Context, environmentConfig *envconf.Config) []string {
	replicaSets := replicaset.List(t, ctx, environmentConfig.Client().Resources(), operator.Namespace)
	requiredFiles := make([]string, 0)
	for _, replicaSet := range replicaSets.Items {
		requiredFiles = append(requiredFiles,
			fmt.Sprintf("%s/%s/ReplicaSet/%s%s", support_archive.ManifestsDirectoryName, replicaSet.Namespace, replicaSet.Name, support_archive.ManifestsFileExtension))
	}
	return requiredFiles
}

func getRequiredServiceFiles(t *testing.T, ctx context.Context, environmentConfig *envconf.Config) []string {
	services := service.List(t, ctx, environmentConfig.Client().Resources(), operator.Namespace)
	requiredFiles := make([]string, 0)
	for _, service := range services.Items {
		requiredFiles = append(requiredFiles,
			fmt.Sprintf("%s/%s/Service/%s%s", support_archive.ManifestsDirectoryName, service.Namespace, service.Name, support_archive.ManifestsFileExtension))
	}
	return requiredFiles
}

func getRequiredWorkloadFiles() []string {
	requiredFiles := make([]string, 0)
	requiredFiles = append(requiredFiles,
		fmt.Sprintf("%s/%s/%s/%s%s",
			support_archive.ManifestsDirectoryName,
			operator.Namespace,
			"Deployment",
			operator.Name,
			support_archive.ManifestsFileExtension))
	requiredFiles = append(requiredFiles,
		fmt.Sprintf("%s/%s/%s/%s%s",
			support_archive.ManifestsDirectoryName,
			operator.Namespace,
			"Deployment",
			e2ewebhook.Name,
			support_archive.ManifestsFileExtension))
	requiredFiles = append(requiredFiles,
		fmt.Sprintf("%s/%s/%s/%s%s",
			support_archive.ManifestsDirectoryName,
			operator.Namespace,
			"DaemonSet",
			csi.Name,
			support_archive.ManifestsFileExtension))
	return requiredFiles
}

func getRequiredNamespaceFiles() []string {
	requiredFiles := make([]string, 0)
	requiredFiles = append(requiredFiles,
		fmt.Sprintf("%s/%s/Namespace-%s%s",
			support_archive.ManifestsDirectoryName,
			operator.Namespace,
			operator.Namespace,
			support_archive.ManifestsFileExtension))
	requiredFiles = append(requiredFiles,
		fmt.Sprintf("%s/%s/Namespace-%s%s",
			support_archive.ManifestsDirectoryName,
			support_archive.InjectedNamespacesManifestsDirectoryName,
			testAppNameInjected,
			support_archive.ManifestsFileExtension))
	return requiredFiles
}

func getRequiredDynaKubeFiles() []string {
	requiredFiles := make([]string, 0)
	requiredFiles = append(requiredFiles,
		fmt.Sprintf("%s/%s/%s/%s%s",
			support_archive.ManifestsDirectoryName,
			dynakube.Namespace,
			"DynaKube",
			dynakube.Name,
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

// Note: mainly for dev purposes, test requires a running cluster with deployed operator to be successful
func TestExecSupportArchive(t *testing.T) {
	t.Skip("dev helper test")
	kubeConfigPath := conf.ResolveKubeConfigFile()
	environmentConfig := envconf.NewWithKubeConfig(kubeConfigPath)

	testSupportArchiveCommand()(context.TODO(), t, environmentConfig)
}
