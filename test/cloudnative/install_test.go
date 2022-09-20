//go:build e2e

package cloudnative

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

const (
	oneAgentInstallContainerName = "install-oneagent"

	installSecretsPath = "../testdata/secrets/cloudnative-install.yaml"
)

func install(t *testing.T) features.Feature {
	secretConfig := getSecretConfig(t)

	defaultInstallation := features.New("default installation")

	installAndDeploy(defaultInstallation, secretConfig, "../testdata/cloudnative/sample-deployment.yaml")
	assessDeployment(defaultInstallation)

	defaultInstallation.Assess("dynakube applied", applyDynakube(secretConfig.ApiUrl, &v1beta1.CloudNativeFullStackSpec{}))

	assessDynakubeStartup(defaultInstallation)
	assessOneAgentsAreRunning(defaultInstallation)

	return defaultInstallation.Feature()
}
