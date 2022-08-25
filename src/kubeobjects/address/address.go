package address

type scalarType interface {
	bool | int | int64 | int32
}

func Of[T scalarType](i T) *T {
	return &i
}
