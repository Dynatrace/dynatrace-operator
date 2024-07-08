package configmap

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/internal/builder"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var setName = builder.SetName[*corev1.ConfigMap]
var SetNamespace = builder.SetNamespace[*corev1.ConfigMap]
var SetLabels = builder.SetLabels[*corev1.ConfigMap]

func Build(owner metav1.Object, name string, data map[string]string, options ...builder.Option[*corev1.ConfigMap]) (*corev1.ConfigMap, error) {
	neededOpts := []builder.Option[*corev1.ConfigMap]{
		setName(name),
		setData(data),
		SetNamespace(owner.GetNamespace()),
	}
	neededOpts = append(neededOpts, options...)

	return builder.Build(owner, &corev1.ConfigMap{}, neededOpts...)
}

func setData(data map[string]string) builder.Option[*corev1.ConfigMap] {
	return func(s *corev1.ConfigMap) {
		s.Data = data
	}
}
