package builder

import (
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects/builder/api"
)

type Builder[T any] struct {
	modifiers []api.Modifier[T]
}

var _ api.Builder[any] = (*Builder[any])(nil)

func (b Builder[T]) Build() T {
	var data T
	for _, m := range b.modifiers {
		m.Modify(&data)
	}
	return data
}

func (b *Builder[T]) AddModifier(modifiers ...api.Modifier[T]) api.Builder[T] {
	b.modifiers = append(b.modifiers, modifiers...)
	return b
}
