package modifiers

import (
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects/kubernetes/statefulset/internal/types"
	appsv1 "k8s.io/api/apps/v1"
)

type GenericModifier struct {
	ModifierFunc func(set *appsv1.StatefulSet)
}

var _ types.Modifier = (*GenericModifier)(nil)

func (s GenericModifier) Modify(sts *appsv1.StatefulSet) {
	s.ModifierFunc(sts)
}
