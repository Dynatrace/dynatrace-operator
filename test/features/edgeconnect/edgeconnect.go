//go:build e2e

package edgeconnect

import (
	"fmt"
	"testing"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	edgeconnectv1alpha2 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha2/edgeconnect"
	edgeconnectClient "github.com/Dynatrace/dynatrace-operator/pkg/clients/edgeconnect"
	controller "github.com/Dynatrace/dynatrace-operator/pkg/controllers/edgeconnect"
	"github.com/Dynatrace/dynatrace-operator/test/helpers"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/edgeconnect"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/tenant"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/exp/slices"
	"golang.org/x/net/context"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/e2e-framework/klient/k8s"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/klient/wait/conditions"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

const (
	testServiceAccountName = "custom-edgeconnect-service-name"
	testNamespaceName      = "dynatrace"
)

type tenantConfig struct {
	// id of the EdgeConnect configuration on the tenant (managed by the operator or the test)
	id string
	// secret OAuth2 client credentials created by the Dynatrace API (normal mode)
	secret tenant.EdgeConnectSecret
}

func NormalModeFeature(t *testing.T) features.Feature {
	builder := features.New("edgeconnect-install")

	secretConfig := tenant.GetEdgeConnectTenantSecret(t)

	edgeConnectTenantConfig := &tenantConfig{}

	testECname := uuid.NewString()
	testHostPattern := fmt.Sprintf("%s.e2eTestHostPattern.internal.org", testECname)

	builder.Assess("create EC configuration on the tenant", createTenantConfig(testECname, secretConfig, edgeConnectTenantConfig, testHostPattern))

	testEdgeConnect := *edgeconnect.New(
		// this tenantConfigName should match with tenant edge connect tenantConfigName
		edgeconnect.WithName(testECname),
		edgeconnect.WithApiServer(secretConfig.ApiServer),
		edgeconnect.WithOAuthClientSecret(buildOAuthClientSecretName(testECname)),
		edgeconnect.WithOAuthEndpoint("https://sso-dev.dynatracelabs.com/sso/oauth2/token"),
	)

	// create OAuth client secret related to the specific EdgeConnect configuration on the tenant
	builder.Assess("create client secret", tenant.CreateClientSecret(edgeConnectTenantConfig.secret, buildOAuthClientSecretName(testEdgeConnect.Name), testEdgeConnect.Namespace))

	// Register operator install
	edgeconnect.Install(builder, helpers.LevelAssess, nil, testEdgeConnect)

	builder.Assess("check EC configuration on the tenant", checkEcExistsOnTheTenant(secretConfig, edgeConnectTenantConfig))
	builder.Assess("delete EdgeConnect CR", edgeconnect.Delete(testEdgeConnect))
	builder.Assess("check if EC configuration is deleted on the tenant", checkEcExistsOnTheTenant(secretConfig, edgeConnectTenantConfig))

	builder.Teardown(tenant.DeleteTenantSecret(buildOAuthClientSecretName(testEdgeConnect.Name), testEdgeConnect.Namespace))
	builder.Teardown(deleteTenantConfig(secretConfig, edgeConnectTenantConfig))

	return builder.Feature()
}

func ProvisionerModeFeature(t *testing.T) features.Feature {
	builder := features.New("edgeconnect-install-provisioner")

	secretConfig := tenant.GetEdgeConnectTenantSecret(t)

	edgeConnectTenantConfig := &tenantConfig{}

	testECname := uuid.NewString()
	testHostPattern := fmt.Sprintf("%s.e2eTestHostPattern.internal.org", testECname)
	testHostPattern2 := fmt.Sprintf("%s.e2eTestHostPattern2.internal.org", testECname)

	testEdgeConnect := *edgeconnect.New(
		// this tenantConfigName should match with tenant edge connect tenantConfigName
		edgeconnect.WithName(testECname),
		edgeconnect.WithApiServer(secretConfig.ApiServer),
		edgeconnect.WithOAuthClientSecret(buildOAuthClientSecretName(testECname)),
		edgeconnect.WithOAuthEndpoint("https://sso-dev.dynatracelabs.com/sso/oauth2/token"),
		edgeconnect.WithOAuthResource(secretConfig.Resource),
		edgeconnect.WithProvisionerMode(true),
		edgeconnect.WithHostPattern(testHostPattern),
	)

	// Register operator install
	edgeconnect.Install(builder, helpers.LevelAssess, &secretConfig, testEdgeConnect)

	builder.Assess("get tenant config", getTenantConfig(testECname, secretConfig, edgeConnectTenantConfig))
	builder.Assess("get EC status", edgeconnect.Get(&testEdgeConnect))

	builder.Assess("check if EC configuration exists on the tenant", checkEcExistsOnTheTenant(secretConfig, edgeConnectTenantConfig))
	builder.Assess("check hostPatterns on the tenant - testHostPattern", checkHostPatternOnTheTenant(secretConfig, edgeConnectTenantConfig, func() string { return testHostPattern }))
	builder.Assess("update hostPatterns", updateHostPatterns(&testEdgeConnect, testHostPattern2))
	builder.Assess("check hostPatterns on the tenant - testHostPattern2", checkHostPatternOnTheTenant(secretConfig, edgeConnectTenantConfig, func() string { return testHostPattern2 }))
	builder.Assess("delete EC custom resource", edgeconnect.Delete(testEdgeConnect))
	builder.Assess("check if EC configuration is deleted on the tenant", checkEcNotExistsOnTheTenant(secretConfig, edgeConnectTenantConfig))

	builder.Teardown(tenant.DeleteTenantSecret(buildOAuthClientSecretName(testEdgeConnect.Name), testEdgeConnect.Namespace))

	return builder.Feature()
}

func AutomationModeFeature(t *testing.T) features.Feature {
	builder := features.New("edgeconnect-install-k8s-automation")

	secretConfig := tenant.GetEdgeConnectTenantSecret(t)

	edgeConnectTenantConfig := &tenantConfig{}
	testECname := uuid.NewString()

	testEdgeConnect := *edgeconnect.New(
		// this tenantConfigName should match with tenant edge connect tenantConfigName
		edgeconnect.WithName(testECname),
		edgeconnect.WithApiServer(secretConfig.ApiServer),
		edgeconnect.WithOAuthClientSecret(buildOAuthClientSecretName(testECname)),
		edgeconnect.WithOAuthEndpoint("https://sso-dev.dynatracelabs.com/sso/oauth2/token"),
		edgeconnect.WithOAuthResource(secretConfig.Resource),
		edgeconnect.WithProvisionerMode(true),
		edgeconnect.WithK8SAutomationMode(true),
		edgeconnect.WithServiceAccount(testServiceAccountName),
	)

	builder.Assess("create ServiceAccount", createServiceAccount())

	// Register operator install
	edgeconnect.Install(builder, helpers.LevelAssess, &secretConfig, testEdgeConnect)

	builder.Assess("get tenant config", getTenantConfig(testECname, secretConfig, edgeConnectTenantConfig))
	builder.Assess("get EC status", edgeconnect.Get(&testEdgeConnect))

	builder.Assess("check if EC configuration exists on the tenant", checkEcExistsOnTheTenant(secretConfig, edgeConnectTenantConfig))
	// k8sautomation.HostPattern has to be executed when the test is running and testEdgeConnect.Status contains real data
	builder.Assess("check hostPatterns - k8s automation", checkHostPatternOnTheTenant(secretConfig, edgeConnectTenantConfig, func() string { //nolint
		return testEdgeConnect.K8sAutomationHostPattern()
	}))
	builder.Assess("check if settings object exists on the tenant", checkSettingsExistsOnTheTenant(secretConfig, &testEdgeConnect))
	builder.Assess("delete EC custom resource", edgeconnect.Delete(testEdgeConnect))
	builder.Assess("check if EC configuration is deleted on the tenant", checkEcNotExistsOnTheTenant(secretConfig, edgeConnectTenantConfig))
	builder.Assess("check if settings object is deleted on the tenant", checkSettingsNotExistsOnTheTenant(secretConfig, &testEdgeConnect))

	builder.Teardown(tenant.DeleteTenantSecret(buildOAuthClientSecretName(testEdgeConnect.Name), testEdgeConnect.Namespace))
	builder.Teardown(deleteServiceAccount())

	return builder.Feature()
}

// createTenantConfig for Normal mode only, preserves the id and OAuth secret of EdgeConnect configuration on the tenant
func createTenantConfig(ecName string, clientSecret tenant.EdgeConnectSecret, edgeConnectTenantConfig *tenantConfig, testHostPattern string) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		ecClt, err := buildEcClient(ctx, clientSecret)
		require.NoError(t, err)

		edgeConnectRequest := edgeconnectClient.NewRequest(ecName, []string{testHostPattern}, []edgeconnectv1alpha2.HostMapping{}, "")
		edgeConnectRequest.ManagedByDynatraceOperator = false

		res, err := ecClt.CreateEdgeConnect(edgeConnectRequest)
		require.NoError(t, err)
		assert.Equal(t, ecName, res.Name)

		edgeConnectTenantConfig.secret.Name = ecName
		edgeConnectTenantConfig.secret.ApiServer = clientSecret.ApiServer
		edgeConnectTenantConfig.secret.TenantUid = clientSecret.TenantUid
		edgeConnectTenantConfig.secret.OauthClientId = res.OauthClientId
		edgeConnectTenantConfig.secret.OauthClientSecret = res.OauthClientSecret
		edgeConnectTenantConfig.secret.Resource = res.OauthClientResource
		edgeConnectTenantConfig.id = res.ID

		return ctx
	}
}

