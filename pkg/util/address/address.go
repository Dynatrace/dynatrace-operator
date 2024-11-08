package address

func Of[T any](i T) *T {
	return &i
}
