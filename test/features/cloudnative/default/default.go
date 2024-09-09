//go:build e2e

package _default

import (
	"fmt"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/test/features/cloudnative"
	"github.com/Dynatrace/dynatrace-operator/test/helpers"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/codemodules"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/istio"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/namespace"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/sample"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/tenant"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

// # With istio enabled
//
// Prerequisites: istio service mesh
//
// Setup: CloudNative deployment with CSI driver
//
// Verify that the operator is working as expected when istio service mesh is
// installed and pre-configured on the cluster.
//
// # Install
//
// Verification that OneAgent is injected to sample application pods and can
// communicate with the *Dynatrace Cluster*.
//
// # Upgrade
//
// Verification that a *released version* can be updated to the *current version*.
// The exact number of *released version* is updated manually. The *released
// version* is installed using configuration files from GitHub.
//
// Sample application Deployment is installed and restarted to check if OneAgent is
// injected and can communicate with the *Dynatrace Cluster*.
//
// # Specific Agent Version)
//
// Verification that the operator is able to set agent version which is given via
// the dynakube. Upgrading to a newer version of agent is also tested.
//
// Sample application Deployment is installed and restarted to check if OneAgent is
// injected and VERSION environment variable is correct.
func Feature(t *testing.T, istioEnabled bool) features.Feature {
	builder := features.New("cloudnative")
	t.Logf("istio enabled: %v", istioEnabled)
	secretConfig := tenant.GetSingleTenantSecret(t)

	options := []dynakube.Option{
		dynakube.WithApiUrl(secretConfig.ApiUrl),
		dynakube.WithCloudNativeSpec(cloudnative.DefaultCloudNativeSpec()),
	}
	if istioEnabled {
		options = append(options, dynakube.WithIstioIntegration())
	}
	testDynakube := *dynakube.New(options...)
	// Register sample app install
	namespaceOptions := []namespace.Option{}
	if istioEnabled {
		namespaceOptions = append(namespaceOptions, namespace.WithIstio())
	}
	sampleNamespace := *namespace.New("cloudnative-sample", namespaceOptions...)
	sampleApp := sample.NewApp(t, &testDynakube, sample.WithNamespace(sampleNamespace), sample.AsDeployment())

	builder.Assess("create sample namespace", sampleApp.InstallNamespace())

	// Register dynakube install
	dynakube.Install(builder, helpers.LevelAssess, &secretConfig, testDynakube)
	builder.Assess("install sample app", sampleApp.Install())

	// Register actual test
	cloudnative.AssessSampleInitContainers(builder, sampleApp)
	if istioEnabled {
		istio.AssessIstio(builder, testDynakube, *sampleApp)
	}

	builder.Assess(fmt.Sprintf("check %s has no conn info", codemodules.RuxitAgentProcFile), codemodules.CheckRuxitAgentProcFileHasNoConnInfo(testDynakube))

	// Register sample, dynakube and operator uninstall
	builder.Teardown(sampleApp.Uninstall())
	dynakube.Delete(builder, helpers.LevelTeardown, testDynakube)

	builder.WithTeardown("deleted tenant secret", tenant.DeleteTenantSecret(testDynakube.Name, testDynakube.Namespace))

	return builder.Feature()
}
