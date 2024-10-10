//go:build e2e

package support_archive

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"os"
	"path"
	"strings"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha2/edgeconnect"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	agconsts "github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/functional"
	"github.com/Dynatrace/dynatrace-operator/test/features/consts"
	"github.com/Dynatrace/dynatrace-operator/test/helpers"
	dynakubeComponents "github.com/Dynatrace/dynatrace-operator/test/helpers/components/dynakube"
	edgeconnectComponents "github.com/Dynatrace/dynatrace-operator/test/helpers/components/edgeconnect"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/operator"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/namespace"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/pod"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/secret"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/statefulset"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/tenant"
	"github.com/Dynatrace/dynatrace-operator/test/project"
	"github.com/google/uuid"
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
		dynakubeComponents.WithActiveGateTLSSecret(consts.AgSecretName),
		dynakubeComponents.WithCustomPullSecret(consts.DevRegistryPullSecretName),
		dynakubeComponents.WithExtensionsEnabledSpec(true),
		dynakubeComponents.WithExtensionsEECImageRefSpec(consts.EecImageRepo, consts.EecImageTag),
	)

	testECname := uuid.NewString()
	testHostPattern := fmt.Sprintf("%s.e2eTestHostPattern.internal.org", testECname)
	edgeConnectTenantConfig := &edgeconnectComponents.TenantConfig{}

	builder.Assess("create EC configuration on the tenant", edgeconnectComponents.CreateTenantConfig(testECname, edgeconnectSecretConfig, edgeConnectTenantConfig, testHostPattern))

	testEdgeConnect := *edgeconnectComponents.New(
		edgeconnectComponents.WithName(testECname),
		edgeconnectComponents.WithApiServer(edgeconnectSecretConfig.ApiServer),
		edgeconnectComponents.WithOAuthClientSecret(edgeconnectComponents.BuildOAuthClientSecretName(testECname)),
		edgeconnectComponents.WithOAuthEndpoint("https://sso-dev.dynatracelabs.com/sso/oauth2/token"),
		edgeconnectComponents.WithOAuthResource(fmt.Sprintf("urn:dtenvironment:%s", edgeconnectSecretConfig.TenantUid)),
	)

	// create OAuth client secret related to the specific EdgeConnect configuration on the tenant
	builder.Assess("create client secret", tenant.CreateClientSecret(&edgeConnectTenantConfig.Secret, edgeconnectComponents.BuildOAuthClientSecretName(testEdgeConnect.Name), testEdgeConnect.Namespace))

	builder.Assess("deploy injected namespace", namespace.Create(*namespace.New(testAppNameInjected, namespace.WithLabels(injectLabels))))
	builder.Assess("deploy NOT injected namespace", namespace.Create(*namespace.New(testAppNameNotInjected)))

	agCrt, err := os.ReadFile(path.Join(project.TestDataDir(), consts.AgCertificate))
	require.NoError(t, err)

	agP12, err := os.ReadFile(path.Join(project.TestDataDir(), consts.AgCertificateAndPrivateKey))
	require.NoError(t, err)

	agSecret := secret.New(consts.AgSecretName, testDynakube.Namespace,
		map[string][]byte{
			dynakube.TLSCertKey:                    agCrt,
			consts.AgCertificateAndPrivateKeyField: agP12,
		})
	builder.Assess("create AG TLS secret", secret.Create(agSecret))

	dynakubeComponents.Install(builder, helpers.LevelAssess, &secretConfig, testDynakube)
	edgeconnectComponents.Install(builder, helpers.LevelAssess, nil, testEdgeConnect)
	builder.Assess("check EC configuration on the tenant", edgeconnectComponents.CheckEcExistsOnTheTenant(edgeconnectSecretConfig, edgeConnectTenantConfig))

	// check if components are running
	builder.Assess("active gate pod is running", statefulset.WaitFor(testDynakube.Name+"-"+agconsts.MultiActiveGateName, testDynakube.Namespace))
	builder.Assess("extensions execution controller started", statefulset.WaitFor(testDynakube.ExtensionsExecutionControllerStatefulsetName(), testDynakube.Namespace))
	builder.Assess("extension collector started", statefulset.WaitFor(testDynakube.ExtensionsCollectorStatefulsetName(), testDynakube.Namespace))

	// Register actual test
	builder.Assess("support archive subcommand can be executed correctly with managed logs", testSupportArchiveCommand(testDynakube, testEdgeConnect, true))
	builder.Assess("support archive subcommand can be executed correctly without managed logs", testSupportArchiveCommand(testDynakube, testEdgeConnect, false))

	builder.WithTeardown("remove injected namespace", namespace.Delete(testAppNameInjected))
	builder.WithTeardown("remove NOT injected namespace", namespace.Delete(testAppNameNotInjected))
	dynakubeComponents.Delete(builder, helpers.LevelTeardown, testDynakube)
	builder.WithTeardown("remove edgeconnect CR", edgeconnectComponents.Delete(testEdgeConnect))
	builder.Teardown(tenant.DeleteTenantSecret(edgeconnectComponents.BuildOAuthClientSecretName(testEdgeConnect.Name), testEdgeConnect.Namespace))
	builder.Teardown(edgeconnectComponents.DeleteTenantConfig(edgeconnectSecretConfig, edgeConnectTenantConfig))
	builder.WithTeardown("deleted tenant secret", tenant.DeleteTenantSecret(testDynakube.Name, testDynakube.Namespace))
	builder.WithTeardown("deleted ag secret", secret.Delete(agSecret))

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
