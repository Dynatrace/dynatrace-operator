package modifiers

import (
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects/kubernetes/statefulset/internal/types"
	internalTypes "github.com/Dynatrace/dynatrace-operator/src/kubeobjects/kubernetes/types"
	appsv1 "k8s.io/api/apps/v1"
)

type AnnotationsAdder struct {
	Annotations internalTypes.Annotations
}

var _ types.Modifier = (*AnnotationsAdder)(nil)

func (s AnnotationsAdder) Modify(om *appsv1.StatefulSet) {
	if om.Annotations == nil {
		om.Annotations = make(internalTypes.Annotations)
	}
	for k, v := range s.Annotations {
		om.Annotations[k] = v
	}
}
