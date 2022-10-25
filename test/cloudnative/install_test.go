//go:build e2e

package cloudnative

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/test/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/setup"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

const (
	oneAgentInstallContainerName = "install-oneagent"
)

func install(t *testing.T) features.Feature {
	secretConfig := getSecretConfig(t)

	defaultInstallation := features.New("default installation")

	setup.InstallAndDeploy(defaultInstallation, secretConfig, "../testdata/cloudnative/sample-deployment.yaml")
	setup.AssessDeployment(defaultInstallation)

	defaultInstallation.Assess("dynakube applied", dynakube.Apply(
		dynakube.NewBuilder().
			WithDefaultObjectMeta().
			ApiUrl(secretConfig.ApiUrl).
			CloudNative(&v1beta1.CloudNativeFullStackSpec{}).
			Build()))

	setup.AssessDynakubeStartup(defaultInstallation)
	assessOneAgentsAreRunning(defaultInstallation)

	return defaultInstallation.Feature()
}
