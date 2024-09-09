//go:build e2e

package publicregistry

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/test/features/cloudnative"
	"github.com/Dynatrace/dynatrace-operator/test/helpers"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/activegate"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/namespace"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/registry"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/sample"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/tenant"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

// Feature defines the e2e test to verify that public-registry images can be deployed by the operator and that they function
// This includes:
//   - ActiveGate StatefulSet gets ready
//   - CodeModules can be downloaded and mounted
//   - OneAgent DaemonSet gets ready
//
// It determines the latest version of each image using the registry.
func Feature(t *testing.T) features.Feature {
	builder := features.New("public-registry-images")
	secretConfig := tenant.GetSingleTenantSecret(t)

	oaSpec := cloudnative.DefaultCloudNativeSpec()
	oaSpec.Image = registry.GetLatestOneAgentImageURI(t)
	oaSpec.CodeModulesImage = registry.GetLatestCodeModulesImageURI(t)

	options := []dynakube.Option{
		dynakube.WithApiUrl(secretConfig.ApiUrl),
		dynakube.WithCloudNativeSpec(oaSpec),
		dynakube.WithActiveGate(),
		dynakube.WithCustomActiveGateImage(registry.GetLatestActiveGateImageURI(t)),
	}
	testDynakube := *dynakube.New(options...)

	// Register sample app install
	sampleNamespace := *namespace.New("public-registry-sample")
	sampleApp := sample.NewApp(t, &testDynakube, sample.WithNamespace(sampleNamespace), sample.AsDeployment())

	builder.Assess("create sample namespace", sampleApp.InstallNamespace())

	// Register dynakube install - will verify OneAgent DaemonSet startup
	dynakube.Install(builder, helpers.LevelAssess, &secretConfig, testDynakube)

	// Install Sample apps - will check if CodeModule could be downloaded and mounted
	builder.Assess("install sample app", sampleApp.Install())
	cloudnative.AssessSampleInitContainers(builder, sampleApp)

	// Check if the ActiveGate could start up
	builder.Assess("ActiveGate started", activegate.WaitForStatefulSet(&testDynakube, "activegate"))

	// Register sample, dynakube and operator uninstall
	builder.Teardown(sampleApp.Uninstall())
	dynakube.Delete(builder, helpers.LevelTeardown, testDynakube)

	return builder.Feature()
}
