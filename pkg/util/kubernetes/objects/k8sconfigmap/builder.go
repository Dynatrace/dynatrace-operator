package k8sconfigmap

import (
	"slices"

	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/objects/internal/builder"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	// Mandatory fields, provided in constructor as named params
	setName      = builder.SetName[*corev1.ConfigMap]
	setNamespace = builder.SetNamespace[*corev1.ConfigMap]

	// Optional fields, provided in constructor as list of options
	SetLabels = builder.SetLabels[*corev1.ConfigMap]
)

func Build(owner metav1.Object, name string, data map[string]string, options ...builder.Option[*corev1.ConfigMap]) (*corev1.ConfigMap, error) {
	neededOpts := slices.Concat([]builder.Option[*corev1.ConfigMap]{
		setName(name),
		setData(data),
		setNamespace(owner.GetNamespace()),
	}, options)

	return builder.Build(owner, &corev1.ConfigMap{}, neededOpts...)
}

func setData(data map[string]string) builder.Option[*corev1.ConfigMap] {
	return func(s *corev1.ConfigMap) {
		s.Data = data
	}
}
