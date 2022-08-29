package api

type Builder[T any] interface {
	Build() T
	AddModifier(...Modifier[T]) Builder[T]
}
