//go:build e2e

package cloudnative

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/test/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/kubeobjects/manifests"
	"github.com/Dynatrace/dynatrace-operator/test/setup"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

func Upgrade(t *testing.T, istioEnabled bool) features.Feature {
	defaultInstallation := features.New("default installation")

	installReleasedOperatorAndDeploySampleApps(t, defaultInstallation, "v0.9.1", istioEnabled)

	// update to snapshot
	setup.InstallDynatraceFromSource(defaultInstallation, nil)
	setup.AssessOperatorDeployment(defaultInstallation)

	assessSampleAppsRestartHalf(defaultInstallation)
	assessOneAgentsAreRunning(defaultInstallation)

	return defaultInstallation.Feature()
}

func installReleasedOperatorAndDeploySampleApps(t *testing.T, defaultInstallation *features.FeatureBuilder, releaseTag string, istioEnabled bool) {
	defaultInstallation.Setup(manifests.InstallFromFile(testNamespaceConfig))

	secretConfig := getSecretConfig(t)
	setup.InstallDynatraceFromGithub(defaultInstallation, &secretConfig, releaseTag)
	setup.AssessOperatorDeployment(defaultInstallation)

	setup.DeploySampleApps(defaultInstallation, sampleDeploymentConfig)

	dynakubeBuilder := dynakube.NewBuilder().
		WithDefaultObjectMeta().
		ApiUrl(secretConfig.ApiUrl).
		CloudNative(&v1beta1.CloudNativeFullStackSpec{})
	if istioEnabled {
		dynakubeBuilder = dynakubeBuilder.WithIstio()
	}
	defaultInstallation.Assess("dynakube applied", dynakube.Apply(dynakubeBuilder.Build()))

	setup.AssessDynakubeStartup(defaultInstallation)
	assessSampleAppsRestart(defaultInstallation)
	assessOneAgentsAreRunning(defaultInstallation)
}
