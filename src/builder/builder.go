package builder

import (
	"github.com/Dynatrace/dynatrace-operator/src/builder/api"
)

type Builder[T any] struct {
	data      *T
	modifiers []api.Modifier[T]
}

var _ api.Builder[any] = (*Builder[any])(nil)

func (b Builder[T]) Build() T {
	if b.data == nil {
		var data T
		b.data = &data
	}
	for _, m := range b.modifiers {
		if m.Enabled() {
			m.Modify(b.data)
		}
	}
	return *b.data
}

func (b *Builder[T]) AddModifier(modifiers ...api.Modifier[T]) api.Builder[T] {
	b.modifiers = append(b.modifiers, modifiers...)
	return b
}

func NewBuilder[T any](data T) Builder[T] {
	return Builder[T]{
		data:      &data,
		modifiers: []api.Modifier[T]{},
	}
}
