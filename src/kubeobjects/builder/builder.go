package builder

import (
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects/builder/api"
)

type Builder[T any] struct {
	data T
}

var _ api.Builder[any] = (*Builder[any])(nil)

func (b *Builder[T]) Build() T {
	return b.data
}

func (b *Builder[T]) AddModifier(modifiers ...api.Modifier[T]) api.Builder[T] {
	for _, m := range modifiers {
		m.Modify(&b.data)
	}
	return b
}
