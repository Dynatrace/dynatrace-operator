//go:build e2e

package edgeconnect

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha2/edgeconnect"
	edgeconnectClient "github.com/Dynatrace/dynatrace-operator/pkg/clients/edgeconnect"
	controller "github.com/Dynatrace/dynatrace-operator/pkg/controllers/edgeconnect"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers"
	ecComponents "github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/components/edgeconnect"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/istio"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/kubeobjects/configmap"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/kubeobjects/manifests"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/proxy"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/tenant"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/project"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/exp/slices"
	"sigs.k8s.io/e2e-framework/klient/k8s"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/klient/wait/conditions"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

const caConfigMapName = "proxy-ca"

func NormalModeFeature(t *testing.T) features.Feature {
	builder := features.New("edgeconnect-install")

	secretConfig := tenant.GetEdgeConnectTenantSecret(t)

	edgeConnectTenantConfig := &ecComponents.TenantConfig{}

	testECname := uuid.NewString()
	testHostPattern := fmt.Sprintf("%s.e2eTestHostPattern.internal.org", testECname)

	builder.Assess("create EC configuration on the tenant", ecComponents.CreateTenantConfig(testECname, secretConfig, edgeConnectTenantConfig, testHostPattern))

	testEdgeConnect := *ecComponents.New(
		// this tenantConfigName should match with tenant edgeConnect tenantConfigName
		ecComponents.WithName(testECname),
		ecComponents.WithAPIServer(secretConfig.APIServer),
		ecComponents.WithOAuthClientSecret(ecComponents.BuildOAuthClientSecretName(testECname)),
		ecComponents.WithOAuthEndpoint("https://sso-dev.dynatracelabs.com/sso/oauth2/token"),
		ecComponents.WithOAuthResource(fmt.Sprintf("urn:dtenvironment:%s", secretConfig.TenantUID)),
	)

	// create OAuth client secret related to the specific EdgeConnect configuration on the tenant
	builder.Assess("create client secret", tenant.CreateClientSecret(&edgeConnectTenantConfig.Secret, ecComponents.BuildOAuthClientSecretName(testEdgeConnect.Name), testEdgeConnect.Namespace))

	ecComponents.Install(builder, helpers.LevelAssess, nil, testEdgeConnect)

	builder.Assess("check EC configuration on the tenant", ecComponents.CheckEcExistsOnTheTenant(secretConfig, edgeConnectTenantConfig))
	builder.Assess("delete EdgeConnect CR", ecComponents.Delete(testEdgeConnect))
	builder.Assess("check if EC configuration is deleted on the tenant", ecComponents.CheckEcExistsOnTheTenant(secretConfig, edgeConnectTenantConfig))

	builder.Teardown(tenant.DeleteTenantSecret(ecComponents.BuildOAuthClientSecretName(testEdgeConnect.Name), testEdgeConnect.Namespace))
	builder.Teardown(ecComponents.DeleteTenantConfig(secretConfig, edgeConnectTenantConfig))

	return builder.Feature()
}

