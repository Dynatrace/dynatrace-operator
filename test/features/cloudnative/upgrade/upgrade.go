//go:build e2e

package upgrade

import (
	"testing"

	dynakubev1beta4 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta4/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/features/cloudnative"
	"github.com/Dynatrace/dynatrace-operator/test/helpers"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/operator"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/namespace"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/sample"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/tenant"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

func Feature(t *testing.T) features.Feature {
	builder := features.New("cloudnative-upgrade")
	secretConfig := tenant.GetSingleTenantSecret(t)
	testDynakube := *dynakube.New(
		dynakube.WithAPIURL(secretConfig.APIURL),
		dynakube.WithCloudNativeSpec(cloudnative.DefaultCloudNativeSpec()),
	)

	sampleNamespace := *namespace.New("upgrade-sample")
	sampleApp := sample.NewApp(t, &testDynakube,
		sample.AsDeployment(),
		sample.WithNamespace(sampleNamespace),
	)
	builder.Assess("create sample namespace", sampleApp.InstallNamespace())

	previousVersionDynakube := &dynakubev1beta4.DynaKube{}
	previousVersionDynakube.ConvertFrom(&testDynakube)
	dynakube.InstallPreviousVersion(builder, helpers.LevelAssess, &secretConfig, *previousVersionDynakube)

	// Register sample app install
	builder.Assess("install sample app", sampleApp.Install())

	// update to snapshot
	withCSI := true
	builder.Assess("upgrade operator", helpers.ToFeatureFunc(operator.InstallViaMake(withCSI), true))
	builder.Assess("restart half of sample apps", sampleApp.Restart())
	cloudnative.AssessSampleInitContainers(builder, sampleApp)

	builder.Teardown(sampleApp.Uninstall())
	dynakube.Delete(builder, helpers.LevelTeardown, testDynakube)

	return builder.Feature()
}
