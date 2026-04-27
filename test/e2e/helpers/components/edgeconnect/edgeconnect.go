//go:build e2e

package edgeconnect

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha1"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha2/edgeconnect"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	edgeconnectClient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/edgeconnect"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/tenant"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2/clientcredentials"
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

// Install creates a tenant secret and waits until the EdgeConnect is Running.
// It also registers the deletion of these resources in reverse order.
func Install(builder *features.FeatureBuilder, secretConfig *tenant.EdgeConnectSecret, ec edgeconnect.EdgeConnect) {
	if secretConfig != nil {
		builder.WithStep("create edgeconnect client Secret", features.LevelAssess, tenant.CreateClientSecret(secretConfig, BuildOAuthClientSecretName(ec.Name), ec.Namespace))
	}
	builder.WithStep(
		fmt.Sprintf("'%s' edgeconnect created", ec.Name),
		features.LevelAssess,
		Create(ec))
	VerifyStartup(builder, features.LevelAssess, ec)
	// The secret is required for correct cleanup, so always delete it last
	builder.WithTeardown("edgeconnect deleted", Delete(ec))
	builder.WithTeardown("delete edgeconnect client Secret", tenant.DeleteTenantSecret(BuildOAuthClientSecretName(ec.Name), ec.Namespace))
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
		clt, err := BuildClient(clientSecret)
		require.NoError(t, err)

		edgeConnectRequest := edgeconnectClient.NewCreateRequest(ecName, []string{testHostPattern}, []edgeconnect.HostMapping{})
		edgeConnectRequest.ManagedByDynatraceOperator = false

		res, err := clt.CreateEdgeConnect(ctx, edgeConnectRequest)
		require.NoError(t, err)
		assert.Equal(t, ecName, res.Name)

		edgeConnectTenantConfig.Secret.Name = ecName
		edgeConnectTenantConfig.Secret.APIServer = clientSecret.APIServer
		edgeConnectTenantConfig.Secret.TenantUID = clientSecret.TenantUID
		edgeConnectTenantConfig.Secret.OauthClientID = res.OauthClientID
		edgeConnectTenantConfig.Secret.OauthClientSecret = res.OauthClientSecret
		edgeConnectTenantConfig.Secret.Resource = res.OauthClientResource
		edgeConnectTenantConfig.ID = res.ID

		return ctx
	}
}

func BuildClient(secret tenant.EdgeConnectSecret) (edgeconnectClient.Client, error) {
	oAuthClient, err := dynatrace.NewOAuthClient(
		clientcredentials.Config{
			ClientID:     secret.OauthClientID,
			ClientSecret: secret.OauthClientSecret,
			TokenURL:     "https://sso-dev.dynatracelabs.com/sso/oauth2/token",
			Scopes: []string{
				"app-engine:edge-connects:read",
				"app-engine:edge-connects:write",
				"app-engine:edge-connects:delete",
				"oauth2:clients:manage",
				"settings:objects:read",
				"settings:objects:write",
			},
		},
		dynatrace.WithBaseURL("https://"+secret.APIServer),
		// Disable keep-alive to prevent dropped network packets in GitHub Actions environment
		dynatrace.WithKeepAlive(false),
	)

	if err != nil {
		return nil, errors.WithStack(err)
	}

	return oAuthClient.EdgeConnect, nil
}

func BuildOAuthClientSecretName(secretName string) string {
	return fmt.Sprintf("client-secret-%s", secretName)
}

func DeleteTenantConfig(clientSecret tenant.EdgeConnectSecret, edgeConnectTenantConfig *TenantConfig) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		clt, err := BuildClient(clientSecret)
		require.NoError(t, err)

		err = clt.DeleteEdgeConnect(ctx, edgeConnectTenantConfig.ID)
		// TODO: use core.IsNotFound after edgeconnect client was refactored
		if err != nil && strings.Contains(err.Error(), "server error 404: Unknown key") {
			err = nil
		}
		require.NoError(t, err)

		return ctx
	}
}

func CheckECExistsOnTheTenant(clientSecret tenant.EdgeConnectSecret, edgeConnectTenantConfig *TenantConfig) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		clt, err := BuildClient(clientSecret)
		require.NoError(t, err)

		_, err = clt.GetEdgeConnect(ctx, edgeConnectTenantConfig.ID)
		require.NoError(t, err)

		return ctx
	}
}
