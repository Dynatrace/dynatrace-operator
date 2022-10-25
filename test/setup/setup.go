package setup

import (
	"github.com/Dynatrace/dynatrace-operator/test/csi"
	"github.com/Dynatrace/dynatrace-operator/test/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/kubeobjects/manifests"
	"github.com/Dynatrace/dynatrace-operator/test/oneagent"
	"github.com/Dynatrace/dynatrace-operator/test/operator"
	"github.com/Dynatrace/dynatrace-operator/test/secrets"
	"github.com/Dynatrace/dynatrace-operator/test/webhook"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

func InstallAndDeploy(builder *features.FeatureBuilder, secretConfig secrets.Secret, deploymentPath string) {
	InstallOperator(builder, secretConfig)
	DeployApplication(builder, deploymentPath)
}

func InstallOperator(builder *features.FeatureBuilder, secretConfig secrets.Secret) {
	builder.Setup(secrets.ApplyDefault(secretConfig))
	builder.Setup(operator.InstallAllForKubernetes())
}

func DeployApplication(builder *features.FeatureBuilder, deploymentPath string) {
	builder.Setup(manifests.InstallFromFile(deploymentPath))
}

func AssessOperatorDeployment(builder *features.FeatureBuilder) {
	builder.Assess("operator started", operator.WaitForDeployment())
	builder.Assess("webhook started", webhook.WaitForDeployment())
	builder.Assess("csi driver started", csi.WaitForDaemonset())
}

func AssessDynakubeStartup(builder *features.FeatureBuilder) {
	builder.Assess("oneagent started", oneagent.WaitForDaemonset())
	builder.Assess("dynakube phase changes to 'Running'", dynakube.WaitForDynakubePhase(dynakube.NewBuilder().WithDefaultObjectMeta().Build()))
}
