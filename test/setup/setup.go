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

func DeploySampleApps(builder *features.FeatureBuilder, deploymentPath string) {
	builder.Setup(manifests.InstallFromFile(deploymentPath))
}

func InstallDynatraceFromSource(builder *features.FeatureBuilder, secretConfig *secrets.Secret) {
	if secretConfig != nil {
		builder.Setup(secrets.ApplyDefault(*secretConfig))
	}
	builder.Setup(operator.InstallFromSource(true))
}

func InstallDynatraceFromGithub(builder *features.FeatureBuilder, secretConfig *secrets.Secret, releaseTag string) {
	if secretConfig != nil {
		builder.Setup(secrets.ApplyDefault(*secretConfig))
	}
	builder.Setup(operator.InstallFromGithub(releaseTag, true))
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