func ProvisionerModeFeature(t *testing.T) features.Feature {
	builder := features.New("edgeconnect-install-provisioner")

	secretConfig := tenant.GetEdgeConnectTenantSecret(t)

	edgeConnectTenantConfig := &ecComponents.TenantConfig{}

	testECname := uuid.NewString()
	testHostPattern := fmt.Sprintf("%s.e2eTestHostPattern.internal.org", testECname)
	testHostPattern2 := fmt.Sprintf("%s.e2eTestHostPattern2.internal.org", testECname)

	testEdgeConnect := *ecComponents.New(
		ecComponents.WithName(testECname),
		ecComponents.WithAPIServer(secretConfig.APIServer),
		ecComponents.WithOAuthClientSecret(ecComponents.BuildOAuthClientSecretName(testECname)),
		ecComponents.WithOAuthEndpoint("https://sso-dev.dynatracelabs.com/sso/oauth2/token"),
		ecComponents.WithOAuthResource(secretConfig.Resource),
		ecComponents.WithProvisionerMode(true),
		ecComponents.WithHostPattern(testHostPattern),
	)

	ecComponents.Install(builder, helpers.LevelAssess, &secretConfig, testEdgeConnect)

	builder.Assess("get tenant config", getTenantConfig(testECname, secretConfig, edgeConnectTenantConfig))
	builder.Assess("get EC status", ecComponents.Get(&testEdgeConnect))

	builder.Assess("check if EC configuration exists on the tenant", ecComponents.CheckEcExistsOnTheTenant(secretConfig, edgeConnectTenantConfig))
	builder.Assess("check hostPatterns on the tenant - testHostPattern", checkHostPatternOnTheTenant(secretConfig, edgeConnectTenantConfig, func() string { return testHostPattern }))
	builder.Assess("update hostPatterns", updateHostPatterns(&testEdgeConnect, testHostPattern2))
	builder.Assess("check hostPatterns on the tenant - testHostPattern2", checkHostPatternOnTheTenant(secretConfig, edgeConnectTenantConfig, func() string { return testHostPattern2 }))
	builder.Assess("delete EC custom resource", ecComponents.Delete(testEdgeConnect))
	builder.Assess("check if EC configuration is deleted on the tenant", checkEcNotExistsOnTheTenant(secretConfig, edgeConnectTenantConfig))

	builder.Teardown(tenant.DeleteTenantSecret(ecComponents.BuildOAuthClientSecretName(testEdgeConnect.Name), testEdgeConnect.Namespace))

	return builder.Feature()
}

func WithHTTPProxy(t *testing.T) features.Feature {
	builder := features.New("edgeconnect-install-http-proxy")

	builder.Setup(func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		ctx, err := istio.AssertIstioNamespace()(ctx, envConfig, t)
		require.NoError(t, err)

		return ctx
	})
	builder.Setup(func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		ctx, err := istio.AssertIstiodDeployment()(ctx, envConfig, t)
		require.NoError(t, err)

		return ctx
	})

	secretConfig := tenant.GetEdgeConnectTenantSecret(t)

	edgeConnectTenantConfig := &ecComponents.TenantConfig{}

	testECname := uuid.NewString()
	testHostPattern := fmt.Sprintf("%s.e2eTestHostPattern.internal.org", testECname)

	testEdgeConnect := *ecComponents.New(
		ecComponents.WithName(testECname),
		ecComponents.WithAPIServer(secretConfig.APIServer),
		ecComponents.WithOAuthClientSecret(ecComponents.BuildOAuthClientSecretName(testECname)),
		ecComponents.WithOAuthEndpoint("https://sso-dev.dynatracelabs.com/sso/oauth2/token"),
		ecComponents.WithOAuthResource(secretConfig.Resource),
		ecComponents.WithProvisionerMode(true),
		ecComponents.WithHostPattern(testHostPattern),
		ecComponents.WithProxy(proxy.EdgeConnectProxySpec),
	)

	dummyDynakube := dynakube.DynaKube{}
	dummyDynakube.Namespace = testEdgeConnect.Namespace
	dummyDynakube.Spec.Proxy = proxy.ProxySpec

	proxy.SetupProxyWithTeardown(t, builder, dummyDynakube)
	proxy.CutOffDynatraceNamespace(builder, proxy.ProxySpec)
	proxy.IsDynatraceNamespaceCutOff(builder, dummyDynakube)

	ecComponents.Install(builder, helpers.LevelAssess, &secretConfig, testEdgeConnect)

	builder.Assess("get tenant config", getTenantConfig(testECname, secretConfig, edgeConnectTenantConfig))
	builder.Assess("get EC status", ecComponents.Get(&testEdgeConnect))
	builder.Assess("check if EC configuration exists on the tenant", ecComponents.CheckEcExistsOnTheTenant(secretConfig, edgeConnectTenantConfig))
	builder.Assess("delete EC custom resource", ecComponents.Delete(testEdgeConnect))
	builder.Assess("check if EC configuration is deleted on the tenant", checkEcNotExistsOnTheTenant(secretConfig, edgeConnectTenantConfig))

	builder.Teardown(tenant.DeleteTenantSecret(ecComponents.BuildOAuthClientSecretName(testEdgeConnect.Name), testEdgeConnect.Namespace))

	return builder.Feature()
}

