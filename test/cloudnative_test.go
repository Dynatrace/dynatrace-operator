//go:build e2e

package test

import (
	"context"
	"os"
	"path"
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/test/activegate"
	"github.com/Dynatrace/dynatrace-operator/test/csi"
	"github.com/Dynatrace/dynatrace-operator/test/kubeobjects/environment"
	"github.com/Dynatrace/dynatrace-operator/test/kubeobjects/manifests"
	"github.com/Dynatrace/dynatrace-operator/test/kubeobjects/namespace"
	"github.com/Dynatrace/dynatrace-operator/test/oneagent"
	"github.com/Dynatrace/dynatrace-operator/test/operator"
	"github.com/Dynatrace/dynatrace-operator/test/secrets"
	"github.com/Dynatrace/dynatrace-operator/test/webhook"
	"github.com/pkg/errors"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/e2e-framework/klient/k8s"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/klient/wait/conditions"
	"sigs.k8s.io/e2e-framework/pkg/env"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

const (
	dynakubeName       = "dynakube"
	dynatraceNamespace = "dynatrace"

	sampleAppsName      = "myapp"
	sampleAppsNamespace = "test-namespace-1"

	oneAgentInstallContainerName = "install-oneagent"

	installSecretsPath = "/testdata/secrets/cloudnative-install.yaml"
)

var testEnvironment env.Environment

func TestMain(m *testing.M) {
	testEnvironment = environment.Get()
	testEnvironment.Setup(deleteDynakubeIfExists())
	testEnvironment.Setup(oneagent.WaitForDaemonSetPodsDeletion())
	testEnvironment.Setup(namespace.DeleteIfExists(sampleAppsNamespace))
	testEnvironment.Setup(namespace.Recreate(dynatraceNamespace))

	testEnvironment.Finish(deleteDynakubeIfExists())
	testEnvironment.Finish(oneagent.WaitForDaemonSetPodsDeletion())
	testEnvironment.Finish(namespace.Delete(sampleAppsNamespace))
	testEnvironment.Finish(namespace.Delete(dynatraceNamespace))

	testEnvironment.Run(m)
}

func TestCloudNative(t *testing.T) {
	testEnvironment.Test(t, install(t))
}

func install(t *testing.T) features.Feature {
	currentWorkingDirectory, err := os.Getwd()
	require.NoError(t, err)

	secretPath := path.Join(currentWorkingDirectory, installSecretsPath)
	secretConfig, err := secrets.NewFromConfig(afero.NewOsFs(), secretPath)

	require.NoError(t, err)

	defaultInstallation := features.New("default installation")

	defaultInstallation.Setup(secrets.ApplyDefault(secretConfig))
	defaultInstallation.Setup(operator.InstallForKubernetes())
	defaultInstallation.Setup(deploySampleApps())

	defaultInstallation.Assess("operator started", operator.WaitForDeployment())
	defaultInstallation.Assess("webhook started", webhook.WaitForDeployment())
	defaultInstallation.Assess("csi driver started", csi.WaitForDaemonset())
	defaultInstallation.Assess("dynakube applied", applyDynakube(secretConfig))
	defaultInstallation.Assess("activegate started", activegate.WaitForStatefulSet())
	defaultInstallation.Assess("oneagent started", oneagent.WaitForDaemonset())
	defaultInstallation.Assess("dynakube phase changes to 'Running'", waitForDynakubePhase())
	defaultInstallation.Assess("restart sample apps", restartSampleApps)
	defaultInstallation.Assess("sample apps have init containers", checkForInitContainers)

	return defaultInstallation.Feature()
}

func checkForInitContainers(ctx context.Context, t *testing.T, config *envconf.Config) context.Context {
	var pods corev1.PodList
	resources := config.Client().Resources()

	require.NoError(t, resources.WithNamespace(sampleAppsNamespace).List(ctx, &pods))

	for _, pod := range pods.Items {
		require.NotNil(t, pod)
		require.NotNil(t, pod.Spec)
		require.NotEmpty(t, pod.Spec.InitContainers)

		oneAgentInstallContainer := pod.Spec.InitContainers[0]
		assert.Equal(t, oneAgentInstallContainerName, oneAgentInstallContainer.Name)
	}

	return ctx
}

func restartSampleApps(ctx context.Context, t *testing.T, config *envconf.Config) context.Context {
	var pods corev1.PodList
	resources := config.Client().Resources()

	require.NoError(t, resources.WithNamespace(sampleAppsNamespace).List(ctx, &pods))

	for _, pod := range pods.Items {
		require.NoError(t, resources.Delete(ctx, &pod))
	}

	var sampleDeployment appsv1.Deployment
	require.NoError(t, resources.Get(ctx, sampleAppsName, sampleAppsNamespace, &sampleDeployment))

	require.NoError(t, wait.For(
		conditions.New(resources).DeploymentConditionMatch(
			&sampleDeployment, appsv1.DeploymentAvailable, corev1.ConditionTrue)))

	return ctx
}

func dynakube() dynatracev1beta1.DynaKube {
	return dynatracev1beta1.DynaKube{
		ObjectMeta: v1.ObjectMeta{
			Name:      dynakubeName,
			Namespace: dynatraceNamespace,
		},
	}
}

func applyDynakube(secretConfig secrets.Secrets) features.Func {
	return func(ctx context.Context, t *testing.T, environmentConfig *envconf.Config) context.Context {
		require.NoError(t, dynatracev1beta1.AddToScheme(environmentConfig.Client().Resources().GetScheme()))

		instance := dynakube()
		instance.Spec = dynatracev1beta1.DynaKubeSpec{
			APIURL: secretConfig.ApiUrl,
			OneAgent: dynatracev1beta1.OneAgentSpec{
				CloudNativeFullStack: &dynatracev1beta1.CloudNativeFullStackSpec{},
			},
			ActiveGate: dynatracev1beta1.ActiveGateSpec{
				Capabilities: []dynatracev1beta1.CapabilityDisplayName{
					dynatracev1beta1.KubeMonCapability.DisplayName,
					dynatracev1beta1.DynatraceApiCapability.DisplayName,
					dynatracev1beta1.RoutingCapability.DisplayName,
					dynatracev1beta1.MetricsIngestCapability.DisplayName,
					dynatracev1beta1.StatsdIngestCapability.DisplayName,
				},
			},
		}

		require.NoError(t, environmentConfig.Client().Resources().Create(ctx, &instance))

		return ctx
	}
}

func deleteDynakubeIfExists() env.Func {
	return func(ctx context.Context, environmentConfig *envconf.Config) (context.Context, error) {
		instance := dynakube()
		resources := environmentConfig.Client().Resources()

		err := dynatracev1beta1.AddToScheme(resources.GetScheme())

		if err != nil {
			return ctx, errors.WithStack(err)
		}

		err = resources.Delete(ctx, &instance)
		_, isNoKindMatchErr := err.(*meta.NoKindMatchError)

		if err != nil {
			if k8serrors.IsNotFound(err) || isNoKindMatchErr {
				// If the dynakube itself or the crd does not exist, everything is fine
				err = nil
			}

			return ctx, errors.WithStack(err)
		}

		err = wait.For(conditions.New(resources).ResourceDeleted(&instance))

		return ctx, err
	}
}

func waitForDynakubePhase() features.Func {
	return func(ctx context.Context, t *testing.T, environmentConfig *envconf.Config) context.Context {
		instance := dynakube()
		resources := environmentConfig.Client().Resources()

		require.NoError(t, wait.For(conditions.New(resources).ResourceMatch(&instance, func(object k8s.Object) bool {
			dynakube, isDynakube := object.(*dynatracev1beta1.DynaKube)
			return isDynakube && dynakube.Status.Phase == dynatracev1beta1.Running
		})))

		return ctx
	}
}

func deploySampleApps() features.Func {
	return manifests.InstallFromFile("./testdata/cloudnative/sample-deployment.yaml")
}
