package modifiers

import (
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects/kubernetes/statefulset/internal/types"
	appsv1 "k8s.io/api/apps/v1"
)

type NamespaceSetter struct {
	Namespace string
}

var _ types.Modifier = (*NamespaceSetter)(nil)

func (s NamespaceSetter) Modify(sts *appsv1.StatefulSet) {
	sts.ObjectMeta.Namespace = s.Namespace
}
