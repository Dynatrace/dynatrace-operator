//go:build e2e

package usepublicregistry

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/exp"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/extensions"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/oneagent"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/image"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/features/cloudnative"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/features/consts"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/components/activegate"
	dynakubeComponents "github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/components/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/components/operator"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/kubernetes/objects/k8sdaemonset"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/kubernetes/objects/k8sdeployment"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/kubernetes/objects/k8snamespace"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/kubernetes/objects/k8ssecret"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/kubernetes/objects/k8sstatefulset"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/platform"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/sample"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/tenant"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/project"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

const (
	publicRegistryOverrideEnvVar = "E2E_PUBLIC_REGISTRY_OVERRIDE"
)

func publicRegistryOverride(t *testing.T) string {
	t.Helper()
	val := os.Getenv(publicRegistryOverrideEnvVar)
	if val == "" {
		t.Skipf("%s must be set", publicRegistryOverrideEnvVar)
	}

	return val
}

// requireDevRegistrySecret returns a Setup feature.Func that fails the test if
// the devregistry pull secret is not already present in the operator namespace.
// The secret is expected to be provisioned out-of-band on the test cluster (see
// CONTRIBUTING.md); without it, the operator cannot pull images from the
// configured registry, so the test cannot meaningfully run.
func requireDevRegistrySecret() features.Func {
	return k8ssecret.Exists(consts.DevRegistryPullSecretName, operator.DefaultNamespace)
}

func OneAgent(t *testing.T) features.Feature {
	return oneAgentFeature(t, "use-public-registry-oneagent", "use-public-registry-oa", "")
}

func OneAgentWithOverride(t *testing.T) features.Feature {
	return oneAgentFeature(t,
		"use-public-registry-oneagent-with-override",
		"use-public-registry-oa-ovrd",
		publicRegistryOverride(t))
}

func ActiveGate(t *testing.T) features.Feature {
	return activeGateFeature(t, "use-public-registry-activegate", "use-public-registry-ag", "")
}

func ActiveGateWithOverride(t *testing.T) features.Feature {
	return activeGateFeature(t,
		"use-public-registry-activegate-with-override",
		"use-public-registry-ag-ovrd",
		publicRegistryOverride(t))
}

func CodeModules(t *testing.T) features.Feature {
	return codeModulesFeature(t,
		"use-public-registry-codemodules",
		"use-public-registry-cm",
		"use-public-registry-cm-sample",
		"")
}

func CodeModulesWithOverride(t *testing.T) features.Feature {
	return codeModulesFeature(t,
		"use-public-registry-codemodules-with-override",
		"use-public-registry-cm-ovrd",
		"use-public-registry-cm-sample-ovrd",
		publicRegistryOverride(t))
}

func DBExecutor(t *testing.T) features.Feature {
	return dbExecutorFeature(t, "use-public-registry-db-executor", "")
}

func DBExecutorOverride(t *testing.T) features.Feature {
	return dbExecutorFeature(t, "use-public-registry-db-executor-with-override", "")
}

func LogMon(t *testing.T) features.Feature {
	return logMonFeature(t,
		"use-public-registry-logmon",
		"")
}

func LogMonWithOverride(t *testing.T) features.Feature {
	return logMonFeature(t,
		"use-public-registry-logmon-with-override",
		publicRegistryOverride(t))
}

func oneAgentFeature(t *testing.T, featureName, dkName, override string) features.Feature {
	builder := features.New(featureName)
	builder.Assess("devregistry pull secret exists", requireDevRegistrySecret())

	secretConfig := tenant.GetSingleTenantSecret(t)

	options := []dynakubeComponents.Option{
		dynakubeComponents.WithName(dkName),
		dynakubeComponents.WithAPIURL(secretConfig.APIURL),
		dynakubeComponents.WithCloudNativeSpec(cloudnative.DefaultCloudNativeSpec()),
	}
	if override != "" {
		options = append(options, dynakubeComponents.WithPublicRegistryOverride(override))
	}
	if !tenant.UsePlatformToken() {
		options = append(options, dynakubeComponents.WithUsePublicRegistryFF())
	}

	testDynakube := *dynakubeComponents.New(options...)

	dynakubeComponents.Install(builder, &secretConfig, testDynakube)

	builder.Assess("OneAgent DaemonSet ready",
		k8sdaemonset.IsReady(testDynakube.OneAgent().GetDaemonsetName(), testDynakube.Namespace))
	builder.Assess("OneAgent status reports public-registry source",
		statusSourceIsPublicRegistry(testDynakube, image.OneAgent))

	return builder.Feature()
}

