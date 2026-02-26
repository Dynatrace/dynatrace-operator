//go:build e2e

package upgrade

import (
	"testing"

	dynakubev1beta5 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta5/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers"
	componentDynakube "github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/components/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/components/operator"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/kubernetes/objects/k8sstatefulset"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/tenant"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

func Feature(t *testing.T) features.Feature {
	builder := features.New("deprecated-secret-upgrade-operator")
	secretConfig := tenant.GetSingleTenantSecret(t)

	options := []componentDynakube.Option{
		componentDynakube.WithAPIURL(secretConfig.APIURL),
		componentDynakube.WithExtensionsPrometheusEnabledSpec(true),
		componentDynakube.WithExtensionsEECImageRef(),
		componentDynakube.WithActiveGate(),
	}

	testDynakube := *componentDynakube.New(options...)

	previousVersionDynakube := &dynakubev1beta5.DynaKube{}
	_ = previousVersionDynakube.ConvertFrom(&testDynakube)
	componentDynakube.InstallPreviousVersion(builder, helpers.LevelAssess, &secretConfig, *previousVersionDynakube)

	legacyName := testDynakube.Name + "-extensions-controller"

	builder.Assess("extension execution controller started", k8sstatefulset.IsReady(legacyName, testDynakube.Namespace))

	builder.Assess("extension collector started", k8sstatefulset.IsReady(testDynakube.OtelCollectorStatefulsetName(), testDynakube.Namespace))

	// update to snapshot
	withCSI := true
	builder.Assess("upgrade operator", helpers.ToFeatureFunc(operator.InstallLocal(withCSI), true))

	builder.Assess("extension execution controller started after upgrade", k8sstatefulset.WaitFor(testDynakube.Extensions().GetExecutionControllerStatefulsetName(), testDynakube.Namespace))

	builder.Assess("extension collector started after upgrade", k8sstatefulset.WaitFor(testDynakube.OtelCollectorStatefulsetName(), testDynakube.Namespace))

	builder.Assess("legacy extensions executor controller deleted", k8sstatefulset.WaitForDeletion(legacyName, testDynakube.Namespace))

	componentDynakube.Delete(builder, helpers.LevelTeardown, testDynakube)

	return builder.Feature()
}
