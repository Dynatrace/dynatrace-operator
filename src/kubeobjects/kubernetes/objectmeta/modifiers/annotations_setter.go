package modifiers

import (
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects/kubernetes/objectmeta/internal/types"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Annotations = map[string]string

type AnnotationsSetter struct {
	Annotations Annotations
}

var _ types.Modifier = (*AnnotationsSetter)(nil)

func (s AnnotationsSetter) Modify(om *v1.ObjectMeta) {
	om.Annotations = s.Annotations
}
