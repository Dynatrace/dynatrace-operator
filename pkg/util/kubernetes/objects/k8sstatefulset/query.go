package k8sstatefulset

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/util/hasher"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8slabel"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/objects/internal/query"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type QueryObject struct {
	query.Generic[*appsv1.StatefulSet, *appsv1.StatefulSetList]
}

func Query(kubeClient client.Client, kubeReader client.Reader) QueryObject {
	return QueryObject{
		query.Generic[*appsv1.StatefulSet, *appsv1.StatefulSetList]{
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
		},
	}
}

func isEqual(current, desired *appsv1.StatefulSet) bool {
	// the replicas check is a workaround to enforce the replica count set on the CR
	// without it any direct changes on the ss will be overseen because the hash will remain the same
	return !hasher.IsAnnotationDifferent(current, desired) && ptr.Deref(desired.Spec.Replicas, 1) == ptr.Deref(current.Spec.Replicas, 1)
}

func mustRecreate(current, desired *appsv1.StatefulSet) bool {
	currentHash := current.Annotations[AnnotationPVCHash]
	desiredHash := desired.Annotations[AnnotationPVCHash]

	return k8slabel.NotEqual(current.Spec.Selector.MatchLabels, desired.Spec.Selector.MatchLabels) || currentHash != desiredHash
}
