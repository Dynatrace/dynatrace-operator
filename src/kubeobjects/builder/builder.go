package builder

import (
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects/builder/api"
)

type Builder[T any] struct {
	modifiers []api.Modifier[T]
	data      T
}

var _ api.Builder[any] = (*Builder[any])(nil)

func (b *Builder[T]) Build() T {
	for _, m := range b.modifiers {
		m.Modify(&b.data)
	}
	return b.data
}

func (b *Builder[T]) AddModifier(modifiers ...api.Modifier[T]) api.Builder[T] {
	b.modifiers = append(b.modifiers, modifiers...)
	return b
}
