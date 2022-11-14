package functional

func Map[In any, Out any](arr []In, transformFn func(it In) Out) []Out {
	var ret = []Out{}
	for _, it := range arr {
		ret = append(ret, transformFn(it))
	}

	return ret
}
