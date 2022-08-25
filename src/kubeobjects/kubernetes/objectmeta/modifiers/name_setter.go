package modifiers

import (
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects/kubernetes/objectmeta/internal/types"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type NameSetter struct {
	Name string
}

var _ types.Modifier = (*NameSetter)(nil)

func (s NameSetter) Modify(om *v1.ObjectMeta) {
	om.Name = s.Name
}
