// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

//go:build e2e

package upgrade

import (
	"fmt"
	"testing"

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
	"sigs.k8s.io/e2e-framework/pkg/features"
)

const withCSI = true

func Feature(t *testing.T, releaseTag string) features.Feature {
	builder := features.New("dk-cloudnative-and-ec-upgrade-operator")

	builder.Assess("install operator "+releaseTag, helpers.ToFeatureFunc(operator.Install(releaseTag, withCSI), true))
	secretConfig := tenant.GetSingleTenantSecret(t)
	testDynakube := *dynakube.New(
		dynakube.WithAPIURL(secretConfig.APIURL),
		dynakube.WithCloudNativeSpec(cloudnative.DefaultCloudNativeSpec()),
	)
	testECname := uuid.NewString()
	testHostPattern := fmt.Sprintf("%s.e2eTestHostPattern.internal.org", testECname)
	edgeConnectTenantConfig := &edgeconnectComponents.TenantConfig{}
	edgeconnectSecretConfig := tenant.GetEdgeConnectTenantSecret(t)
	builder.Assess("create EC configuration on the tenant", edgeconnectComponents.CreateTenantConfig(testECname, edgeconnectSecretConfig, edgeConnectTenantConfig, testHostPattern))

	testEdgeConnect := *edgeconnectComponents.New(
		edgeconnectComponents.WithName(testECname),
		edgeconnectComponents.WithAPIServer(edgeconnectSecretConfig.APIServer),
		edgeconnectComponents.WithOAuthClientSecret(edgeconnectComponents.BuildOAuthClientSecretName(testECname)),
		edgeconnectComponents.WithOAuthEndpoint("https://sso-dev.dynatracelabs.com/sso/oauth2/token"),
		edgeconnectComponents.WithOAuthResource(fmt.Sprintf("urn:dtenvironment:%s", edgeconnectSecretConfig.TenantUID)),
	)

	// create OAuth client secret related to the specific EdgeConnect configuration on the tenant
	builder.Assess("create client secret", tenant.CreateClientSecret(&edgeConnectTenantConfig.Secret, edgeconnectComponents.BuildOAuthClientSecretName(testEdgeConnect.Name), testEdgeConnect.Namespace))

	// install EC
	edgeconnectComponents.Install(builder, nil, testEdgeConnect)
	builder.Assess("check EC configuration on the tenant", edgeconnectComponents.CheckECExistsOnTheTenant(edgeconnectSecretConfig, edgeConnectTenantConfig))

	sampleNamespace := *k8snamespace.New("upgrade-sample")
	sampleApp := sample.NewApp(t, &testDynakube,
		sample.AsDeployment(),
		sample.WithNamespace(sampleNamespace),
	)
	builder.Assess("create sample namespace", sampleApp.InstallNamespace())

	previousVersionDynakube := &dynakubev1beta5.DynaKube{}
	require.NoError(t, previousVersionDynakube.ConvertFrom(&testDynakube))
	dynakube.InstallPreviousVersion(builder, helpers.LevelAssess, &secretConfig, *previousVersionDynakube)

	// Register sample app install
	builder.Assess("install sample app", sampleApp.Install())

	// update to snapshot
	builder.Assess("upgrade operator", helpers.ToFeatureFunc(operator.InstallLocal(withCSI), true))

	// Guarantees the operator reconciles after upgrade and before restarting the app
	dynakube.TriggerReconciliation(builder, testDynakube)

	builder.Assess("restart half of sample apps", sampleApp.Restart())
	cloudnative.AssessSampleInitContainers(builder, sampleApp)

	builder.Teardown(sampleApp.Uninstall())

	builder.Teardown(tenant.DeleteTenantSecret(edgeconnectComponents.BuildOAuthClientSecretName(testEdgeConnect.Name), testEdgeConnect.Namespace))
	builder.Teardown(edgeconnectComponents.DeleteTenantConfig(edgeconnectSecretConfig, edgeConnectTenantConfig))

	return builder.Feature()
}
