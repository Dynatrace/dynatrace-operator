package modifiers

import (
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects/kubernetes/statefulset/internal/types"
	appsv1 "k8s.io/api/apps/v1"
)

type ReplicasSetter struct {
	Replicas *int32
}

var _ types.Modifier = (*ReplicasSetter)(nil)

func (s ReplicasSetter) Modify(sts *appsv1.StatefulSet) {
	sts.Spec.Replicas = s.Replicas
}
