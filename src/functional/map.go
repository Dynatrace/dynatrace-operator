package functional

func Map[In any, Out any](arr []In, transformFn func(it In) Out) []Out {
	ret := make([]Out, len(arr))
	for i, it := range arr {
		ret[i] = transformFn(it)
	}
	return ret
}

func Filter[T any](arr []T, filterFn func(val T) bool) []T {
	ret := make([]T, 0)
	for _, val := range arr {
		if filterFn(val) {
			ret = append(ret, val)
		}
	}
	return ret
}
