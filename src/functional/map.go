package functional

func Map[In any, Out any](arr []In, fn func(it In) Out) []Out {
	var newArray = []Out{}
	for _, it := range arr {
		newArray = append(newArray, fn(it))
	}
	return newArray
}
