package address_of

type scalarType interface {
	bool | int | int64
}

func Scalar[T scalarType](i T) *T {
	return &i
}
