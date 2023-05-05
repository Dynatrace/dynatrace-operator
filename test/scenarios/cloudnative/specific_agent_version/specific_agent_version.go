//go:build e2e

package specific_agent_version

import (
	"context"
	"sort"
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/arch"
	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
	"github.com/Dynatrace/dynatrace-operator/src/version"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/oneagent"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/steps/assess"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/steps/teardown"
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
	sort.Strings(versions)
	oldVersion, newVersion := assignVersions(t, versions, version.SemanticVersion{}, version.SemanticVersion{})

	dynakubeBuilder := dynakube.NewBuilder().
		WithDefaultObjectMeta().
		WithDynakubeNamespaceSelector().
		ApiUrl(secretConfig.ApiUrl).
		CloudNativeWithAgentVersion(cloudnative.DefaultCloudNativeSpec(), oldVersion)
	testDynakube := dynakubeBuilder.Build()

	// Register operator install
	assess.InstallOperatorFromSource(builder, testDynakube)

	// Register actual test
	assess.InstallDynakube(builder, &secretConfig, testDynakube)
	builder.Assess("checking version of oneagent", assessVersionChecks(testDynakube))

	updatedDynakube := testDynakube.DeepCopy()
	updatedDynakube.Spec.OneAgent.CloudNativeFullStack.Version = newVersion.String()
	builder.Assess("update dynakube with new agent version", dynakube.Update(*updatedDynakube))
	builder.Assess("agents are redeploying", dynakube.WaitForDynakubePhase(*updatedDynakube, dynatracev1beta1.Deploying))
	builder.Assess("agents redeployed successfully", dynakube.WaitForDynakubePhase(*updatedDynakube, dynatracev1beta1.Running))
	builder.Assess("checking version of oneagent", assessVersionChecks(testDynakube))

	// Register sample, dynakube and operator uninstall
	teardown.UninstallDynatrace(builder, testDynakube)

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
	return func(ctx context.Context, t *testing.T, environmentConfig *envconf.Config) context.Context {
		daemonset, err := oneagent.Get(ctx, environmentConfig.Client().Resources(), testDynakube)
		require.NoError(t, err)
		require.Contains(t, daemonset.Spec.Template.Spec.Containers[0].Image, testDynakube.OneAgentVersion())
		return ctx
	}
}

// this method assigns the queried versions to the variables 'old' and 'new'.
// it makes sure, that old gets an older version than new, in order to
// be able to simulate the upgrade of version.
func assignVersions(t *testing.T, versions []string, old version.SemanticVersion, new version.SemanticVersion) (version.SemanticVersion, version.SemanticVersion) {
	require.GreaterOrEqual(t, len(versions), 4)
	first := versions[len(versions)/2]
	second := versions[len(versions)-1]

	firstVersion, err := version.ExtractSemanticVersion(first)
	require.NoError(t, err)
	secondVersion, err := version.ExtractSemanticVersion(second)
	require.NoError(t, err)

	compare := version.CompareSemanticVersions(firstVersion, secondVersion)
	if compare > 0 {
		old = secondVersion
		new = firstVersion
	} else if compare < 0 {
		old = firstVersion
		new = secondVersion
	}
	return old, new
}
