//go:build e2e

package classic

import (
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/test/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/oneagent"
	"github.com/Dynatrace/dynatrace-operator/test/setup"
	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/e2e-framework/pkg/features"
	"testing"
)

func install(t *testing.T) features.Feature {
	installClassicFullStack := features.New("install classic fullstack")
	secretConfig := getSecretConfig(t)

	setup.InstallAndDeploy(installClassicFullStack, secretConfig, "../testdata/classic-fullstack/sample-deployment.yaml")
	setup.AssessDeployment(installClassicFullStack)

	installClassicFullStack.Assess("install dynakube", dynakube.ApplyClassicFullStack(secretConfig.ApiUrl, &dynatracev1beta1.HostInjectSpec{
		Env: []v1.EnvVar{
			{
				Name:  "ONEAGENT_ENABLE_VOLUME_STORAGE",
				Value: "true",
			},
		},
	}))

	setup.AssessDynakubeStartup(installClassicFullStack)

	installClassicFullStack.Assess("os agent can connect", oneagent.OSAgentCanConnect())

	return installClassicFullStack.Feature()
}
