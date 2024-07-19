package pod

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/hasher"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/internal/query"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func Query(kubeClient client.Client, kubeReader client.Reader, log logd.Logger) query.Generic[*corev1.Pod, *corev1.PodList] {
	return query.Generic[*corev1.Pod, *corev1.PodList]{
		Target:     &corev1.Pod{},
		ListTarget: &corev1.PodList{},
		ToList: func(sl *corev1.PodList) []*corev1.Pod {
			out := []*corev1.Pod{}
			for _, s := range sl.Items {
				out = append(out, &s)
			}

			return out
		},
		IsEqual:      isEqual,
		MustRecreate: mustRecreate,

		KubeClient: kubeClient,
		KubeReader: kubeReader,
		Log:        log,
	}
}

func isEqual(current, desired *corev1.Pod) bool {
	return !hasher.IsAnnotationDifferent(current, desired) // TODO: is this relevant?
}

func mustRecreate(current, desired *corev1.Pod) bool {
	return false
}
