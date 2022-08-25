package modifiers

import (
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects/kubernetes/podtemplatespec/internal/types"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ObjectMetaSetter struct {
	ObjectMeta v1.ObjectMeta
}

var _ types.Modifier = (*ObjectMetaSetter)(nil)

func (s ObjectMetaSetter) Modify(pts *corev1.PodTemplateSpec) {
	pts.ObjectMeta = s.ObjectMeta
}
