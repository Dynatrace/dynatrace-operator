package k8shpa

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/e2e-framework/klient/k8s"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/klient/wait/conditions"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

func Create(hpa *autoscalingv1.HorizontalPodAutoscaler) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		err := envConfig.Client().Resources().Create(ctx, hpa)

		if k8serrors.IsAlreadyExists(err) {
			require.NoError(t, envConfig.Client().Resources().Update(ctx, hpa))

			return ctx
		}

		require.NoError(t, err)

		return ctx
	}
}

func Delete(hpa *autoscalingv1.HorizontalPodAutoscaler) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		err := envConfig.Client().Resources().Delete(ctx, hpa)

		if err != nil {
			if k8serrors.IsNotFound(err) {
				err = nil
			}
		}

		require.NoError(t, err)

		return ctx
	}
}

func WaitCurrentReplicas(hpa *autoscalingv1.HorizontalPodAutoscaler, replicas int32) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		resources := envConfig.Client().Resources()

		err := wait.For(conditions.New(resources).ResourceScaled(hpa, func(object k8s.Object) int32 {
			return object.(*autoscalingv1.HorizontalPodAutoscaler).Status.CurrentReplicas
		}, replicas), wait.WithTimeout(5*time.Minute))

		require.NoError(t, err)

		return ctx
	}
}
