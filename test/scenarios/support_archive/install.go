//go:build e2e

package support_archive

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1/dynakube"
	"github.com/Dynatrace/dynatrace-operator/src/functional"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/operator"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/namespace"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/pod"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/steps/assess"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/tenant"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/exp/slices"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

const testAppNameNotInjected = "application1"
const testAppNameInjected = "application2"

func supportArchiveExecution(t *testing.T) features.Feature {
	builder := features.New("support archive execution")
	secretConfig := tenant.GetSingleTenantSecret(t)

	injectLabels := map[string]string{
		"inject": "me",
	}

	dynakubeBuilder := dynakube.NewBuilder().
		WithDefaultObjectMeta().
		NamespaceSelector(metav1.LabelSelector{
			MatchLabels: injectLabels,
		}).
		ApiUrl(secretConfig.ApiUrl).
		CloudNative(&dynatracev1beta1.CloudNativeFullStackSpec{}).
		WithActiveGate()
	testDynakube := dynakubeBuilder.Build()

	// Register sample namespace creat and delete
	builder.Assess("create sample injected namespace", namespace.Create(namespace.NewBuilder(testAppNameInjected).WithLabels(injectLabels).Build()))
	builder.Assess("create sample not injected namespace", namespace.Create(namespace.NewBuilder(testAppNameNotInjected).Build()))
	builder.Teardown(namespace.Delete(testAppNameInjected))
	builder.Teardown(namespace.Delete(testAppNameNotInjected))

	// Register operator + dynakube install and teardown
	assess.InstallDynatraceWithTeardown(builder, &secretConfig, testDynakube)

	// Register actual test
	builder.Assess("support archive subcommand can be executed correctly with managed logs", testSupportArchiveCommand(testDynakube, true))
	builder.Assess("support archive subcommand can be executed correctly without managed logs", testSupportArchiveCommand(testDynakube, false))

	return builder.Feature()
}

func testSupportArchiveCommand(testDynakube dynatracev1beta1.DynaKube, collectManaged bool) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		commandLineArguments := []string{"--stdout"}
		if !collectManaged {
			commandLineArguments = append(commandLineArguments, "--managed-logs=false")
		}

		result := executeSupportArchiveCommand(ctx, t, envConfig, commandLineArguments, testDynakube.Namespace)
		require.NotNil(t, result)

		zipReader, err := zip.NewReader(bytes.NewReader(result.StdOut.Bytes()), int64(result.StdOut.Len()))

		require.NoError(t, err)

		requiredFiles := newRequiredFiles(t, ctx, envConfig.Client().Resources(), testDynakube, collectManaged).
			collectRequiredFiles()
		for _, file := range zipReader.File {
			requiredFiles = assertFile(t, requiredFiles, *file)
		}

		assert.Emptyf(t, requiredFiles, "Support archive does not contain all expected files.")
		logMissingFiles(t, requiredFiles)
		return ctx
	}
}

func executeSupportArchiveCommand(ctx context.Context, t *testing.T, envConfig *envconf.Config, cmdLineArguments []string, namespace string) *pod.ExecutionResult { //nolint:revive
	environmentResources := envConfig.Client().Resources()

	pods := pod.List(t, ctx, environmentResources, namespace)
	require.NotNil(t, pods.Items)

	operatorPods := functional.Filter(pods.Items, func(podItem corev1.Pod) bool {
		return strings.Contains(podItem.Name, "dynatrace-operator")
	})

	require.Len(t, operatorPods, 1)
	command := []string{"/usr/local/bin/dynatrace-operator", "support-archive"}
	command = append(command, cmdLineArguments...)

	executionResult, err := pod.Exec(ctx, envConfig.Client().Resources(),
		operatorPods[0],
		operator.DeploymentName,
		command...,
	)
	require.NoError(t, err)

	return executionResult
}

func assertFile(t *testing.T, requiredFiles []string, zipFile zip.File) []string {
	zipFileName := zipFile.Name
	index := slices.IndexFunc(requiredFiles, func(file string) bool { return file == zipFileName })

	if index != -1 {
		requiredFiles = slices.Delete(requiredFiles, index, index+1)
	} else {
		t.Error("unexpected file found", "filename:", zipFileName)
	}

	assert.NotZerof(t, zipFile.FileInfo().Size(), "File %s is empty.", zipFileName)

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