func WithHTTPSProxy(t *testing.T) features.Feature {
	builder := features.New("edgeconnect-install-https-proxy")

	builder.Setup(func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		ctx, err := istio.AssertIstioNamespace()(ctx, envConfig, t)
		require.NoError(t, err)

		return ctx
	})
	builder.Setup(func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		ctx, err := istio.AssertIstiodDeployment()(ctx, envConfig, t)
		require.NoError(t, err)

		return ctx
	})

	secretConfig := tenant.GetEdgeConnectTenantSecret(t)

	edgeConnectTenantConfig := &ecComponents.TenantConfig{}

	testECname := uuid.NewString()
	testHostPattern := fmt.Sprintf("%s.e2eTestHostPattern.internal.org", testECname)

	testEdgeConnect := *ecComponents.New(
		ecComponents.WithName(testECname),
		ecComponents.WithAPIServer(secretConfig.APIServer),
		ecComponents.WithOAuthClientSecret(ecComponents.BuildOAuthClientSecretName(testECname)),
		ecComponents.WithOAuthEndpoint("https://sso-dev.dynatracelabs.com/sso/oauth2/token"),
		ecComponents.WithOAuthResource(secretConfig.Resource),
		ecComponents.WithProvisionerMode(true),
		ecComponents.WithHostPattern(testHostPattern),
		// When using proxy spec with squid in HTTPS mode, the EdgeConnect HTTP client fails to connect.
		// This might be fixed in the future, but for now using the HTTPS_PROXY circumvents the issue.
		ecComponents.WithEnvValue("HTTPS_PROXY", proxy.HTTPSProxySpec.Value),
		ecComponents.WithCACert(caConfigMapName),
	)

	proxyCert, proxyPk, err := proxy.CreateProxyTLSCertAndKey()
	require.NoError(t, err, "failed to create proxy TLS secret")

	// Add customCA config map
	caConfigMap := configmap.New(caConfigMapName, testEdgeConnect.Namespace,
		map[string]string{dynakube.TrustedCAKey: string(proxyCert)})
	builder.Assess("create trusted CAs config map", configmap.Create(caConfigMap))
	builder.Teardown(configmap.Delete(caConfigMap))

	dummyDynakube := dynakube.DynaKube{}
	dummyDynakube.Namespace = testEdgeConnect.Namespace
	dummyDynakube.Spec.Proxy = proxy.HTTPSProxySpec

	proxy.SetupProxyWithCustomCAandTeardown(t, builder, dummyDynakube, proxyCert, proxyPk)
	proxy.CutOffDynatraceNamespace(builder, proxy.HTTPSProxySpec)
	proxy.IsDynatraceNamespaceCutOff(builder, dummyDynakube)

	ecComponents.Install(builder, helpers.LevelAssess, &secretConfig, testEdgeConnect)

	builder.Assess("get tenant config", getTenantConfig(testECname, secretConfig, edgeConnectTenantConfig))
	builder.Assess("get EC status", ecComponents.Get(&testEdgeConnect))
	builder.Assess("check if EC configuration exists on the tenant", ecComponents.CheckEcExistsOnTheTenant(secretConfig, edgeConnectTenantConfig))
	builder.Assess("delete EC custom resource", ecComponents.Delete(testEdgeConnect))
	builder.Assess("check if EC configuration is deleted on the tenant", checkEcNotExistsOnTheTenant(secretConfig, edgeConnectTenantConfig))

	builder.Teardown(tenant.DeleteTenantSecret(ecComponents.BuildOAuthClientSecretName(testEdgeConnect.Name), testEdgeConnect.Namespace))

	return builder.Feature()
}

