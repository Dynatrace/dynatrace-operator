//go:build e2e

package support_archive

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"

	edgeconnectv1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha1/edgeconnect"
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/functional"
	"github.com/Dynatrace/dynatrace-operator/test/helpers"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/edgeconnect"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/operator"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/namespace"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/pod"
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

type CustomResources struct {
	dynakube    dynatracev1beta1.DynaKube
	edgeconnect edgeconnectv1beta1.EdgeConnect
}

func Feature(t *testing.T) features.Feature {
	builder := features.New("support archive execution")
	builder.WithLabel("name", "support-archive")
	secretConfig := tenant.GetSingleTenantSecret(t)
	edgeconnectSecretConfig := tenant.GetEdgeConnectTenantSecret(t)

	injectLabels := map[string]string{
		"inject": "me",
	}

	testDynakube := *dynakube.New(
		dynakube.WithNamespaceSelector(metav1.LabelSelector{
			MatchLabels: injectLabels,
		}),
		dynakube.WithApiUrl(secretConfig.ApiUrl),
		dynakube.WithCloudNativeSpec(&dynatracev1beta1.CloudNativeFullStackSpec{}),
		dynakube.WithActiveGate(),
	)

	testEdgeConnect := *edgeconnect.New(
		// this name should match with tenant edge connect name
		edgeconnect.WithName(edgeconnectSecretConfig.Name),
		edgeconnect.WithApiServer(edgeconnectSecretConfig.ApiServer),
		edgeconnect.WithOAuthClientSecret(fmt.Sprintf("%s-client-secret", edgeconnectSecretConfig.Name)),
		edgeconnect.WithOAuthEndpoint("https://sso-dev.dynatracelabs.com/sso/oauth2/token"),
		edgeconnect.WithOAuthResource(fmt.Sprintf("urn:dtenvironment:%s", edgeconnectSecretConfig.TenantUid)),
		edgeconnect.WithCustomPullSecret(fmt.Sprintf("%s-docker-pull-secret", edgeconnectSecretConfig.Name)),
	)

	builder.Assess("deploy injected namespace", namespace.Create(*namespace.New(testAppNameInjected, namespace.WithLabels(injectLabels))))
	builder.Assess("deploy NOT injected namespace", namespace.Create(*namespace.New(testAppNameNotInjected)))
	dynakube.Install(builder, helpers.LevelAssess, &secretConfig, testDynakube)
	edgeconnect.Install(builder, helpers.LevelAssess, &edgeconnectSecretConfig, testEdgeConnect)

	// Register actual test
	builder.Assess("support archive subcommand can be executed correctly with managed logs", testSupportArchiveCommand(testDynakube, testEdgeConnect, true))
	builder.Assess("support archive subcommand can be executed correctly without managed logs", testSupportArchiveCommand(testDynakube, testEdgeConnect, false))

	builder.WithTeardown("remove injected namespace", namespace.Delete(testAppNameInjected))
	builder.WithTeardown("remove NOT injected namespace", namespace.Delete(testAppNameNotInjected))
	dynakube.Delete(builder, helpers.LevelTeardown, testDynakube)
	builder.WithTeardown("remove edgeconnect CR", edgeconnect.Delete(testEdgeConnect))
	return builder.Feature()
}

func testSupportArchiveCommand(testDynakube dynatracev1beta1.DynaKube, testEdgeConnect edgeconnectv1beta1.EdgeConnect, collectManaged bool) features.Func {
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
			dynakube:    testDynakube,
			edgeconnect: testEdgeConnect,
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
	} else if !strings.HasSuffix(zipFileName, "_previous.log") {
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
