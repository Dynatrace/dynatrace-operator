package modifiers

import (
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects/kubernetes/statefulset/internal/types"
	appsv1 "k8s.io/api/apps/v1"
)

type NameSetter struct {
	Name string
}

var _ types.Modifier = (*NameSetter)(nil)

func (s NameSetter) Modify(sts *appsv1.StatefulSet) {
	sts.ObjectMeta.Name = s.Name
}
