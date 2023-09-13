//go:build e2e

package specific_agent_version

import (
	"context"
	"sort"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/src/api/status"
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1/dynakube"
	"github.com/Dynatrace/dynatrace-operator/src/arch"
	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
	"github.com/Dynatrace/dynatrace-operator/src/version"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/oneagent"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/setup"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/tenant"
	"github.com/Dynatrace/dynatrace-operator/test/scenarios/cloudnative"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

func specificAgentVersion(t *testing.T) features.Feature {
	builder := features.New("cloudnative with specific agent version")
	secretConfig := tenant.GetSingleTenantSecret(t)

	versions := getAvailableVersions(secretConfig, t)

	oldVersion, newVersion := assignVersions(t, versions)

	t.Logf("update %s -> %s", oldVersion, newVersion)

	dynakubeBuilder := dynakube.NewBuilder().
		WithDefaultObjectMeta().
		WithDynakubeNamespaceSelector().
		ApiUrl(secretConfig.ApiUrl).
		CloudNativeWithAgentVersion(cloudnative.DefaultCloudNativeSpec(), oldVersion)
	testDynakube := dynakubeBuilder.Build()

	// // Register operator install
	// assess.InstallOperatorFromSource(builder, testDynakube)
	// // Register actual test
	// assess.InstallDynakube(builder, &secretConfig, testDynakube)
	setup := setup.NewEnvironmentSetup(
		setup.CreateDefaultDynatraceNamespace(),
		setup.DeployOperatorViaMake(testDynakube.NeedsCSIDriver()),
		setup.CreateDynakube(secretConfig, testDynakube))
	setup.CreateSetupSteps(builder)
	builder.Assess("checking version of oneagent", assessVersionChecks(testDynakube))

	updatedDynakube := testDynakube.DeepCopy()
	updatedDynakube.Spec.OneAgent.CloudNativeFullStack.Version = newVersion.String()
	builder.Assess("update dynakube with new agent version", dynakube.Update(*updatedDynakube))
	builder.Assess("agents are redeploying", dynakube.WaitForDynakubePhase(*updatedDynakube, status.Deploying))
	builder.Assess("agents redeployed successfully", dynakube.WaitForDynakubePhase(*updatedDynakube, status.Running))
	builder.Assess("checking version of oneagent", assessVersionChecks(testDynakube))

	// Register sample, dynakube and operator uninstall
	// teardown.UninstallDynatrace(builder, testDynakube)
	setup.CreateTeardownSteps(builder)
	return builder.Feature()
}

func getAvailableVersions(secret tenant.Secret, t *testing.T) []string {
	dtc, err := dtclient.NewClient(secret.ApiUrl, secret.ApiToken, secret.ApiToken)
	require.NoError(t, err)
	versions, err := dtc.GetAgentVersions(dtclient.OsUnix, dtclient.InstallerTypeDefault, arch.FlavorDefault, arch.Arch)
	require.NoError(t, err)

	return versions
}

func assessVersionChecks(testDynakube dynatracev1beta1.DynaKube) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		daemonset, err := oneagent.Get(ctx, envConfig.Client().Resources(), testDynakube)
		require.NoError(t, err)
		require.Contains(t, daemonset.Spec.Template.Spec.Containers[0].Image, testDynakube.OneAgentVersion())
		return ctx
	}
}

// this method returns two different versions in order to be able to simulate the upgrade of version.
// Different versions mean different enough to make an update happen (different sprints or different release
// numbers in the same sprint).
func assignVersions(t *testing.T, stringVersions []string) (version.SemanticVersion, version.SemanticVersion) {
	require.GreaterOrEqual(t, len(stringVersions), 4)

	versions := make([]version.SemanticVersion, len(stringVersions))

	for i, stringVersion := range stringVersions {
		semanticVersion, err := version.ExtractSemanticVersion(stringVersion)
		require.NoError(t, err)
		versions[i] = semanticVersion
	}

	sort.Slice(versions, func(i, j int) bool {
		return version.CompareSemanticVersions(versions[i], versions[j]) < 0
	})

	secondVersion := versions[len(versions)-1]

	firstIndex := len(versions) / 2

	for firstIndex >= 0 {
		firstVersion := versions[firstIndex]

		if version.AreDevBuildsInTheSameSprint(firstVersion, secondVersion) {
			firstIndex--
			continue
		}

		return firstVersion, secondVersion
	}
	require.Fail(t, "two different versions not found (different sprint numbers or release numbers needed)", "versions", versions)
	return version.SemanticVersion{}, version.SemanticVersion{}
}
