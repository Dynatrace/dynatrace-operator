package modifiers

import (
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/activegate/internal/statefulset/agbuilder/internal/types"
	appsv1 "k8s.io/api/apps/v1"
)

type NoopModifier struct {
	Msg string
}

var _ types.Modifier = (*NoopModifier)(nil)

func (a NoopModifier) Modify(sts *appsv1.StatefulSet) {
	if sts.Annotations == nil {
		sts.Annotations = make(map[string]string)
	}
	sts.Annotations["noop"] = a.Msg
}
