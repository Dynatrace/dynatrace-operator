package builder

type Modifier[T any] interface {
	Modify(*T)
}

type Builder[T any] interface {
	Build() T
	AddModifier(...Modifier[T]) Builder[T]
}
