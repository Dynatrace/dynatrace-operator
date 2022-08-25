package modifiers

import (
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects/kubernetes/statefulset/internal/types"
	appsv1 "k8s.io/api/apps/v1"
)

type Annotations = map[string]string

type AnnotationsSetter struct {
	Annotations Annotations
}

var _ types.Modifier = (*AnnotationsSetter)(nil)

func (s AnnotationsSetter) Modify(sts *appsv1.StatefulSet) {
	sts.ObjectMeta.Annotations = s.Annotations
}
