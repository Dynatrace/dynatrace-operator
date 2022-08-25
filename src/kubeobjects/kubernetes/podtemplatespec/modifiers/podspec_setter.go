package modifiers

import (
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects/kubernetes/podtemplatespec/internal/types"
	corev1 "k8s.io/api/core/v1"
)

type PodSpecSetter struct {
	PodSpec corev1.PodSpec
}

var _ types.Modifier = (*PodSpecSetter)(nil)

func (s PodSpecSetter) Modify(pts *corev1.PodTemplateSpec) {
	pts.Spec = s.PodSpec
}