func activeGateFeature(t *testing.T, featureName, dkName, override string) features.Feature {
	builder := features.New(featureName)
	builder.Assess("devregistry pull secret exists", requireDevRegistrySecret())

	secretConfig := tenant.GetSingleTenantSecret(t)

	options := []dynakubeComponents.Option{
		dynakubeComponents.WithName(dkName),
		dynakubeComponents.WithAPIURL(secretConfig.APIURL),
		dynakubeComponents.WithActiveGate(),
	}
	if override != "" {
		options = append(options, dynakubeComponents.WithPublicRegistryOverride(override))
	}
	if !tenant.UsePlatformToken() {
		options = append(options, dynakubeComponents.WithUsePublicRegistryFF())
	}

	testDynakube := *dynakubeComponents.New(options...)

	dynakubeComponents.Install(builder, &secretConfig, testDynakube)

	const agComponent = "activegate"
	builder.Assess("ActiveGate StatefulSet ready",
		k8sstatefulset.IsReady(activegate.GetActiveGateStateFulSetName(&testDynakube, agComponent), testDynakube.Namespace))
	builder.Assess("ActiveGate status reports public-registry source",
		statusSourceIsPublicRegistry(testDynakube, image.ActiveGate))

	return builder.Feature()
}

func codeModulesFeature(t *testing.T, featureName, dkName, sampleNamespaceName, override string) features.Feature {
	builder := features.New(featureName)
	builder.Assess("devregistry pull secret exists", requireDevRegistrySecret())

	secretConfig := tenant.GetSingleTenantSecret(t)

	options := []dynakubeComponents.Option{
		dynakubeComponents.WithName(dkName),
		dynakubeComponents.WithAPIURL(secretConfig.APIURL),
		dynakubeComponents.WithApplicationMonitoringSpec(&oneagent.ApplicationMonitoringSpec{}),
	}
	if override != "" {
		options = append(options, dynakubeComponents.WithPublicRegistryOverride(override))
	}
	if !tenant.UsePlatformToken() {
		options = append(options, dynakubeComponents.WithUsePublicRegistryFF())
	}

	testDynakube := *dynakubeComponents.New(options...)

	sampleNamespace := *k8snamespace.New(sampleNamespaceName)
	sampleApp := sample.NewApp(t, &testDynakube,
		sample.AsDeployment(),
		sample.WithNamespace(sampleNamespace),
		// The injected init container image is pulled by kubelet from the user's
		// namespace, so the user's pod must reference the registry pull secret.
		sample.WithImagePullSecret(consts.DevRegistryPullSecretName),
	)

	builder.Assess("create sample namespace", sampleApp.InstallNamespace())

	dynakubeComponents.Install(builder, &secretConfig, testDynakube)

	builder.Assess("install sample app", sampleApp.Install())
	builder.Assess("CodeModules status reports public-registry source",
		statusSourceIsPublicRegistry(testDynakube, image.CodeModules))

	builder.Teardown(sampleApp.Uninstall())

	return builder.Feature()
}

func dbExecutorFeature(t *testing.T, featureName, override string) features.Feature {
	builder := features.New(featureName)
	builder.Assess("devregistry pull secret exists", requireDevRegistrySecret())
	testDatabaseID := "mysql"

	secretConfig := tenant.GetSingleTenantSecret(t)

	options := []dynakubeComponents.Option{
		dynakubeComponents.WithAPIURL(secretConfig.APIURL),
		dynakubeComponents.WithExtensionsEECImageRef(t),
		dynakubeComponents.WithExtensionsDatabases(extensions.DatabaseSpec{ID: testDatabaseID + "-a"}, extensions.DatabaseSpec{ID: testDatabaseID + "-b"}, extensions.DatabaseSpec{ID: testDatabaseID + "-c"}),
		dynakubeComponents.WithActiveGate(),
	}
	if override != "" {
		options = append(options, dynakubeComponents.WithPublicRegistryOverride(override))
	}
	if !tenant.UsePlatformToken() {
		options = append(options, dynakubeComponents.WithUsePublicRegistryFF())
	}

	testDynakube := *dynakubeComponents.New(options...)

	dynakubeComponents.Install(builder, &secretConfig, testDynakube)

	builder.Assess("active gate pod is running", activegate.CheckContainer(&testDynakube))

	builder.Assess("extensions execution controller started", k8sstatefulset.IsReady(testDynakube.Extensions().GetExecutionControllerStatefulsetName(), testDynakube.Namespace))

	builder.Assess("extensions db-a datasource deployment started", k8sdeployment.IsReady(testDynakube.Extensions().GetDatabaseDatasourceName(testDatabaseID+"-a"), testDynakube.Namespace))
	builder.Assess("extensions db-b datasource deployment started", k8sdeployment.IsReady(testDynakube.Extensions().GetDatabaseDatasourceName(testDatabaseID+"-b"), testDynakube.Namespace))
	builder.Assess("extensions db-c datasource deployment started", k8sdeployment.IsReady(testDynakube.Extensions().GetDatabaseDatasourceName(testDatabaseID+"-c"), testDynakube.Namespace))

	return builder.Feature()
}

