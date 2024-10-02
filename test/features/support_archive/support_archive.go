//go:build e2e

package support_archive

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha2/edgeconnect"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/functional"
	"github.com/Dynatrace/dynatrace-operator/test/helpers"
	dynakubeComponents "github.com/Dynatrace/dynatrace-operator/test/helpers/components/dynakube"
	edgeconnectComponents "github.com/Dynatrace/dynatrace-operator/test/helpers/components/edgeconnect"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/operator"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/namespace"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/pod"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/rand"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/tenant"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/exp/slices"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

const (
	testAppNameNotInjected = "application1"
	testAppNameInjected    = "application2"
	defaultRandomLength    = 5
)

type CustomResources struct {
	dk dynakube.DynaKube
	ec edgeconnect.EdgeConnect
}

// Setup: DTO with CSI driver
//
// Verification if support-archive package created by the support-archive command and printed
// to the standard output is a valid tar.gz package and contains required *operator-version.txt*
// file.
func Feature(t *testing.T) features.Feature {
	builder := features.New("support-archive")
	secretConfig := tenant.GetSingleTenantSecret(t)
	edgeconnectSecretConfig := tenant.GetEdgeConnectTenantSecret(t)

	injectLabels := map[string]string{
		"inject": "me",
	}

	testDynakube := *dynakubeComponents.New(
		dynakubeComponents.WithOneAgentNamespaceSelector(metav1.LabelSelector{
			MatchLabels: injectLabels,
		}),
		dynakubeComponents.WithApiUrl(secretConfig.ApiUrl),
		dynakubeComponents.WithCloudNativeSpec(&dynakube.CloudNativeFullStackSpec{}),
		dynakubeComponents.WithActiveGate(),
	)

	testECname, err := rand.GetRandomName(rand.WithLength(defaultRandomLength), rand.WithPrefix("test-edgeconnect-support-"))
	require.NoError(t, err)

	testEdgeConnect := *edgeconnectComponents.New(
		edgeconnectComponents.WithName(testECname),
		edgeconnectComponents.WithApiServer(edgeconnectSecretConfig.ApiServer),
		edgeconnectComponents.WithOAuthClientSecret(fmt.Sprintf("%s-client-secret", testECname)),
		edgeconnectComponents.WithOAuthEndpoint("https://sso-dev.dynatracelabs.com/sso/oauth2/token"),
		edgeconnectComponents.WithOAuthResource(fmt.Sprintf("urn:dtenvironment:%s", edgeconnectSecretConfig.TenantUid)),
	)

	builder.Assess("deploy injected namespace", namespace.Create(*namespace.New(testAppNameInjected, namespace.WithLabels(injectLabels))))
	builder.Assess("deploy NOT injected namespace", namespace.Create(*namespace.New(testAppNameNotInjected)))
	dynakubeComponents.Install(builder, helpers.LevelAssess, &secretConfig, testDynakube)
	edgeconnectComponents.Install(builder, helpers.LevelAssess, &edgeconnectSecretConfig, testEdgeConnect)

	// Register actual test
	builder.Assess("support archive subcommand can be executed correctly with managed logs", testSupportArchiveCommand(testDynakube, testEdgeConnect, true))
	builder.Assess("support archive subcommand can be executed correctly without managed logs", testSupportArchiveCommand(testDynakube, testEdgeConnect, false))

	builder.WithTeardown("remove injected namespace", namespace.Delete(testAppNameInjected))
	builder.WithTeardown("remove NOT injected namespace", namespace.Delete(testAppNameNotInjected))
	dynakubeComponents.Delete(builder, helpers.LevelTeardown, testDynakube)
	builder.WithTeardown("remove edgeconnect CR", edgeconnectComponents.Delete(testEdgeConnect))

	return builder.Feature()
}

func testSupportArchiveCommand(testDynakube dynakube.DynaKube, testEdgeConnect edgeconnect.EdgeConnect, collectManaged bool) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		commandLineArguments := []string{"--stdout"}
		if !collectManaged {
			commandLineArguments = append(commandLineArguments, "--managed-logs=false")
		}

		result := executeSupportArchiveCommand(ctx, t, envConfig, commandLineArguments, testDynakube.Namespace)
		require.NotNil(t, result)

		zipReader, err := zip.NewReader(bytes.NewReader(result.StdOut.Bytes()), int64(result.StdOut.Len()))

		require.NoError(t, err)

		customResources := CustomResources{
			dk: testDynakube,
			ec: testEdgeConnect,
		}
		requiredFiles := newRequiredFiles(t, ctx, envConfig.Client().Resources(), customResources, collectManaged).
			collectRequiredFiles()
		for _, file := range zipReader.File {
			requiredFiles = assertFile(t, requiredFiles, *file)
		}

		assert.Emptyf(t, requiredFiles, "Support archive does not contain all expected files.")
		logMissingFiles(t, requiredFiles)

		return ctx
	}
}

func executeSupportArchiveCommand(ctx context.Context, t *testing.T, envConfig *envconf.Config, cmdLineArguments []string, namespace string) *pod.ExecutionResult {
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
		operator.ContainerName,
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
