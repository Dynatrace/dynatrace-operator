package k8sdaemonset

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/hasher"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8slabel"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/objects/internal/query"
	appsv1 "k8s.io/api/apps/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type QueryObject = query.Generic[*appsv1.DaemonSet, *appsv1.DaemonSetList]

func Query(kubeClient client.Client, kubeReader client.Reader, log logd.Logger) QueryObject {
	return query.Generic[*appsv1.DaemonSet, *appsv1.DaemonSetList]{
		Target:     &appsv1.DaemonSet{},
		ListTarget: &appsv1.DaemonSetList{},
		ToList: func(sl *appsv1.DaemonSetList) []*appsv1.DaemonSet {
			out := []*appsv1.DaemonSet{}
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

func isEqual(current, desired *appsv1.DaemonSet) bool {
	return !hasher.IsAnnotationDifferent(current, desired)
}

func mustRecreate(current, desired *appsv1.DaemonSet) bool {
	return k8slabel.NotEqual(current.Spec.Selector.MatchLabels, desired.Spec.Selector.MatchLabels)
}
