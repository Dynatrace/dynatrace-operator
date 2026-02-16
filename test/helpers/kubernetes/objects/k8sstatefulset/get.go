//go:build e2e

package k8sstatefulset

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

func Get(ctx context.Context, resource *resources.Resources, name, namespace string) (appsv1.StatefulSet, error) {
	var stateFulSet appsv1.StatefulSet
	err := resource.Get(ctx, name, namespace, &stateFulSet)

	return stateFulSet, err
}

func IsReady(name, namespace string) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		resources := envConfig.Client().Resources()
		sts, err := Get(ctx, resources, name, namespace)
		require.NoError(t, err)
		assert.Equal(t, sts.Status.Replicas, sts.Status.ReadyReplicas)

		return ctx
	}
}
