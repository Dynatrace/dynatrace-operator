package k8sservice

import (
	"reflect"

	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8slabel"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/objects/internal/query"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func Query(kubeClient client.Client, kubeReader client.Reader, log logd.Logger) query.Generic[*corev1.Service, *corev1.ServiceList] {
	return query.Generic[*corev1.Service, *corev1.ServiceList]{
		Target:     &corev1.Service{},
		ListTarget: &corev1.ServiceList{},
		ToList: func(list *corev1.ServiceList) []*corev1.Service {
			out := make([]*corev1.Service, len(list.Items))
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

func isEqual(current, other *corev1.Service) bool {
	return reflect.DeepEqual(current.Spec.Ports, other.Spec.Ports) && reflect.DeepEqual(current.Labels, other.Labels) && reflect.DeepEqual(current.OwnerReferences, other.OwnerReferences) && reflect.DeepEqual(current.Spec.Selector, other.Spec.Selector)
}

func mustRecreate(current, desired *corev1.Service) bool {
	return k8slabel.NotEqual(current.Spec.Selector, desired.Spec.Selector)
}
