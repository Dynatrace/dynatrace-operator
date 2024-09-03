package statefulset

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/internal/builder"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	// Mandatory fields, provided in constructor as named params
	setName      = builder.SetName[*appsv1.StatefulSet]
	setNamespace = builder.SetNamespace[*appsv1.StatefulSet]

	// Optional fields, provided in constructor as list of options
	SetLabels = builder.SetLabels[*appsv1.StatefulSet]
)

func Build(owner metav1.Object, name string, container corev1.Container, options ...builder.Option[*appsv1.StatefulSet]) (*appsv1.StatefulSet, error) {
	neededOpts := []builder.Option[*appsv1.StatefulSet]{
		setName(name),
		SetContainer(container),
		setNamespace(owner.GetNamespace()),
	}
	neededOpts = append(neededOpts, options...)

	return builder.Build(owner, &appsv1.StatefulSet{}, neededOpts...)
}

func SetContainer(container corev1.Container) builder.Option[*appsv1.StatefulSet] {
	return func(s *appsv1.StatefulSet) {
		targetIndex := 0
		for index := range s.Spec.Template.Spec.Containers {
			if s.Spec.Template.Spec.Containers[targetIndex].Name == container.Name {
				targetIndex = index

				break
			}
		}

		if targetIndex == 0 {
			s.Spec.Template.Spec.Containers = make([]corev1.Container, 1)
		}

		s.Spec.Template.Spec.Containers[targetIndex] = container
	}
}
