//go:build e2e

package deployment

import (
	"context"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
)

type PodConsumer func(pod corev1.Pod)

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

func (query *Query) Get() (appsv1.Deployment, error) {
	var deployment appsv1.Deployment
	err := query.resource.Get(query.ctx, query.objectKey.Name, query.objectKey.Namespace, &deployment)

	return deployment, err
}

func (query *Query) ForEachPod(consumer PodConsumer) error {
	var pods corev1.PodList
	deployment, err := query.Get()

	if err != nil {
		return err
	}

	err = query.resource.List(query.ctx, &pods, resources.WithLabelSelector(labels.FormatLabels(deployment.Spec.Selector.MatchLabels)))

	if err != nil {
		return err
	}

	for _, pod := range pods.Items {
		consumer(pod)
	}

	return nil
}
