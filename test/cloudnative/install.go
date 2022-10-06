//go:build e2e

package cloudnative

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/test/dynakube"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

const (
	oneAgentInstallContainerName = "install-oneagent"
)

func install(t *testing.T) features.Feature {
	secretConfig := dynakube.GetSecretConfig(t)

	defaultInstallation := features.New("default installation")

	installAndDeploy(defaultInstallation, secretConfig, "../testdata/cloudnative/sample-deployment.yaml")
	assessDeployment(defaultInstallation)

	defaultInstallation.Assess("dynakube applied", dynakube.ApplyDynakube(secretConfig.ApiUrl, &v1beta1.CloudNativeFullStackSpec{}, nil))

	assessDynakubeStartup(defaultInstallation)
	assessOneAgentsAreRunning(defaultInstallation)

	return defaultInstallation.Feature()
}
