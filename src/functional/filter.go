package functional

func Filter[T any](arr []T, filterFn func(val T) bool) []T {
	ret := make([]T, 0)
	for _, val := range arr {
		if filterFn(val) {
			ret = append(ret, val)
		}
	}
	return ret
}
