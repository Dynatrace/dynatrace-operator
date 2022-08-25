package modifiers

import (
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects/kubernetes/objectmeta/internal/types"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Labels = map[string]string

type LabelsSetter struct {
	Labels Labels
}

var _ types.Modifier = (*LabelsSetter)(nil)

func (s LabelsSetter) Modify(om *v1.ObjectMeta) {
	om.Labels = s.Labels
}
