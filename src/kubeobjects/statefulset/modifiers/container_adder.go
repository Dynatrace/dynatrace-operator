package modifiers

import (
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects/statefulset"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

type ContainerAdder struct {
	container corev1.Container
}

var _ statefulset.Modifier = (*ContainerAdder)(nil)

func (c ContainerAdder) Modify(sts *appsv1.StatefulSet) {
	if sts.Spec.Template.Spec.Containers == nil {
		sts.Spec.Template.Spec.Containers = make([]corev1.Container, 0)
	}
	sts.Spec.Template.Spec.Containers = append(sts.Spec.Template.Spec.Containers, c.container)
}
