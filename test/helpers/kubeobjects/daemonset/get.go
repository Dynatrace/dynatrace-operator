//go:build e2e

package daemonset

import (
	"context"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
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

func (query *Query) Get() (appsv1.DaemonSet, error) {
	var daemonSet appsv1.DaemonSet
	err := query.resource.Get(query.ctx, query.objectKey.Name, query.objectKey.Namespace, &daemonSet)

	return daemonSet, err
}

func (query *Query) Delete() error {
	daemonSet, err := query.Get()
	if err != nil && !k8sErrors.IsNotFound(err) {
		return err
	}

	return query.resource.Delete(query.ctx, &daemonSet)
}

func (query *Query) ForEachPod(actionFunc PodConsumer) error {
	var pods corev1.PodList
	daemonSet, err := query.Get()

	if err != nil {
		return err
	}

	err = query.resource.List(query.ctx, &pods, resources.WithLabelSelector(labels.FormatLabels(daemonSet.Spec.Selector.MatchLabels)))

	if err != nil {
		return err
	}

	for _, pod := range pods.Items {
		actionFunc(pod)
	}

	return nil
}
