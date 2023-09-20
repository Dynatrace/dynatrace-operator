//go:build e2e

package edgeconnect

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/src/api/status"
	"github.com/Dynatrace/dynatrace-operator/src/api/v1alpha1"
	edgeconnectv1alpha1 "github.com/Dynatrace/dynatrace-operator/src/api/v1alpha1/edgeconnect"
	"github.com/stretchr/testify/require"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/e2e-framework/klient/k8s"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/klient/wait/conditions"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

const (
	defaultName      = "edgeconnect"
	defaultNamespace = "dynatrace"
)

type Builder struct {
	edgeConnect edgeconnectv1alpha1.EdgeConnect
}

func NewBuilder() Builder {
	return Builder{
		edgeConnect: edgeconnectv1alpha1.EdgeConnect{
			ObjectMeta: metav1.ObjectMeta{
				Name:      defaultName,
				Namespace: defaultNamespace,
			},
			Spec:   edgeconnectv1alpha1.EdgeConnectSpec{},
			Status: edgeconnectv1alpha1.EdgeConnectStatus{},
		},
	}
}

func (edgeConnectBuilder Builder) Name(name string) Builder {
	edgeConnectBuilder.edgeConnect.ObjectMeta.Name = name
	return edgeConnectBuilder
}

func (edgeConnectBuilder Builder) ApiServer(apiURL string) Builder {
	edgeConnectBuilder.edgeConnect.Spec.ApiServer = apiURL
	return edgeConnectBuilder
}

func (edgeConnectBuilder Builder) OAuthClientSecret(clientSecretName string) Builder {
	edgeConnectBuilder.edgeConnect.Spec.OAuth.ClientSecret = clientSecretName
	return edgeConnectBuilder
}

func (edgeConnectBuilder Builder) OAuthResource(resource string) Builder {
	edgeConnectBuilder.edgeConnect.Spec.OAuth.Resource = resource
	return edgeConnectBuilder
}

func (edgeConnectBuilder Builder) OAuthEndpoint(endpoint string) Builder {
	edgeConnectBuilder.edgeConnect.Spec.OAuth.Endpoint = endpoint
	return edgeConnectBuilder
}

func (edgeConnectBuilder Builder) CustomPullSecret(secretName string) Builder {
	edgeConnectBuilder.edgeConnect.Spec.CustomPullSecret = secretName
	return edgeConnectBuilder
}

func (edgeConnectBuilder Builder) Build() edgeconnectv1alpha1.EdgeConnect {
	return edgeConnectBuilder.edgeConnect
}

func Create(edgeConnect edgeconnectv1alpha1.EdgeConnect) features.Func {
	return func(ctx context.Context, t *testing.T, environmentConfig *envconf.Config) context.Context {
		require.NoError(t, v1alpha1.AddToScheme(environmentConfig.Client().Resources().GetScheme()))
		require.NoError(t, environmentConfig.Client().Resources().Create(ctx, &edgeConnect))
		return ctx
	}
}

func Delete(edgeConnect edgeconnectv1alpha1.EdgeConnect) features.Func {
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

		err = wait.For(conditions.New(resources).ResourceDeleted(&edgeConnect))
		require.NoError(t, err)
		return ctx
	}
}

func WaitForEdgeConnectPhase(edgeConnect edgeconnectv1alpha1.EdgeConnect, phase status.DeploymentPhase) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		resources := envConfig.Client().Resources()

		err := wait.For(conditions.New(resources).ResourceMatch(&edgeConnect, func(object k8s.Object) bool {
			ec, isEdgeConnect := object.(*edgeconnectv1alpha1.EdgeConnect)
			return isEdgeConnect && ec.Status.DeploymentPhase == phase
		}))

		require.NoError(t, err)

		return ctx
	}
}
