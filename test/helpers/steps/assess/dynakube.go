//go:build e2e

package assess

import (
	"fmt"

	"github.com/Dynatrace/dynatrace-operator/src/api/status"
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/oneagent"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/steps/teardown"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/tenant"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

func InstallDynakube(builder *features.FeatureBuilder, secretConfig *tenant.Secret, testDynakube dynatracev1beta1.DynaKube) {
	CreateDynakube(builder, secretConfig, testDynakube)
	VerifyDynakubeStartup(builder, testDynakube)
}

func InstallDynakubeWithTeardown(builder *features.FeatureBuilder, secretConfig *tenant.Secret, testDynakube dynatracev1beta1.DynaKube) {
	CreateDynakube(builder, secretConfig, testDynakube)
	VerifyDynakubeStartup(builder, testDynakube)
	teardown.DeleteDynakube(builder, testDynakube)
}

func CreateDynakube(builder *features.FeatureBuilder, secretConfig *tenant.Secret, testDynakube dynatracev1beta1.DynaKube) {
	if secretConfig != nil {
		builder.Assess("created tenant secret", tenant.CreateTenantSecret(*secretConfig, testDynakube.Name, testDynakube.Namespace))
	}
	builder.Assess(
		fmt.Sprintf("'%s' dynakube created", testDynakube.Name),
		dynakube.Create(testDynakube))
}

func UpdateDynakube(builder *features.FeatureBuilder, testDynakube dynatracev1beta1.DynaKube) {
	builder.Assess("dynakube updated", dynakube.Update(testDynakube))
}

func DeleteDynakube(builder *features.FeatureBuilder, testDynakube dynatracev1beta1.DynaKube) {
	builder.Assess("dynakube deleted", dynakube.Delete(testDynakube))
	if testDynakube.NeedsOneAgent() {
		builder.Assess("oneagent pods stopped", oneagent.WaitForDaemonSetPodsDeletion(testDynakube))
	}
}

func VerifyDynakubeStartup(builder *features.FeatureBuilder, testDynakube dynatracev1beta1.DynaKube) {
	if testDynakube.NeedsOneAgent() {
		builder.Assess("oneagent started", oneagent.WaitForDaemonset(testDynakube))
	}
	builder.Assess(
		fmt.Sprintf("'%s' dynakube phase changes to 'Running'", testDynakube.Name),
		dynakube.WaitForDynakubePhase(testDynakube, status.Running))
}
