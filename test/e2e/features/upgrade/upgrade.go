// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

//go:build e2e

package upgrade

import (
	"context"
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
	rbacv1 "k8s.io/api/rbac/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/e2e-framework/klient/k8s"
	"sigs.k8s.io/e2e-framework/pkg/env"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

const (
	withCSI                  = true
	extensionsPrometheusName = "dynatrace-extensions-prometheus"
)

// sanitizeReleaseTag makes a release tag (e.g. "1.10.2") safe to use inside a Kubernetes object name,
// which must be a valid RFC 1123 label and therefore cannot contain dots.
func sanitizeReleaseTag(releaseTag string) string {
	return strings.ReplaceAll(releaseTag, ".", "-")
}

// cleanupOrphanedManifestResources removes cluster-scoped resources left over by a previous manifest installation.
func cleanupOrphanedManifestResources() features.Func {
	return func(ctx context.Context, t *testing.T, c *envconf.Config) context.Context {
		objects := []k8s.Object{
			&rbacv1.ClusterRole{ObjectMeta: metav1.ObjectMeta{Name: extensionsPrometheusName}},
			&rbacv1.ClusterRoleBinding{ObjectMeta: metav1.ObjectMeta{Name: extensionsPrometheusName}},
		}

		for _, obj := range objects {
			err := c.Client().Resources().Delete(ctx, obj)
			if !k8serrors.IsNotFound(err) {
				require.NoError(t, err)
			}
		}

		return ctx
	}
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

	return buildUpgradeFeature(t, features.New(featureName), releaseTag, installOld, viaManifests)
}

func buildUpgradeFeature(t *testing.T, builder *features.FeatureBuilder, releaseTag string, installOld env.Func, viaManifests bool) features.Feature {
	// if manifest upgrade test was run before - the remaining resources can fail helm install
	builder.Assess("cleanup orphaned manifest resources", cleanupOrphanedManifestResources())
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

	if viaManifests {
		builder.WithTeardown("cleanup old manifest resources", cleanupOrphanedManifestResources())
	}

	return builder.Feature()
}
