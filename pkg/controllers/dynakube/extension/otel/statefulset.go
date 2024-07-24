package otel

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/statefulset"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

const (
	statefulsetName = "dynatrace-extensions-collector"
)

func (r *reconciler) buildStatefulset() (*appsv1.StatefulSet, error) {
	container := corev1.Container{}

	return statefulset.Build(r.dk, statefulsetName, container)
}

func (r *reconciler) createOrUpdateStatefulset(ctx context.Context) error {
	return nil
}
