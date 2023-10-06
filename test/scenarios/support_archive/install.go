//go:build e2e

package support_archive

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"github.com/Dynatrace/dynatrace-operator/src/util/functional"
	"strings"
	"testing"

	edgeconnectv1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1alpha1/edgeconnect"
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/edgeconnect"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/operator"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/namespace"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/pod"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/setup"
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

func supportArchiveExecution(t *testing.T) features.Feature {
	builder := features.New("support archive execution")
	secretConfig := tenant.GetSingleTenantSecret(t)
	edgeconnectSecretConfig := tenant.GetEdgeConnectTenantSecret(t)

	injectLabels := map[string]string{
		"inject": "me",
	}

	testDynakube := dynakube.NewBuilder().
		WithDefaultObjectMeta().
		NamespaceSelector(metav1.LabelSelector{
			MatchLabels: injectLabels,
		}).
		ApiUrl(secretConfig.ApiUrl).
		CloudNative(&dynatracev1beta1.CloudNativeFullStackSpec{}).
		WithActiveGate().Build()

	testEdgeConnect := edgeconnect.NewBuilder().
		// this name should match with tenant edge connect name
		Name(edgeconnectSecretConfig.Name).
		ApiServer(edgeconnectSecretConfig.ApiServer).
		OAuthClientSecret(fmt.Sprintf("%s-client-secret", edgeconnectSecretConfig.Name)).
		OAuthEndpoint("https://sso-dev.dynatracelabs.com/sso/oauth2/token").
		OAuthResource(fmt.Sprintf("urn:dtenvironment:%s", edgeconnectSecretConfig.TenantUid)).
		CustomPullSecret(fmt.Sprintf("%s-docker-pull-secret", edgeconnectSecretConfig.Name)).
		Build()

	setup.CreateFeatureEnvironment(builder,
		setup.CreateNamespace(namespace.NewBuilder(testAppNameInjected).WithLabels(injectLabels).Build()),
		setup.CreateNamespace(namespace.NewBuilder(testAppNameNotInjected).Build()),
		setup.CreateNamespaceWithoutTeardown(namespace.NewBuilder(testDynakube.Namespace).Build()),
		setup.DeployOperatorViaMake(testDynakube.NeedsCSIDriver()),
		setup.CreateDynakube(secretConfig, testDynakube),
		setup.CreateEdgeConnect(edgeconnectSecretConfig, testEdgeConnect),
	)
	// Register actual test
	builder.Assess("support archive subcommand can be executed correctly with managed logs", testSupportArchiveCommand(testDynakube, testEdgeConnect, true))
	builder.Assess("support archive subcommand can be executed correctly without managed logs", testSupportArchiveCommand(testDynakube, testEdgeConnect, false))

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
