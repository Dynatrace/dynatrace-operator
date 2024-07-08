package secret

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/internal/builder"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var setName = builder.SetName[*corev1.Secret]
var SetNamespace = builder.SetNamespace[*corev1.Secret]
var SetLabels = builder.SetLabels[*corev1.Secret]

func Build(owner metav1.Object, name string, data map[string][]byte, options ...builder.Option[*corev1.Secret]) (*corev1.Secret, error) {
	neededOpts := []builder.Option[*corev1.Secret]{
		setName(name),
		setData(data),
		SetNamespace(owner.GetNamespace()),
	}
	neededOpts = append(neededOpts, options...)

	return builder.Build(owner, &corev1.Secret{}, neededOpts...)
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
