package statefulset

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/hasher"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/internal/query"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/labels"
	appsv1 "k8s.io/api/apps/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type MustRecreateFunc func(current, desired *appsv1.StatefulSet) bool

func Query(kubeClient client.Client, kubeReader client.Reader, log logd.Logger) query.Generic[*appsv1.StatefulSet, *appsv1.StatefulSetList] {
	return query.Generic[*appsv1.StatefulSet, *appsv1.StatefulSetList]{
		Target:     &appsv1.StatefulSet{},
		ListTarget: &appsv1.StatefulSetList{},
		ToList: func(sl *appsv1.StatefulSetList) []*appsv1.StatefulSet {
			out := []*appsv1.StatefulSet{}
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

func RecreateQuery(kubeClient client.Client, kubeReader client.Reader, log logd.Logger, customMustRecreate MustRecreateFunc) query.Generic[*appsv1.StatefulSet, *appsv1.StatefulSetList] {
	return query.Generic[*appsv1.StatefulSet, *appsv1.StatefulSetList]{
		Target:     &appsv1.StatefulSet{},
		ListTarget: &appsv1.StatefulSetList{},
		ToList: func(sl *appsv1.StatefulSetList) []*appsv1.StatefulSet {
			out := []*appsv1.StatefulSet{}
			for _, s := range sl.Items {
				out = append(out, &s)
			}

			return out
		},
		IsEqual: isEqual,
		MustRecreate: func(current, desired *appsv1.StatefulSet) bool {
			return mustRecreate(current, desired) || customMustRecreate(current, desired)
		},

		KubeClient: kubeClient,
		KubeReader: kubeReader,
		Log:        log,
	}
}

func isEqual(current, desired *appsv1.StatefulSet) bool {
	return !hasher.IsAnnotationDifferent(current, desired)
}

func mustRecreate(current, desired *appsv1.StatefulSet) bool {
	return labels.NotEqual(current.Spec.Selector.MatchLabels, desired.Spec.Selector.MatchLabels)
}
