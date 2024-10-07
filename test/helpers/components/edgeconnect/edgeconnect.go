//go:build e2e

package edgeconnect

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha1"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha2/edgeconnect"
	edgeconnectClient "github.com/Dynatrace/dynatrace-operator/pkg/clients/edgeconnect"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/tenant"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"sigs.k8s.io/e2e-framework/klient/k8s"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/klient/wait/conditions"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

type TenantConfig struct {
	// ID of the EdgeConnect configuration on the tenant (managed by the operator or the test)
	ID string
	// Secret OAuth2 client credentials created by the Dynatrace API (normal mode)
	Secret tenant.EdgeConnectSecret
}

func Install(builder *features.FeatureBuilder, level features.Level, secretConfig *tenant.EdgeConnectSecret, testEdgeConnect edgeconnect.EdgeConnect) {
	if secretConfig != nil {
		builder.WithStep("create edgeconnect client Secret", level, tenant.CreateClientSecret(secretConfig, BuildOAuthClientSecretName(testEdgeConnect.Name), testEdgeConnect.Namespace))
	}
	builder.WithStep(
		fmt.Sprintf("'%s' edgeconnect created", testEdgeConnect.Name),
		level,
		Create(testEdgeConnect))
	VerifyStartup(builder, level, testEdgeConnect)
}

func VerifyStartup(builder *features.FeatureBuilder, level features.Level, testEdgeConnect edgeconnect.EdgeConnect) {
	builder.WithStep(
		fmt.Sprintf("'%s' edgeconnect phase changes to 'Running'", testEdgeConnect.Name),
		level,
		WaitForPhase(testEdgeConnect, status.Running))
}

func Create(edgeConnect edgeconnect.EdgeConnect) features.Func {
	return func(ctx context.Context, t *testing.T, environmentConfig *envconf.Config) context.Context {
		require.NoError(t, environmentConfig.Client().Resources().Create(ctx, &edgeConnect))

		return ctx
	}
}

func Get(ec *edgeconnect.EdgeConnect) features.Func {
	return func(ctx context.Context, t *testing.T, environmentConfig *envconf.Config) context.Context {
		require.NoError(t, environmentConfig.Client().Resources().Get(ctx, ec.Name, ec.Namespace, ec))

		return ctx
	}
}

func Delete(edgeConnect edgeconnect.EdgeConnect) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		resources := envConfig.Client().Resources()

		err := v1alpha1.AddToScheme(resources.GetScheme())
		require.NoError(t, err)

		err = resources.Delete(ctx, &edgeConnect)
		isNoKindMatchErr := meta.IsNoMatchError(err)

		if err != nil {
			if k8serrors.IsNotFound(err) || isNoKindMatchErr {
				// If the edgeconnect itself or the crd does not exist, everything is fine
				err = nil
			}
			require.NoError(t, err)
		}

		err = wait.For(conditions.New(resources).ResourceDeleted(&edgeConnect), wait.WithTimeout(1*time.Minute))
		require.NoError(t, err)

		return ctx
	}
}

func WaitForPhase(edgeConnect edgeconnect.EdgeConnect, phase status.DeploymentPhase) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		resources := envConfig.Client().Resources()

		err := wait.For(conditions.New(resources).ResourceMatch(&edgeConnect, func(object k8s.Object) bool {
			ec, isEdgeConnect := object.(*edgeconnect.EdgeConnect)

			return isEdgeConnect && ec.Status.DeploymentPhase == phase
		}), wait.WithTimeout(5*time.Minute))

		require.NoError(t, err)

		return ctx
	}
}

// CreateTenantConfig for Normal mode only, preserves the ID and OAuth Secret of EdgeConnect configuration on the tenant
func CreateTenantConfig(ecName string, clientSecret tenant.EdgeConnectSecret, edgeConnectTenantConfig *TenantConfig, testHostPattern string) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		ecClt, err := BuildEcClient(ctx, clientSecret)
		require.NoError(t, err)

		edgeConnectRequest := edgeconnectClient.NewRequest(ecName, []string{testHostPattern}, []edgeconnect.HostMapping{}, "")
		edgeConnectRequest.ManagedByDynatraceOperator = false

		res, err := ecClt.CreateEdgeConnect(edgeConnectRequest)
		require.NoError(t, err)
		assert.Equal(t, ecName, res.Name)

		edgeConnectTenantConfig.Secret.Name = ecName
		edgeConnectTenantConfig.Secret.ApiServer = clientSecret.ApiServer
		edgeConnectTenantConfig.Secret.TenantUid = clientSecret.TenantUid
		edgeConnectTenantConfig.Secret.OauthClientId = res.OauthClientId
		edgeConnectTenantConfig.Secret.OauthClientSecret = res.OauthClientSecret
		edgeConnectTenantConfig.Secret.Resource = res.OauthClientResource
		edgeConnectTenantConfig.ID = res.ID

		return ctx
	}
}

func BuildEcClient(ctx context.Context, secret tenant.EdgeConnectSecret) (edgeconnectClient.Client, error) {
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

func BuildOAuthClientSecretName(secretName string) string {
	return fmt.Sprintf("client-secret-%s", secretName)
}

func DeleteTenantConfig(clientSecret tenant.EdgeConnectSecret, edgeConnectTenantConfig *TenantConfig) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		ecClt, err := BuildEcClient(ctx, clientSecret)
		require.NoError(t, err)

		err = ecClt.DeleteEdgeConnect(edgeConnectTenantConfig.ID)
		require.NoError(t, err)

		return ctx
	}
}

func CheckEcExistsOnTheTenant(clientSecret tenant.EdgeConnectSecret, edgeConnectTenantConfig *TenantConfig) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		ecClt, err := BuildEcClient(ctx, clientSecret)
		require.NoError(t, err)

		_, err = ecClt.GetEdgeConnect(edgeConnectTenantConfig.ID)
		require.NoError(t, err)

		return ctx
	}
}
