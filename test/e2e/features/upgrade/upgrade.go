// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

//go:build e2e

package upgrade

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"uuid"

	dynakubev1beta5 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta5/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/features/cloudnative"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/components/dynakube"
	edgeconnectComponents "github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/components/edgeconnect"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/components/operator"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/kubernetes/objects/k8snamespace"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/sample"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/tenant"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/e2e-framework/pkg/env"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

const withCSI = true

// sanitizeReleaseTag makes a release tag (e.g. "1.10.2") safe to use inside a Kubernetes object name,
// which must be a valid RFC 1123 label and therefore cannot contain dots.
func sanitizeReleaseTag(releaseTag string) string {
	return strings.ReplaceAll(releaseTag, ".", "-")
}

// Feature builds an upgrade scenario that installs a released operator version and upgrades to the current build.
func Feature(t *testing.T, releaseTag string) features.Feature {
	viaManifests := os.Getenv("MANIFESTS") == "true"

	featureName := "dk-upgrade-operator-via-helm"
	installOld := operator.Install(releaseTag, withCSI)

	if viaManifests {
		featureName = "dk-upgrade-operator-via-manifest"
		installOld = operator.InstallReleasedManifest(releaseTag, withCSI)
	}

	return buildUpgradeFeature(t, features.New(featureName), releaseTag, installOld)
}

func buildUpgradeFeature(t *testing.T, builder *features.FeatureBuilder, releaseTag string, installOld env.Func) features.Feature {
	builder.Assess("install operator "+releaseTag, helpers.ToFeatureFunc(installOld, true))

	secretConfig := tenant.GetSingleTenantSecret(t)
	testDynakube := dynakube.New(
		dynakube.WithName("dynakube-"+sanitizeReleaseTag(releaseTag)),
		dynakube.WithAPIURL(secretConfig.APIURL),
		dynakube.WithCloudNativeSpec(cloudnative.DefaultCloudNativeSpec()),
	)

	testECname := uuid.New().String()
	testHostPattern := fmt.Sprintf("%s.e2eTestHostPattern.internal.org", testECname)
	edgeConnectTenantConfig := &edgeconnectComponents.TenantConfig{}
	edgeconnectSecretConfig := tenant.GetEdgeConnectTenantSecret(t)
	builder.Assess("create EC configuration on the tenant", edgeconnectComponents.CreateTenantConfig(testECname, edgeconnectSecretConfig, edgeConnectTenantConfig, testHostPattern))

	testEdgeConnect := edgeconnectComponents.New(
		edgeconnectComponents.WithName(testECname),
		edgeconnectComponents.WithAPIServer(edgeconnectSecretConfig.APIServer),
		edgeconnectComponents.WithOAuthClientSecret(edgeconnectComponents.BuildOAuthClientSecretName(testECname)),
		edgeconnectComponents.WithOAuthEndpoint("https://sso-dev.dynatracelabs.com/sso/oauth2/token"),
		edgeconnectComponents.WithOAuthResource(fmt.Sprintf("urn:dtenvironment:%s", edgeconnectSecretConfig.TenantUID)),
	)

	// create OAuth client secret related to the specific EdgeConnect configuration on the tenant
	builder.Assess("create client secret", tenant.CreateClientSecret(&edgeConnectTenantConfig.Secret, edgeconnectComponents.BuildOAuthClientSecretName(testEdgeConnect.Name), testEdgeConnect.Namespace))

	// install EC
	edgeconnectComponents.Install(builder, tenant.EdgeConnectSecret{}, testEdgeConnect)
	builder.Assess("check EC configuration on the tenant", edgeconnectComponents.CheckECExistsOnTheTenant(edgeconnectSecretConfig, edgeConnectTenantConfig))

	// Register sample app install
	sampleNamespace := k8snamespace.New("upgrade-sample-" + sanitizeReleaseTag(releaseTag))
	sampleApp := sample.NewApp(t, testDynakube,
		sample.AsDeployment(),
		sample.WithNamespace(sampleNamespace),
	)

	previousVersionDynakube := &dynakubev1beta5.DynaKube{}
	require.NoError(t, previousVersionDynakube.ConvertFrom(testDynakube))
	dynakube.InstallPreviousVersion(builder, helpers.LevelAssess, secretConfig, previousVersionDynakube)

	builder.Assess("create sample namespace", sampleApp.InstallNamespace())
	builder.Assess("install sample app", sampleApp.Install())

	// update to the current build
	builder.Assess("upgrade operator", helpers.ToFeatureFunc(operator.InstallLocal(withCSI), true))

	// Guarantees the operator reconciles after upgrade and before restarting the app
	dynakube.TriggerReconciliation(builder, testDynakube)

	builder.Assess("restart sample app", sampleApp.Restart())
	cloudnative.AssessSampleInitContainers(builder, sampleApp)

	builder.Teardown(sampleApp.Uninstall())
	builder.WithTeardown("delete EC tenant config",
		edgeconnectComponents.DeleteTenantConfig(edgeconnectSecretConfig, edgeConnectTenantConfig))

	// The DynaKube has to be gone before the operator is removed, otherwise it's stuck on its finalizer.
	dynakube.Cleanup(builder, testDynakube)

	builder.WithTeardown("uninstall operator", helpers.SkipOnFailFast(helpers.ToFeatureFunc(operator.Uninstall(withCSI), false)))

	return builder.Feature()
}
