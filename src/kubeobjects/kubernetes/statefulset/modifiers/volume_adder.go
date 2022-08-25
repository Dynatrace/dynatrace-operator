package modifiers

import (
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects/kubernetes/statefulset/internal/types"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

type VolumeAdder struct {
	volume corev1.Volume
}

var _ types.Modifier = (*VolumeAdder)(nil)

func (c VolumeAdder) Modify(sts *appsv1.StatefulSet) {
	if sts.Spec.Template.Spec.Volumes == nil {
		sts.Spec.Template.Spec.Volumes = make([]corev1.Volume, 0)
	}
	sts.Spec.Template.Spec.Volumes = append(sts.Spec.Template.Spec.Volumes, c.volume)
}
