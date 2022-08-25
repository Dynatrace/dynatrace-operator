package modifiers

import (
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects/kubernetes/objectmeta/internal/types"
	internalTypes "github.com/Dynatrace/dynatrace-operator/src/kubeobjects/kubernetes/types"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type AnnotationsSetter struct {
	Annotations internalTypes.Annotations
}

var _ types.Modifier = (*AnnotationsSetter)(nil)

func (s AnnotationsSetter) Modify(om *v1.ObjectMeta) {
	om.Annotations = s.Annotations
}
