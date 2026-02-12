package k8sservice

import (
	"slices"

	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/objects/internal/builder"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	// Mandatory fields, provided in constructor as named params
	setName      = builder.SetName[*corev1.Service]
	setNamespace = builder.SetNamespace[*corev1.Service]

	// Optional fields, provided in constructor as list of options
	SetLabels = builder.SetLabels[*corev1.Service]
)

func Build(owner metav1.Object, name string, selectorLabels map[string]string, svcPort []corev1.ServicePort, options ...builder.Option[*corev1.Service]) (*corev1.Service, error) {
	neededOpts := slices.Concat([]builder.Option[*corev1.Service]{
		setName(name),
		setPorts(svcPort),
		setSelectorLabels(selectorLabels),
		setNamespace(owner.GetNamespace()),
	}, options)

	return builder.Build(owner, &corev1.Service{}, neededOpts...)
}

func setPorts(svcPorts []corev1.ServicePort) builder.Option[*corev1.Service] {
	return func(s *corev1.Service) {
		s.Spec.Ports = svcPorts
	}
}

func setSelectorLabels(labels map[string]string) builder.Option[*corev1.Service] {
	return func(s *corev1.Service) {
		s.Spec.Selector = labels
	}
}

func SetType(serviceType corev1.ServiceType) builder.Option[*corev1.Service] {
	return func(s *corev1.Service) {
		s.Spec.Type = serviceType
	}
}
