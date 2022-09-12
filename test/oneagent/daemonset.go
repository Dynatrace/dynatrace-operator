package oneagent

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/test/kubeobjects/daemonset"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/pkg/env"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

type PodConsumer func(pod corev1.Pod)

func WaitForDaemonset() features.Func {
	return daemonset.WaitFor("dynakube-oneagent", "dynatrace")
}

func WaitForDaemonSetPodsDeletion() env.Func {
	return daemonset.WaitForPodsDeletion("dynakube-oneagent", "dynatrace")
}

func Get(ctx context.Context, resource *resources.Resources) (appsv1.DaemonSet, error) {
	var daemonSet appsv1.DaemonSet
	err := resource.Get(ctx, "dynakube-oneagent", "dynatrace", &daemonSet)
	return daemonSet, err
}

func ForEachPod(ctx context.Context, resource *resources.Resources, consumer PodConsumer) error {
	var pods corev1.PodList
	daemonSet, err := Get(ctx, resource)

	if err != nil {
		return err
	}

	err = resource.List(ctx, &pods, resources.WithLabelSelector(labels.FormatLabels(daemonSet.Labels)))

	if err != nil {
		return err
	}

	for _, pod := range pods.Items {
		consumer(pod)
	}

	return nil
}
