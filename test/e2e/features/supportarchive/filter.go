package supportarchive

func filter[T any](arr []T, predicate func(val T) bool) []T {
	var ret []T

	for _, val := range arr {
		if predicate(val) {
			ret = append(ret, val)
		}
	}

	return ret
}
