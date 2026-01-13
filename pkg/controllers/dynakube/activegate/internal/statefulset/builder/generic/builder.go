package generic

type Builder[T any] interface {
	Build() (T, error)
	AddModifier(...Modifier[T]) Builder[T]
}

type Modifier[T any] interface {
	Enabled() bool
	Modify(*T) error
}

// noinspection GoNameStartsWithPackageName
type GenericBuilder[T any] struct { //nolint: revive
	data      *T
	modifiers []Modifier[T]
}

var _ Builder[any] = (*GenericBuilder[any])(nil)

func (b GenericBuilder[T]) Build() (T, error) {
	var data T
	if b.data == nil {
		b.data = &data
	}

	for _, m := range b.modifiers {
		if m.Enabled() {
			err := m.Modify(b.data)
			if err != nil {
				return *b.data, err
			}
		}
	}

	return *b.data, nil
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
