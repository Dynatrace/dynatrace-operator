package functional

func Filter[T any](arr []T, predicate func(val T) bool) []T {
	ret := make([]T, 0)

	for _, val := range arr {
		if predicate(val) {
			ret = append(ret, val)
		}
	}

	return ret
}
