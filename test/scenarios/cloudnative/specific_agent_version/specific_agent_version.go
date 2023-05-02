//go:build e2e

package specific_agent_version

import (
	"context"
	"sort"
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/arch"
	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	"github.com/Dynatrace/dynatrace-operator/src/version"
	"github.com/Dynatrace/dynatrace-operator/src/webhook"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/namespace"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/sampleapps"
	sample "github.com/Dynatrace/dynatrace-operator/test/helpers/sampleapps/base"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/steps/assess"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/steps/teardown"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/tenant"
	"github.com/Dynatrace/dynatrace-operator/test/scenarios/cloudnative"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

const agentVersion = "VERSION"

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
	sampleNamespace := namespace.NewBuilder("specific-agent-sample").WithLabels(testDynakube.NamespaceSelector().MatchLabels).Build()
	sampleApp := sampleapps.NewSampleDeployment(t, testDynakube)
	sampleApp.WithNamespace(sampleNamespace)
	builder.Assess("create sample namespace", sampleApp.InstallNamespace())

	// Register sample app install
	builder.Assess("install sample", sampleApp.Install())

	// Register operator install
	assess.InstallOperatorFromSource(builder, testDynakube)

	// Register actual test
	assess.InstallDynakube(builder, &secretConfig, testDynakube)
	assessVersionChecks(builder, oldVersion, sampleApp)
	builder.Assess("update dynakube", updateDynakube(testDynakube, newVersion))
	assessVersionChecks(builder, newVersion, sampleApp)

	// Register sample, dynakube and operator uninstall
	builder.Teardown(sampleApp.UninstallNamespace())
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

func assessVersionChecks(builder *features.FeatureBuilder, version version.SemanticVersion, sampleApp sample.App) {
	builder.Assess("restart sample apps", sampleApp.Restart)
	cloudnative.AssessSampleInitContainers(builder, sampleApp)
	builder.Assess("check env vars of init container", checkVersionInSampleApp(version, sampleApp))
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

func updateDynakube(testDynakube dynatracev1beta1.DynaKube, semanticVersion version.SemanticVersion) features.Func {
	return func(ctx context.Context, t *testing.T, environmentConfig *envconf.Config) context.Context {
		resources := environmentConfig.Client().Resources()
		var dk dynatracev1beta1.DynaKube

		err := dynatracev1beta1.AddToScheme(resources.GetScheme())
		require.NoError(t, err)

		err = resources.Get(ctx, testDynakube.Name, testDynakube.Namespace, &dk)
		require.NoError(t, err)

		dk.Status.UpdatedTimestamp = metav1.Now()
		dk.Spec.OneAgent.CloudNativeFullStack.Version = semanticVersion.String()
		err = resources.Update(ctx, &dk)
		require.NoError(t, err)

		return ctx
	}
}

func checkVersionInSampleApp(semanticVersion version.SemanticVersion, sampleApp sample.App) features.Func {
	return func(ctx context.Context, t *testing.T, environmentConfig *envconf.Config) context.Context {
		resources := environmentConfig.Client().Resources()
		pods := sampleApp.GetPods(ctx, t, resources)

		for _, podItem := range pods.Items {
			require.NotNil(t, podItem)
			require.NotNil(t, podItem.Spec)
			require.NotNil(t, podItem.Spec.InitContainers)
			initContainer := podItem.Spec.InitContainers[0]
			checkVersionInInitContainer(t, initContainer, semanticVersion)
		}
		return ctx
	}
}

func checkVersionInInitContainer(t *testing.T, initContainer corev1.Container, semanticVersion version.SemanticVersion) {
	if initContainer.Name == webhook.InstallContainerName {
		require.NotNil(t, initContainer.Env)
		require.True(t, kubeobjects.EnvVarIsIn(initContainer.Env, agentVersion))
		for _, env := range initContainer.Env {
			if env.Name == agentVersion {
				require.NotNil(t, env.Value)
				assert.Equal(t, semanticVersion.String(), env.Value)
			}
		}
	}
}