var (
	customServiceAccount = filepath.Join(project.TestDataDir(), "edgeconnect/custom-service-account.yaml")
)

func AutomationModeFeature(t *testing.T) features.Feature {
	const customServiceAccountName = "custom-edgeconnect"

	builder := features.New("edgeconnect-install-k8s-automation")

	secretConfig := tenant.GetEdgeConnectTenantSecret(t)

	edgeConnectTenantConfig := &ecComponents.TenantConfig{}
	testECname := uuid.NewString()

	testEdgeConnect := *ecComponents.New(
		ecComponents.WithName(testECname),
		ecComponents.WithAPIServer(secretConfig.APIServer),
		ecComponents.WithOAuthClientSecret(ecComponents.BuildOAuthClientSecretName(testECname)),
		ecComponents.WithOAuthEndpoint("https://sso-dev.dynatracelabs.com/sso/oauth2/token"),
		ecComponents.WithOAuthResource(secretConfig.Resource),
		ecComponents.WithProvisionerMode(true),
		ecComponents.WithK8SAutomationMode(true),
		ecComponents.WithServiceAccount(customServiceAccountName),
	)

	builder.Assess("create ServiceAccount", createServiceAccount())

	ecComponents.Install(builder, helpers.LevelAssess, &secretConfig, testEdgeConnect)

	builder.Assess("get tenant config", getTenantConfig(testECname, secretConfig, edgeConnectTenantConfig))
	builder.Assess("get EC status", ecComponents.Get(&testEdgeConnect))

	builder.Assess("check if EC configuration exists on the tenant", ecComponents.CheckEcExistsOnTheTenant(secretConfig, edgeConnectTenantConfig))
	// k8sautomation.HostPattern has to be executed when the test is running and testEdgeConnect.Status contains real data
	builder.Assess("check hostPatterns - k8s automation", checkHostPatternOnTheTenant(secretConfig, edgeConnectTenantConfig, func() string { //nolint
		return testEdgeConnect.K8sAutomationHostPattern()
	}))
	builder.Assess("check if settings object exists on the tenant", checkSettingsExistsOnTheTenant(secretConfig, &testEdgeConnect))
	builder.Assess("delete EC custom resource", ecComponents.Delete(testEdgeConnect))
	builder.Assess("check if EC configuration is deleted on the tenant", checkEcNotExistsOnTheTenant(secretConfig, edgeConnectTenantConfig))
	builder.Assess("check if settings object is deleted on the tenant", checkSettingsNotExistsOnTheTenant(secretConfig, &testEdgeConnect))

	builder.Teardown(tenant.DeleteTenantSecret(ecComponents.BuildOAuthClientSecretName(testEdgeConnect.Name), testEdgeConnect.Namespace))
	builder.Teardown(deleteServiceAccount())

	return builder.Feature()
}

// getTenantConfig for Provisioner and K8SAutomation modes, preserves the id of EdgeConnect configuration on the tenant
func getTenantConfig(ecName string, clientSecret tenant.EdgeConnectSecret, edgeConnectTenantConfig *ecComponents.TenantConfig) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		ecClt, err := ecComponents.BuildEcClient(ctx, clientSecret)
		require.NoError(t, err)

		ecs, err := ecClt.GetEdgeConnects(ecName)
		require.NoError(t, err)

		assert.LessOrEqual(t, len(ecs.EdgeConnects), 1, "Found multiple EdgeConnect objects with the same tenantConfigName", "count", ecs.EdgeConnects)
		assert.NotEmpty(t, ecs.EdgeConnects, "EdgeConnect object not found", "count", ecs.EdgeConnects)

		assert.Equal(t, ecName, ecs.EdgeConnects[0].Name, "expected EC object not found on the tenant")
		assert.True(t, ecs.EdgeConnects[0].ManagedByDynatraceOperator)

		// the ID of EC configuration on the tenant is important only
		// the OAuth clientSecret used by the test and the OAuth secret used by the operator are the same
		edgeConnectTenantConfig.ID = ecs.EdgeConnects[0].ID

		return ctx
	}
}

