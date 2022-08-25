package modifiers

import (
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects/kubernetes/statefulset/internal/types"
	appsv1 "k8s.io/api/apps/v1"
)

type PodManagementPolicySetter struct {
	PodManagementPolicy appsv1.PodManagementPolicyType
}

var _ types.Modifier = (*PodManagementPolicySetter)(nil)

func (s PodManagementPolicySetter) Modify(sts *appsv1.StatefulSet) {
	sts.Spec.PodManagementPolicy = s.PodManagementPolicy
}
