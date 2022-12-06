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

	"github.com/Dynatrace/dynatrace-operator/src/cmd/support_archive"
	"github.com/Dynatrace/dynatrace-operator/src/functional"
	"github.com/Dynatrace/dynatrace-operator/test/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/kubeobjects/environment"
	"github.com/Dynatrace/dynatrace-operator/test/kubeobjects/namespace"
	"github.com/Dynatrace/dynatrace-operator/test/kubeobjects/pod"
	"github.com/Dynatrace/dynatrace-operator/test/oneagent"
	"github.com/Dynatrace/dynatrace-operator/test/operator"
	"github.com/Dynatrace/dynatrace-operator/test/setup"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/exp/slices"
	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/e2e-framework/klient/conf"
	"sigs.k8s.io/e2e-framework/pkg/env"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

var testEnvironment env.Environment

func TestMain(m *testing.M) {
	testEnvironment = environment.Get()
	testEnvironment.BeforeEachTest(dynakube.DeleteIfExists(dynakube.NewBuilder().WithDefaultObjectMeta().Build()))
	testEnvironment.BeforeEachTest(oneagent.WaitForDaemonSetPodsDeletion())
	testEnvironment.BeforeEachTest(namespace.Recreate(namespace.NewBuilder(dynakube.Namespace).Build()))

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
	supportArchiveExecution.Assess("support archive subcommand can be executed correctly", checkSupportArchiveExecution())
	return supportArchiveExecution.Feature()
}

func checkSupportArchiveExecution() features.Func {
	return func(ctx context.Context, t *testing.T, environmentConfig *envconf.Config) context.Context {
		result := executeSupportArchive(ctx, t, environmentConfig, "--stdout")
		require.NotNil(t, result)

		zipReader, err := gzip.NewReader(result.StdOut)
		require.NoError(t, err)

		requiredFiles := make([]string, 0)
		requiredFiles = append(requiredFiles, support_archive.OperatorVersionFileName)
		requiredFiles = append(requiredFiles, getRequiredLogFiles(t, ctx, environmentConfig)...)

		tarReader := tar.NewReader(zipReader)

		hdr, err := tarReader.Next()
		for err == nil {
			index := slices.IndexFunc(requiredFiles, func(file string) bool { return file == hdr.Name })
			assert.NotEqualf(t, -1, index, "Found unexpected file %s.", hdr.Name)

			if index != -1 {
				requiredFiles = slices.Delete(requiredFiles, index, index+1)
			}

			assert.NotZerof(t, hdr.Size, "File %s is empty.", hdr.Name)
			hdr, err = tarReader.Next()
		}
		require.Equal(t, io.EOF, err)
		assert.Emptyf(t, requiredFiles, "Support archive does not contain all expected files.")
		return ctx
	}
}

func executeSupportArchive(ctx context.Context, t *testing.T, environmentConfig *envconf.Config, cmdLineArguments string) *pod.ExecutionResult {
	resources := environmentConfig.Client().Resources()

	pods := operator.Get(t, ctx, resources)
	require.NotNil(t, pods.Items)

	operatorPods := functional.Filter(pods.Items, func(podItem v1.Pod) bool {
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

func getRequiredLogFiles(t *testing.T, ctx context.Context, envConfig *envconf.Config) []string {
	resources := envConfig.Client().Resources()
	opPods := operator.Get(t, ctx, resources)

	requiredLogs := make([]string, 0)

	for _, pod := range opPods.Items {
		for _, container := range pod.Spec.Containers {
			requiredLogs = append(requiredLogs, fmt.Sprintf("%s/%s/%s.log",
				support_archive.LogsDirectoryName, pod.Name, container.Name))
		}
	}
	return requiredLogs
}

// Note: mainly for dev purposes, test requires a running cluster with deployed operator to be successful
func TestExecSupportArchive(t *testing.T) {
	t.Skip("dev helper test")
	kubeConfigPath := conf.ResolveKubeConfigFile()
	envConfig := envconf.NewWithKubeConfig(kubeConfigPath)

	checkSupportArchiveExecution()(context.TODO(), t, envConfig)
}
