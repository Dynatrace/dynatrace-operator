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
	"github.com/Dynatrace/dynatrace-operator/test/helpers/tenant"
	"github.com/stretchr/testify/require"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"sigs.k8s.io/e2e-framework/klient/k8s"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/klient/wait/conditions"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

func Install(builder *features.FeatureBuilder, level features.Level, secretConfig *tenant.EdgeConnectSecret, testEdgeConnect edgeconnect.EdgeConnect) {
	if secretConfig != nil {
		builder.WithStep("create edgeconnect client secret", level, tenant.CreateClientSecret(*secretConfig, fmt.Sprintf("%s-client-secret", testEdgeConnect.Name), testEdgeConnect.Namespace))
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
