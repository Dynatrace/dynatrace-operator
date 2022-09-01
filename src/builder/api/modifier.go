package api

type Modifier[T any] interface {
	Enabled() bool
	Modify(*T)
}
