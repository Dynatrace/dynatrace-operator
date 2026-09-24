// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

//go:build e2e

package upgrade

import (
	"context"
	"fmt"
	"strings"
	"testing"

	dynakubelatest "github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	dynakubev1beta5 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta5/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/features/cloudnative"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/components/dynakube"
	edgeconnectComponents "github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/components/edgeconnect"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/components/operator"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/kubernetes/objects/k8snamespace"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/sample"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/tenant"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/e2e-framework/pkg/env"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

const withCSI = true

// sanitizeReleaseTag makes a release tag (e.g. "1.10.2") safe to use inside a Kubernetes object name,
// which must be a valid RFC 1123 label and therefore cannot contain dots.
func sanitizeReleaseTag(releaseTag string) string {
	return strings.ReplaceAll(releaseTag, ".", "-")
}

type upgradeOptions struct {
	featureName      string
	sampleNamespace  string
	installOld       env.Func
	installNew       env.Func
	teardownOperator func(b *features.FeatureBuilder, dk *dynakubelatest.DynaKube)
}

func Feature(t *testing.T, releaseTag string) features.Feature {
	return buildUpgradeFeature(t, releaseTag, upgradeOptions{
		featureName:     "dk-upgrade-operator-via-helm",
		sampleNamespace: "helm-upgrade-sample-" + sanitizeReleaseTag(releaseTag),
		installOld:      operator.Install(releaseTag, withCSI),
		installNew:      operator.InstallLocal(withCSI),
		teardownOperator: func(b *features.FeatureBuilder, _ *dynakubelatest.DynaKube) {
			b.WithTeardown("uninstall operator",
				helpers.ToFeatureFunc(func(ctx context.Context, c *envconf.Config) (context.Context, error) {
					// If we cleaned up during a fail-fast (aka.: /debug) it wouldn't be possible to investigate the error.
					if c.FailFast() {
						return ctx, nil
					}

					return operator.Uninstall(withCSI)(ctx, c)
				}, false))
		},
	})
}

// ManifestFeature builds an upgrade scenario
// that installs a released operator version via raw kubectl apply, then upgrades to the current build.
func ManifestFeature(t *testing.T, releaseTag string) features.Feature {
	return buildUpgradeFeature(t, releaseTag, upgradeOptions{
		featureName:     "dk-upgrade-operator-via-manifest",
		sampleNamespace: "manifest-upgrade-sample-" + sanitizeReleaseTag(releaseTag),
		installOld:      operator.InstallReleasedManifest(releaseTag, withCSI),
		installNew:      operator.InstallLocalViaManifests(withCSI),
		teardownOperator: func(b *features.FeatureBuilder, dk *dynakubelatest.DynaKube) {
			dynakube.Delete(b, features.LevelTeardown, dk)
			b.WithTeardown("delete tenant secret",
				func(ctx context.Context, t *testing.T, c *envconf.Config) context.Context {
					// If we cleaned up during a fail-fast (aka.: /debug) it wouldn't be possible to investigate the error.
					if c.FailFast() {
						return ctx
					}

					return tenant.DeleteTenantSecret(dk.Name, dk.Namespace)(ctx, t, c)
				})
			b.WithTeardown("uninstall operator via manifests",
				helpers.ToFeatureFunc(func(ctx context.Context, c *envconf.Config) (context.Context, error) {
					// If we cleaned up during a fail-fast (aka.: /debug) it wouldn't be possible to investigate the error.
					if c.FailFast() {
						return ctx, nil
					}

					return operator.UninstallCurrentManifests(withCSI)(ctx, c)
				}, false))
		},
	})
}

func buildUpgradeFeature(t *testing.T, releaseTag string, opts upgradeOptions) features.Feature {
	builder := features.New(opts.featureName)

	builder.Assess("install operator "+releaseTag, helpers.ToFeatureFunc(opts.installOld, true))

	secretConfig := tenant.GetSingleTenantSecret(t)
	testDynakube := dynakube.New(
		dynakube.WithName("dynakube-"+sanitizeReleaseTag(releaseTag)),
		dynakube.WithAPIURL(secretConfig.APIURL),
		dynakube.WithCloudNativeSpec(cloudnative.DefaultCloudNativeSpec()),
	)

	testECname := uuid.NewString()
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
	builder.Assess("create client secret", func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		return tenant.CreateClientSecret(
			edgeConnectTenantConfig.Secret,
			edgeconnectComponents.BuildOAuthClientSecretName(testEdgeConnect.Name),
			testEdgeConnect.Namespace,
		)(ctx, t, envConfig)
	})

	// install EC
	edgeconnectComponents.Install(builder, tenant.EdgeConnectSecret{}, testEdgeConnect)
	builder.Assess("check EC configuration on the tenant", edgeconnectComponents.CheckECExistsOnTheTenant(edgeconnectSecretConfig, edgeConnectTenantConfig))

	// Register sample app install
	sampleNamespace := k8snamespace.New(opts.sampleNamespace)
	sampleApp := sample.NewApp(t, testDynakube,
		sample.AsDeployment(),
		sample.WithNamespace(sampleNamespace),
	)

	previousVersionDynakube := &dynakubev1beta5.DynaKube{}
	require.NoError(t, previousVersionDynakube.ConvertFrom(testDynakube))
	dynakube.InstallPreviousVersion(builder, helpers.LevelAssess, secretConfig, previousVersionDynakube)

	builder.Assess("create sample namespace", sampleApp.InstallNamespace())
	builder.Assess("install sample app", sampleApp.Install())

	// update to snapshot (helm) or latest manifest
	builder.Assess("upgrade operator", helpers.ToFeatureFunc(opts.installNew, true))

	// Guarantees the operator reconciles after upgrade and before restarting the app
	dynakube.TriggerReconciliation(builder, testDynakube)

	builder.Assess("restart sample app", sampleApp.Restart())
	cloudnative.AssessSampleInitContainers(builder, sampleApp)

	builder.Teardown(sampleApp.Uninstall())
	builder.WithTeardown("delete EC tenant config",
		edgeconnectComponents.DeleteTenantConfig(edgeconnectSecretConfig, edgeConnectTenantConfig))

	opts.teardownOperator(builder, testDynakube)

	return builder.Feature()
}