// getTenantConfig for Provisioner and K8SAutomation modes, preserves the id of EdgeConnect configuration on the tenant
func getTenantConfig(ecName string, clientSecret tenant.EdgeConnectSecret, edgeConnectTenantConfig *tenantConfig) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		ecClt, err := buildEcClient(ctx, clientSecret)
		require.NoError(t, err)

		ecs, err := ecClt.GetEdgeConnects(ecName)
		require.NoError(t, err)

		assert.LessOrEqual(t, len(ecs.EdgeConnects), 1, "Found multiple EdgeConnect objects with the same tenantConfigName", "count", ecs.EdgeConnects)
		assert.NotEmpty(t, ecs.EdgeConnects, "EdgeConnect object not found", "count", ecs.EdgeConnects)

		assert.Equal(t, ecName, ecs.EdgeConnects[0].Name, "expected EC object not found on the tenant")
		assert.True(t, ecs.EdgeConnects[0].ManagedByDynatraceOperator)

		// the id of EC configuration on the tenant is important only
		// the OAuth clientSecret used by the test and the OAuth secret used by the operator are the same
		edgeConnectTenantConfig.id = ecs.EdgeConnects[0].ID

		return ctx
	}
}

func deleteTenantConfig(clientSecret tenant.EdgeConnectSecret, edgeConnectTenantConfig *tenantConfig) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		ecClt, err := buildEcClient(ctx, clientSecret)
		require.NoError(t, err)

		err = ecClt.DeleteEdgeConnect(edgeConnectTenantConfig.id)
		require.NoError(t, err)

		return ctx
	}
}