func checkHostPatternOnTheTenant(clientSecret tenant.EdgeConnectSecret, edgeConnectTenantConfig *ecComponents.TenantConfig, hostPattern func() string) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		ecClt, err := ecComponents.BuildEcClient(ctx, clientSecret)
		require.NoError(t, err)

		ec, err := ecClt.GetEdgeConnect(edgeConnectTenantConfig.ID)
		require.NoError(t, err)

		host := hostPattern()
		assert.True(t, slices.Contains(ec.HostPatterns, host), "hostPattern %s not found", host)

		return ctx
	}
}

func checkEcNotExistsOnTheTenant(clientSecret tenant.EdgeConnectSecret, edgeConnectTenantConfig *ecComponents.TenantConfig) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		ecClt, err := ecComponents.BuildEcClient(ctx, clientSecret)
		require.NoError(t, err)

		_, err = ecClt.GetEdgeConnect(edgeConnectTenantConfig.ID)
		// err.Message: Unknown key: eb27ac05-c0c7-4d88-9bb1-804b39e3429b
		// err.Code: 404
		require.Error(t, err)

		return ctx
	}
}

func checkSettingsExistsOnTheTenant(clientSecret tenant.EdgeConnectSecret, testEdgeConnect *edgeconnect.EdgeConnect) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		ecClt, err := ecComponents.BuildEcClient(ctx, clientSecret)
		require.NoError(t, err)

		require.NotEmpty(t, testEdgeConnect.Status.KubeSystemUID)

		envSetting, err := controller.GetConnectionSetting(ecClt, testEdgeConnect.Name, testEdgeConnect.Namespace, testEdgeConnect.Status.KubeSystemUID)
		require.NoError(t, err)

		assert.Equal(t, testEdgeConnect.Name, envSetting.Value.Name)
		assert.Equal(t, testEdgeConnect.Namespace, envSetting.Value.Namespace)

		return ctx
	}
}

func checkSettingsNotExistsOnTheTenant(clientSecret tenant.EdgeConnectSecret, testEdgeConnect *edgeconnect.EdgeConnect) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		ecClt, err := ecComponents.BuildEcClient(ctx, clientSecret)
		require.NoError(t, err)

		require.NotEmpty(t, testEdgeConnect.Status.KubeSystemUID)

		se, err := controller.GetConnectionSetting(ecClt, testEdgeConnect.Name, testEdgeConnect.Namespace, testEdgeConnect.Status.KubeSystemUID)
		require.NoError(t, err)
		assert.Equal(t, edgeconnectClient.EnvironmentSetting{}, se)

		return ctx
	}
}

func createServiceAccount() features.Func {
	return helpers.ToFeatureFunc(manifests.InstallFromFile(customServiceAccount), true)
}

func deleteServiceAccount() features.Func {
	return helpers.ToFeatureFunc(manifests.UninstallFromFile(customServiceAccount), true)
}

func updateHostPatterns(testEdgeConnect *edgeconnect.EdgeConnect, hostPattern string) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		testEdgeConnect.Spec.HostPatterns = []string{
			hostPattern,
		}
		err := envConfig.Client().Resources().Update(ctx, testEdgeConnect)
		if err != nil {
			t.Error("failed to update EdgeConnect CR", err)

			return ctx
		}
		const timeout = 2 * time.Minute
		resources := envConfig.Client().Resources()
		err = wait.For(conditions.New(resources).ResourceMatch(testEdgeConnect, func(object k8s.Object) bool {
			ec, isEC := object.(*edgeconnect.EdgeConnect)

			return isEC && ec.Status.DeploymentPhase == status.Running
		}), wait.WithTimeout(timeout))

		require.NoError(t, err)

		return ctx
	}
}
