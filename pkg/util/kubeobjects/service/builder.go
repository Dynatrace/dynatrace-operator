package service

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/internal/builder"
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

func Build(owner metav1.Object, name string, selectorLabels map[string]string, svcPort corev1.ServicePort, options ...builder.Option[*corev1.Service]) (*corev1.Service, error) {
	neededOpts := []builder.Option[*corev1.Service]{
		setName(name),
		setPorts(svcPort),
		setSelectorLabels(selectorLabels),
		setNamespace(owner.GetNamespace()),
	}
	neededOpts = append(neededOpts, options...)

	return builder.Build(owner, &corev1.Service{}, neededOpts...)
}

func setPorts(svcPort corev1.ServicePort) builder.Option[*corev1.Service] {
	return func(s *corev1.Service) {
		targetIndex := 0
		for index := range s.Spec.Ports {
			if s.Spec.Ports[targetIndex].Name == svcPort.Name {
				targetIndex = index

				break
			}
		}

		if targetIndex == 0 {
			s.Spec.Ports = make([]corev1.ServicePort, 1)
		}

		s.Spec.Ports[targetIndex].Name = svcPort.Name
		s.Spec.Ports[targetIndex].Port = svcPort.Port
		s.Spec.Ports[targetIndex].Protocol = svcPort.Protocol
		s.Spec.Ports[targetIndex].TargetPort = svcPort.TargetPort
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
