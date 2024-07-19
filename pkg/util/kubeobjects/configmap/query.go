package configmap

import (
	"reflect"

	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/internal/query"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func Query(kubeClient client.Client, kubeReader client.Reader, log logd.Logger) query.Generic[*corev1.ConfigMap, *corev1.ConfigMapList] {
	return query.Generic[*corev1.ConfigMap, *corev1.ConfigMapList]{
		Target:     &corev1.ConfigMap{},
		ListTarget: &corev1.ConfigMapList{},
		ToList: func(cml *corev1.ConfigMapList) []*corev1.ConfigMap {
			out := []*corev1.ConfigMap{}
			for _, cm := range cml.Items {
				out = append(out, &cm)
			}

			return out
		},
		IsEqual:      isEqual,
		MustRecreate: func(_, _ *corev1.ConfigMap) bool { return false },

		KubeClient: kubeClient,
		KubeReader: kubeReader,
		Log:        log,
	}
}

func isEqual(configMap *corev1.ConfigMap, other *corev1.ConfigMap) bool {
	return reflect.DeepEqual(configMap.Data, other.Data) && reflect.DeepEqual(configMap.Labels, other.Labels) && reflect.DeepEqual(configMap.OwnerReferences, other.OwnerReferences)
}
