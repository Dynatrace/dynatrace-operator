package api

type Modifier[T any] interface {
	Modify(*T)
}
