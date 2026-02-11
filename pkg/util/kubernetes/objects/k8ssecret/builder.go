package k8ssecret

import (
	"slices"

	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/objects/internal/builder"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	// Mandatory fields, provided in constructor as named params
	setName      = builder.SetName[*corev1.Secret]
	setNamespace = builder.SetNamespace[*corev1.Secret]

	// Optional fields, provided in constructor as list of options
	SetLabels = builder.SetLabels[*corev1.Secret]
)

func Build(owner metav1.Object, name string, data map[string][]byte, options ...builder.Option[*corev1.Secret]) (*corev1.Secret, error) {
	neededOpts := slices.Concat([]builder.Option[*corev1.Secret]{
		setName(name),
		setData(data),
		setNamespace(owner.GetNamespace()),
	}, options)

	return builder.Build(owner, &corev1.Secret{}, neededOpts...)
}

func BuildForNamespace(name, namespace string, data map[string][]byte, options ...builder.Option[*corev1.Secret]) (*corev1.Secret, error) {
	neededOpts := slices.Concat([]builder.Option[*corev1.Secret]{
		setName(name),
		setData(data),
		setNamespace(namespace),
	}, options)

	return builder.Build(nil, &corev1.Secret{}, neededOpts...)
}

func setData(data map[string][]byte) builder.Option[*corev1.Secret] {
	return func(s *corev1.Secret) {
		s.Data = data
	}
}

func SetType(secretType corev1.SecretType) builder.Option[*corev1.Secret] {
	return func(s *corev1.Secret) {
		s.Type = secretType
	}
}
