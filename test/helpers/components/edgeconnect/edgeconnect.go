//go:build e2e

package edgeconnect

import (
	"context"
	"fmt"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha1"
	edgeconnectv1alpha1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha1/edgeconnect"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/tenant"
	"github.com/stretchr/testify/require"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"sigs.k8s.io/e2e-framework/klient/k8s"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/klient/wait/conditions"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
	"testing"
	"time"
)

func Install(builder *features.FeatureBuilder, level features.Level, secretConfig *tenant.EdgeConnectSecret, testEdgeConnect edgeconnectv1alpha1.EdgeConnect) {
	if secretConfig != nil {
		builder.WithStep("create edgeconnect client secret", level, tenant.CreateClientSecret(*secretConfig, fmt.Sprintf("%s-client-secret", testEdgeConnect.Name), testEdgeConnect.Namespace))
		builder.WithStep("create edgeconnect docker pull secret", level, tenant.CreateDockerPullSecret(*secretConfig, fmt.Sprintf("%s-docker-pull-secret", testEdgeConnect.Name), testEdgeConnect.Namespace))
	}
	builder.WithStep(
		fmt.Sprintf("'%s' edgeconnect created", testEdgeConnect.Name),
		level,
		Create(testEdgeConnect))
	VerifyStartup(builder, level, testEdgeConnect)
}

func VerifyStartup(builder *features.FeatureBuilder, level features.Level, testEdgeConnect edgeconnectv1alpha1.EdgeConnect) {
	builder.WithStep(
		fmt.Sprintf("'%s' edgeconnect phase changes to 'Running'", testEdgeConnect.Name),
		level,
		WaitForPhase(testEdgeConnect, status.Running))
}

func Create(edgeConnect edgeconnectv1alpha1.EdgeConnect) features.Func {
	return func(ctx context.Context, t *testing.T, environmentConfig *envconf.Config) context.Context {
		require.NoError(t, dynatracev1alpha1.AddToScheme(environmentConfig.Client().Resources().GetScheme()))
		require.NoError(t, environmentConfig.Client().Resources().Create(ctx, &edgeConnect))
		return ctx
	}
}

func Delete(edgeConnect edgeconnectv1alpha1.EdgeConnect) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		resources := envConfig.Client().Resources()

		err := dynatracev1alpha1.AddToScheme(resources.GetScheme())
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

func WaitForPhase(edgeConnect edgeconnectv1alpha1.EdgeConnect, phase status.DeploymentPhase) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		resources := envConfig.Client().Resources()

		err := wait.For(conditions.New(resources).ResourceMatch(&edgeConnect, func(object k8s.Object) bool {
			ec, isEdgeConnect := object.(*edgeconnectv1alpha1.EdgeConnect)
			return isEdgeConnect && ec.Status.DeploymentPhase == phase
		}), wait.WithTimeout(5*time.Minute))

		require.NoError(t, err)

		return ctx
	}
}
