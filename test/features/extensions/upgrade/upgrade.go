//go:build e2e

package upgrade

import (
	"github.com/Dynatrace/dynatrace-operator/test/features/consts"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/statefulset"
	"testing"

	dynakubev1beta5 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta5/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/helpers"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/dynakube"
	componentDynakube "github.com/Dynatrace/dynatrace-operator/test/helpers/components/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/operator"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/namespace"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/sample"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/tenant"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

func Feature(t *testing.T) features.Feature {
	builder := features.New("extensions-upgrade")
	secretConfig := tenant.GetSingleTenantSecret(t)

	options := []componentDynakube.Option{
		componentDynakube.WithAPIURL(secretConfig.APIURL),
		componentDynakube.WithExtensionsEnabledSpec(true),
		componentDynakube.WithExtensionsEECImageRefSpec(consts.EecImageRepo, consts.EecImageTag),
		componentDynakube.WithActiveGate(),
		componentDynakube.WithActiveGateTLSSecret(consts.AgSecretName),
	}

	testDynakube := *componentDynakube.New(options...)

	sampleNamespace := *namespace.New("upgrade-extensions-sample")
	sampleApp := sample.NewApp(t, &testDynakube,
		sample.AsDeployment(),
		sample.WithNamespace(sampleNamespace),
	)
	builder.Assess("create sample namespace", sampleApp.InstallNamespace())

	previousVersionDynakube := &dynakubev1beta5.DynaKube{}
	previousVersionDynakube.ConvertFrom(&testDynakube)
	dynakube.InstallPreviousVersion(builder, helpers.LevelAssess, &secretConfig, *previousVersionDynakube)

	// update to snapshot
	withCSI := true
	builder.Assess("upgrade operator", helpers.ToFeatureFunc(operator.InstallViaMake(withCSI), true))

	//	builder.Assess("active gate pod is running", checkActiveGateContainer(&testDynakube))

	builder.Assess("extensions execution controller started", statefulset.WaitFor(testDynakube.Extensions().GetExecutionControllerStatefulsetName(), testDynakube.Namespace))

	builder.Assess("extension collector started", statefulset.WaitFor(testDynakube.OtelCollectorStatefulsetName(), testDynakube.Namespace))

	builder.Teardown(sampleApp.Uninstall())
	dynakube.Delete(builder, helpers.LevelTeardown, testDynakube)

	return builder.Feature()
}
