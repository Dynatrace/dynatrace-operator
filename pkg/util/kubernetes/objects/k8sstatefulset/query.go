package k8sstatefulset

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/hasher"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8slabel"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/objects/internal/query"
	appsv1 "k8s.io/api/apps/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func Query(kubeClient client.Client, kubeReader client.Reader, log logd.Logger) query.Generic[*appsv1.StatefulSet, *appsv1.StatefulSetList] {
	return query.Generic[*appsv1.StatefulSet, *appsv1.StatefulSetList]{
		Target:     &appsv1.StatefulSet{},
		ListTarget: &appsv1.StatefulSetList{},
		ToList: func(list *appsv1.StatefulSetList) []*appsv1.StatefulSet {
			out := make([]*appsv1.StatefulSet, len(list.Items))
			for i, item := range list.Items {
				out[i] = &item
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

func isEqual(current, desired *appsv1.StatefulSet) bool {
	return !hasher.IsAnnotationDifferent(current, desired)
}

func mustRecreate(current, desired *appsv1.StatefulSet) bool {
	currentHash := current.Annotations[AnnotationPVCHash]
	desiredHash := desired.Annotations[AnnotationPVCHash]

	return k8slabel.NotEqual(current.Spec.Selector.MatchLabels, desired.Spec.Selector.MatchLabels) || currentHash != desiredHash
}
