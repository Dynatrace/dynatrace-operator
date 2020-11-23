package dtversion

import (
	"context"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-operator/pkg/apis/dynatrace/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type PodFinder struct {
	client.Client
	instance *dynatracev1alpha1.DynaKube
	labels   map[string]string
}

func NewPodFinder(clt client.Client, instance *dynatracev1alpha1.DynaKube, matchLabels map[string]string) *PodFinder {
	return &PodFinder{
		Client:   clt,
		instance: instance,
		labels:   matchLabels,
	}
}

func (r *PodFinder) FindPods() ([]corev1.Pod, error) {
	podList := &corev1.PodList{}
	err := r.List(context.TODO(), podList, r.buildListOptions()...)
	if err != nil {
		return nil, err
	}
	return podList.Items, nil
}

func (r *PodFinder) buildListOptions() []client.ListOption {
	return []client.ListOption{
		client.InNamespace(r.instance.Namespace),
		client.MatchingLabels(r.labels),
	}
}
