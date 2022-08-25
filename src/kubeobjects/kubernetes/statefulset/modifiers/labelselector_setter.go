package modifiers

import (
	internalTypes "github.com/Dynatrace/dynatrace-operator/src/kubeobjects/kubernetes/statefulset/internal/types"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type LabelSelectorSetter struct {
	LabelSelector *metav1.LabelSelector
}

var _ internalTypes.Modifier = (*LabelSelectorSetter)(nil)

func (s LabelSelectorSetter) Modify(sts *appsv1.StatefulSet) {
	sts.Spec.Selector = s.LabelSelector
}
