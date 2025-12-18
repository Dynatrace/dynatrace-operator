//go:build e2e

package upgrade

import (
	"testing"

	dynakubev1beta5 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta5/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/features/consts"
	"github.com/Dynatrace/dynatrace-operator/test/helpers"
	componentDynakube "github.com/Dynatrace/dynatrace-operator/test/helpers/components/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/operator"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/statefulset"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/tenant"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

func Feature(t *testing.T) features.Feature {
	builder := features.New("deprecated-secret-upgrade-operator")
	secretConfig := tenant.GetSingleTenantSecret(t)

	options := []componentDynakube.Option{
		componentDynakube.WithAPIURL(secretConfig.APIURL),
		componentDynakube.WithExtensionsPrometheusEnabledSpec(true),
		componentDynakube.WithExtensionsEECImageRefSpec(consts.EecImageRepo, consts.EecImageTag),
		componentDynakube.WithActiveGate(),
	}

	testDynakube := *componentDynakube.New(options...)

	previousVersionDynakube := &dynakubev1beta5.DynaKube{}
	_ =previousVersionDynakube.ConvertFrom(&testDynakube)
	componentDynakube.InstallPreviousVersion(builder, helpers.LevelAssess, &secretConfig, *previousVersionDynakube)

	legacyName := testDynakube.Name + "-extensions-controller"

	builder.Assess("extension execution controller started", statefulset.WaitFor(legacyName, testDynakube.Namespace))

	builder.Assess("extension collector started", statefulset.WaitFor(testDynakube.OtelCollectorStatefulsetName(), testDynakube.Namespace))

	// update to snapshot
	withCSI := true
	builder.Assess("upgrade operator", helpers.ToFeatureFunc(operator.InstallLocal(withCSI), true))

	builder.Assess("extension execution controller started after upgrade", statefulset.WaitFor(testDynakube.Extensions().GetExecutionControllerStatefulsetName(), testDynakube.Namespace))

	builder.Assess("extension collector started after upgrade", statefulset.WaitFor(testDynakube.OtelCollectorStatefulsetName(), testDynakube.Namespace))

	builder.Assess("legacy extensions executor controller deleted", statefulset.WaitForDeletion(legacyName, testDynakube.Namespace))

	componentDynakube.Delete(builder, helpers.LevelTeardown, testDynakube)

	return builder.Feature()
}
