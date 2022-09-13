package builder


type Builder[T any] interface {
	Build() T
	AddModifier(...Modifier[T]) Builder[T]
}

type Modifier[T any] interface {
	Enabled() bool
	Modify(*T)
}


type GenericBuilder[T any] struct {
	data      *T
	modifiers []Modifier[T]
}

var _ Builder[any] = (*GenericBuilder[any])(nil)

func (b GenericBuilder[T]) Build() T {
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

func (b *GenericBuilder[T]) AddModifier(modifiers ...Modifier[T]) Builder[T] {
	b.modifiers = append(b.modifiers, modifiers...)
	return b
}

func NewBuilder[T any](data T) GenericBuilder[T] {
	return GenericBuilder[T]{
		data:      &data,
		modifiers: []Modifier[T]{},
	}
}
