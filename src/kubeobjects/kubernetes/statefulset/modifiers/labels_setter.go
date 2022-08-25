package modifiers

import (
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects/kubernetes/statefulset/internal/types"
	appsv1 "k8s.io/api/apps/v1"
)

type Labels = map[string]string

type LabelsSetter struct {
	Labels Labels
}

var _ types.Modifier = (*LabelsSetter)(nil)

func (s LabelsSetter) Modify(sts *appsv1.StatefulSet) {
	sts.ObjectMeta.Labels = s.Labels
}