func checkEcExistsOnTheTenant(clientSecret tenant.EdgeConnectSecret, edgeConnectTenantConfig *tenantConfig) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		ecClt, err := buildEcClient(ctx, clientSecret)
		require.NoError(t, err)

		_, err = ecClt.GetEdgeConnect(edgeConnectTenantConfig.id)
		require.NoError(t, err)

		return ctx
	}
}

func checkHostPatternOnTheTenant(clientSecret tenant.EdgeConnectSecret, edgeConnectTenantConfig *tenantConfig, hostPattern func() string) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		ecClt, err := buildEcClient(ctx, clientSecret)
		require.NoError(t, err)

		ec, err := ecClt.GetEdgeConnect(edgeConnectTenantConfig.id)
		require.NoError(t, err)

		host := hostPattern()
		assert.True(t, slices.Contains(ec.HostPatterns, host), "hostPattern %s not found", host)

		return ctx
	}
}

func checkEcNotExistsOnTheTenant(clientSecret tenant.EdgeConnectSecret, edgeConnectTenantConfig *tenantConfig) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		ecClt, err := buildEcClient(ctx, clientSecret)
		require.NoError(t, err)

		_, err = ecClt.GetEdgeConnect(edgeConnectTenantConfig.id)
		// err.Message: Unknown key: eb27ac05-c0c7-4d88-9bb1-804b39e3429b
		// err.Code: 404
		require.Error(t, err)

		return ctx
	}
}

