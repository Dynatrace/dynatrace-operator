package modifiers

import (
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects/kubernetes/statefulset/internal/types"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ObjectMetaSetter struct {
	ObjectMeta v1.ObjectMeta
}

var _ types.Modifier = (*ObjectMetaSetter)(nil)

func (s ObjectMetaSetter) Modify(sts *appsv1.StatefulSet) {
	sts.ObjectMeta = s.ObjectMeta
}
