package modifiers

import (
	internalTypes "github.com/Dynatrace/dynatrace-operator/src/kubeobjects/kubernetes/statefulset/internal/types"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
)

type PodTemplateSpecSetter struct {
	PodTemplateSpec v1.PodTemplateSpec
}

var _ internalTypes.Modifier = (*PodTemplateSpecSetter)(nil)

func (s PodTemplateSpecSetter) Modify(sts *appsv1.StatefulSet) {
	sts.Spec.Template = s.PodTemplateSpec
}