func checkSettingsExistsOnTheTenant(clientSecret tenant.EdgeConnectSecret, testEdgeConnect *edgeconnectv1alpha2.EdgeConnect) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		ecClt, err := buildEcClient(ctx, clientSecret)
		require.NoError(t, err)

		require.NotEmpty(t, testEdgeConnect.Status.KubeSystemUID)

		envSetting, err := controller.GetConnectionSetting(ecClt, testEdgeConnect.Name, testEdgeConnect.Namespace, testEdgeConnect.Status.KubeSystemUID)
		require.NoError(t, err)

		assert.Equal(t, testEdgeConnect.Name, envSetting.Value.Name)
		assert.Equal(t, testEdgeConnect.Namespace, envSetting.Value.Namespace)

		return ctx
	}
}

func checkSettingsNotExistsOnTheTenant(clientSecret tenant.EdgeConnectSecret, testEdgeConnect *edgeconnectv1alpha2.EdgeConnect) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		ecClt, err := buildEcClient(ctx, clientSecret)
		require.NoError(t, err)

		require.NotEmpty(t, testEdgeConnect.Status.KubeSystemUID)

		se, err := controller.GetConnectionSetting(ecClt, testEdgeConnect.Name, testEdgeConnect.Namespace, testEdgeConnect.Status.KubeSystemUID)
		require.NoError(t, err)
		assert.Equal(t, edgeconnectClient.EnvironmentSetting{}, se)

		return ctx
	}
}

func buildEcClient(ctx context.Context, secret tenant.EdgeConnectSecret) (edgeconnectClient.Client, error) {
	clt, err := edgeconnectClient.NewClient(
		secret.OauthClientId,
		secret.OauthClientSecret,
		edgeconnectClient.WithBaseURL("https://"+secret.ApiServer),
		edgeconnectClient.WithTokenURL("https://sso-dev.dynatracelabs.com/sso/oauth2/token"),
		edgeconnectClient.WithOauthScopes([]string{
			"app-engine:edge-connects:read",
			"app-engine:edge-connects:write",
			"app-engine:edge-connects:delete",
			"oauth2:clients:manage",
			"settings:objects:read",
			"settings:objects:write",
		}),
		edgeconnectClient.WithContext(ctx),
	)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return clt, nil
}

func createServiceAccount() features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		err := envConfig.Client().Resources().Create(ctx, serviceAccount(testServiceAccountName, testNamespaceName))
		if err != nil {
			t.Error("failed to create service account", err)

			return ctx
		}

		return ctx
	}
}

func deleteServiceAccount() features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		err := envConfig.Client().Resources().Delete(ctx, serviceAccount(testServiceAccountName, testNamespaceName))
		if err != nil {
			t.Error("failed to delete service account", err)

			return ctx
		}

		return ctx
	}
}

func serviceAccount(name string, namespace string) *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
}

func updateHostPatterns(testEdgeConnect *edgeconnectv1alpha2.EdgeConnect, hostPattern string) features.Func {
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
			ec, isEC := object.(*edgeconnectv1alpha2.EdgeConnect)

			return isEC && ec.Status.DeploymentPhase == status.Running
		}), wait.WithTimeout(timeout))

		require.NoError(t, err)

		return ctx
	}
}

func buildOAuthClientSecretName(secretName string) string {
	return fmt.Sprintf("%s-client-secret", secretName)
}
