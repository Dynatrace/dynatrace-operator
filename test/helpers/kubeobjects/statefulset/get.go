//go:build e2e

package statefulset

import (
	"context"

	appsv1 "k8s.io/api/apps/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
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
