package modifiers

import (
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects/kubernetes/objectmeta/internal/types"
	internalTypes "github.com/Dynatrace/dynatrace-operator/src/kubeobjects/kubernetes/types"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type AnnotationsAdder struct {
	Annotations internalTypes.Annotations
}

var _ types.Modifier = (*AnnotationsAdder)(nil)

func (s AnnotationsAdder) Modify(om *v1.ObjectMeta) {
	if om.Annotations == nil {
		om.Annotations = make(internalTypes.Annotations)
	}
	for k, v := range s.Annotations {
		om.Annotations[k] = v
	}
}
