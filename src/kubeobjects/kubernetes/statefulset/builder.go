package statefulset

import (
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects/builder"
	appsv1 "k8s.io/api/apps/v1"
)

type Modifier = builder.Modifier[appsv1.StatefulSet]

type Builder struct {
	sts appsv1.StatefulSet
}

var _ builder.Builder[appsv1.StatefulSet] = (*Builder)(nil)

func (s *Builder) Build() appsv1.StatefulSet {
	return s.sts
}

func (s *Builder) AddModifier(modifiers ...Modifier) builder.Builder[appsv1.StatefulSet] {
	for _, m := range modifiers {
		m.Modify(&s.sts)
	}
	return s
}