func logMonFeature(t *testing.T, featureName, override string) features.Feature {
	builder := features.New(featureName)
	builder.Assess("devregistry pull secret exists", requireDevRegistrySecret())

	secretConfig := tenant.GetSingleTenantSecret(t)

	options := []dynakubeComponents.Option{
		dynakubeComponents.WithAPIURL(secretConfig.APIURL),
		dynakubeComponents.WithLogMonitoring(),
		dynakubeComponents.WithLogMonitoringImageRef(t),
		dynakubeComponents.WithActiveGate(),
		dynakubeComponents.WithActiveGateTLSSecret(consts.AgSecretName),
	}
	if override != "" {
		options = append(options, dynakubeComponents.WithPublicRegistryOverride(override))
	}
	if !tenant.UsePlatformToken() {
		options = append(options, dynakubeComponents.WithUsePublicRegistryFF())
	}

	isOpenshift, err := platform.NewResolver().IsOpenshift()
	require.NoError(t, err)
	if isOpenshift {
		options = append(options, dynakubeComponents.WithAnnotations(map[string]string{
			exp.OAPrivilegedKey: "true",
		}))
	}

	testDynakube := *dynakubeComponents.New(options...)

	agCrt, err := os.ReadFile(filepath.Join(project.TestDataDir(), consts.AgCertificate))
	require.NoError(t, err)

	agP12, err := os.ReadFile(filepath.Join(project.TestDataDir(), consts.AgCertificateAndPrivateKey))
	require.NoError(t, err)

	agSecret := k8ssecret.New(consts.AgSecretName, testDynakube.Namespace,
		map[string][]byte{
			dynakube.ServerCertKey:                 agCrt,
			consts.AgCertificateAndPrivateKeyField: agP12,
		})
	builder.Assess("create AG TLS secret", k8ssecret.Create(agSecret))

	dynakubeComponents.Install(builder, &secretConfig, testDynakube)

	builder.Assess("active gate pod is running", activegate.CheckContainer(&testDynakube))

	builder.Assess("log agent started", k8sdaemonset.IsReady(testDynakube.LogMonitoring().GetDaemonSetName(), testDynakube.Namespace))

	builder.WithTeardown("deleted ag secret", k8ssecret.Delete(agSecret))

	return builder.Feature()
}

// statusSourceIsPublicRegistry refetches the DynaKube and verifies that the
// given component's reported version-source is status.PublicRegistryVersionSource.
// This is the operator's own signal that the use-public-registry FF resolution
// path was taken, so the assertion is identical for the with-override and
// without-override flavors.
func statusSourceIsPublicRegistry(dk dynakube.DynaKube, component image.ComponentType) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		var current dynakube.DynaKube
		require.NoError(t,
			envConfig.Client().Resources().Get(ctx, dk.Name, dk.Namespace, &current))

		var actual status.VersionSource
		switch component {
		case image.OneAgent:
			actual = current.Status.OneAgent.Source
		case image.ActiveGate:
			actual = current.Status.ActiveGate.Source
		case image.CodeModules:
			actual = current.Status.CodeModules.Source
		default:
			require.Failf(t, "unknown component", "unknown component %q", component)

			return ctx
		}

		assert.Equalf(t, status.PublicRegistryVersionSource, actual,
			"expected %s status.source == %q, got %q", component, status.PublicRegistryVersionSource, actual)

		return ctx
	}
}
