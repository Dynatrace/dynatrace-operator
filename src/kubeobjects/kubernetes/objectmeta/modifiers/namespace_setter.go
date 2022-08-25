package modifiers

import (
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects/kubernetes/objectmeta/internal/types"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type NamespaceSetter struct {
	Namespace string
}

var _ types.Modifier = (*NamespaceSetter)(nil)

func (s NamespaceSetter) Modify(om *v1.ObjectMeta) {
	om.Namespace = s.Namespace
}
