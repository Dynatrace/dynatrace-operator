package supportarchive

func Map[In any, Out any](arr []In, transformFn func(it In) Out) []Out {
	ret := make([]Out, len(arr))

	for i, it := range arr {
		ret[i] = transformFn(it)
	}

	return ret
}
