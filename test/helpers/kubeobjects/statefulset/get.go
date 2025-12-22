//go:build e2e

package statefulset

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

type Query struct {
	ctx       context.Context
	resource  *resources.Resources
	objectKey client.ObjectKey
}

func NewQuery(ctx context.Context, resource *resources.Resources, objectKey client.ObjectKey) *Query {
	return &Query{
		ctx:       ctx,
		resource:  resource,
		objectKey: objectKey,
	}
}

func (query *Query) Get() (appsv1.StatefulSet, error) {
	var stateFulSet appsv1.StatefulSet
	err := query.resource.Get(query.ctx, query.objectKey.Name, query.objectKey.Namespace, &stateFulSet)

	return stateFulSet, err
}

func IsReady(name, namespace string) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		resources := envConfig.Client().Resources()
		sts, err := NewQuery(ctx, resources, client.ObjectKey{Name: name, Namespace: namespace}).Get()
		require.NoError(t, err)
		assert.Equal(t, sts.Status.Replicas, sts.Status.ReadyReplicas)

		return ctx
	}
}
