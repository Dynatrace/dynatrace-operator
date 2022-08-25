package modifiers

import (
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects/kubernetes/objectmeta/internal/types"
	types2 "github.com/Dynatrace/dynatrace-operator/src/kubeobjects/kubernetes/types"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type LabelsSetter struct {
	Labels types2.Labels
}

var _ types.Modifier = (*LabelsSetter)(nil)

func (s LabelsSetter) Modify(om *v1.ObjectMeta) {
	om.Labels = s.Labels
}
