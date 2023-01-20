package address

func Of[A any](value A) *A {
	return &value
}
