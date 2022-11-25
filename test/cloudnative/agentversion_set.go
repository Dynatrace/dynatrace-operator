//go:build e2e

package cloudnative

import (
	"context"
	"testing"
	"time"

	"github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/arch"
	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	"github.com/Dynatrace/dynatrace-operator/src/version"
	"github.com/Dynatrace/dynatrace-operator/test/csi"
	"github.com/Dynatrace/dynatrace-operator/test/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/kubeobjects/manifests"
	"github.com/Dynatrace/dynatrace-operator/test/kubeobjects/namespace"
	"github.com/Dynatrace/dynatrace-operator/test/operator"
	"github.com/Dynatrace/dynatrace-operator/test/sampleapps"
	"github.com/Dynatrace/dynatrace-operator/test/secrets"
	"github.com/Dynatrace/dynatrace-operator/test/setup"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/exp/rand"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/e2e-framework/klient/k8s"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/klient/wait/conditions"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

const VERSION = "VERSION"

var injectionLabel = map[string]string{
	"inject": "dynakube",
}

func AgentVersionSet(t *testing.T) features.Feature {
	secretConfig, err := secrets.DefaultSingleTenant(afero.NewOsFs())
	require.NoError(t, err)

	versions := getAvailableVersions(secretConfig, t)
	oldVersion, newVersion := assignVersions(t, versions, version.SemanticVersion{}, version.SemanticVersion{})

	dk := dynakube.NewBuilder().
		WithDefaultObjectMeta().
		WithDynakubeNamespaceSelector().
		ApiUrl(secretConfig.ApiUrl).
		CloudNativeWithAgentVersion(&v1beta1.CloudNativeFullStackSpec{}, oldVersion).
		Build()

	agentVersionset := features.New("cloudnative with agent version set")
	agentVersionset.Setup(namespace.Create(
		namespace.NewBuilder("test-namespace-1").
			WithLabels(injectionLabel).
			Build()),
	)
	agentVersionset.Setup(secrets.ApplyDefault(secretConfig))
	agentVersionset.Setup(operator.InstallAllForKubernetes())
	agentVersionset.Setup(manifests.InstallFromFile("../testdata/cloudnative/sample-deployment.yaml"))

	setup.AssessDeployment(agentVersionset)

	agentVersionset.Assess("install dynakube", dynakube.Apply(dk))

	assessVersionChecks(agentVersionset, oldVersion)

	agentVersionset.Assess("update dynakube", updateDynakube(newVersion))
	agentVersionset.Assess("dynakube phase changes to 'Running'",
		dynakube.WaitForDynakubePhase(dynakube.NewBuilder().WithDefaultObjectMeta().Build()))
	setup.AssessDeployment(agentVersionset)

	assessVersionChecks(agentVersionset, newVersion)

	return agentVersionset.Feature()
}

func getAvailableVersions(secret secrets.Secret, t *testing.T) []string {
	dtc, err := dtclient.NewClient(secret.ApiUrl, secret.ApiToken, secret.ApiToken)
	require.NoError(t, err)
	versions, err := dtc.GetAgentVersions(dtclient.OsUnix, dtclient.InstallerTypeDefault, arch.FlavorDefault, arch.Arch)
	require.NoError(t, err)

	return versions
}

func assessVersionChecks(builder *features.FeatureBuilder, version version.SemanticVersion) {
	builder.Assess("restart csi driver", restartCSIDriver)
	builder.Assess("start sample apps and injection", sampleapps.Restart)
	builder.Assess("check init containers", checkInitContainers)
	builder.Assess("check env vars of init container", checkVersionInSampleApp(version))
}

func assignVersions(t *testing.T, versions []string,
	old version.SemanticVersion, new version.SemanticVersion) (version.SemanticVersion, version.SemanticVersion) {

	lowerHalf := versions[0 : len(versions)/2]
	upperHalf := versions[len(versions)/2+1 : len(versions)-1]

	lowerValue := lowerHalf[rand.Intn(len(lowerHalf))]
	upperValue := upperHalf[rand.Intn(len(upperHalf))]

	lowerVersion, err := version.ExtractSemanticVersion(lowerValue)
	require.NoError(t, err)
	newerVersion, err := version.ExtractSemanticVersion(upperValue)
	require.NoError(t, err)

	compare := version.CompareSemanticVersions(lowerVersion, newerVersion)
	if compare > 0 {
		old = newerVersion
		new = lowerVersion
	} else if compare < 0 {
		old = lowerVersion
		new = newerVersion
	}
	return old, new
}

func updateDynakube(semanticVersion version.SemanticVersion) features.Func {
	return func(ctx context.Context, t *testing.T, environmentConfig *envconf.Config) context.Context {
		resources := environmentConfig.Client().Resources()
		var dk dynatracev1beta1.DynaKube

		err := dynatracev1beta1.AddToScheme(resources.GetScheme())
		require.NoError(t, err)

		err = resources.Get(ctx, "dynakube", "dynatrace", &dk)
		require.NoError(t, err)

		dk.Status.UpdatedTimestamp = metav1.Now()
		dk.Spec.OneAgent.CloudNativeFullStack.Version = semanticVersion.String()
		err = resources.Update(ctx, &dk)
		require.NoError(t, err)

		return ctx
	}
}

func restartCSIDriver(ctx context.Context, t *testing.T, environmentConfig *envconf.Config) context.Context {
	resources := environmentConfig.Client().Resources()
	var csiDaemonSet appsv1.DaemonSet

	err := csi.ForEachPod(ctx, resources, func(podItem corev1.Pod) {
		require.NoError(t, resources.Delete(ctx, &podItem))
	})

	require.NoError(t, resources.Get(ctx, "dynatrace-oneagent-csi-driver", "dynatrace", &csiDaemonSet))
	require.NoError(t, wait.For(conditions.New(resources).ResourceMatch(&csiDaemonSet, func(object k8s.Object) bool {
		daemonSet, isDaemonSet := object.(*appsv1.DaemonSet)
		return isDaemonSet && daemonSet.Status.NumberReady == daemonSet.Status.NumberAvailable
	}), wait.WithTimeout(10*time.Minute)))

	require.NoError(t, err)
	time.Sleep(time.Minute)

	return ctx
}

func checkVersionInSampleApp(semanticVersion version.SemanticVersion) features.Func {
	return func(ctx context.Context, t *testing.T, environmentConfig *envconf.Config) context.Context {
		time.Sleep(time.Minute)
		resources := environmentConfig.Client().Resources()
		pods := sampleapps.Get(t, ctx, resources)

		for _, podItem := range pods.Items {
			require.NotNil(t, podItem)
			require.NotNil(t, podItem.Spec)
			require.NotNil(t, podItem.Spec.InitContainers)

			for _, container := range podItem.Spec.InitContainers {
				if container.Name == "install-oneagent" {
					require.NotNil(t, container.Env)
					require.True(t, kubeobjects.EnvVarIsIn(container.Env, VERSION))
					for _, env := range container.Env {
						if env.Name == VERSION {
							require.NotNil(t, env.Value)
							assert.Equal(t, semanticVersion.String(), env.Value)
						}
					}
				}
			}
		}
		return ctx
	}
}
