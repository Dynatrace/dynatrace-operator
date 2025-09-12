package maputil

import (
	"maps"
	"strconv"
)

func GetField(values map[string]string, key string, defaultValue string) string {
	if x := values[key]; x != "" {
		return x
	}

	return defaultValue
}

func GetFieldBool(values map[string]string, key string, defaultValue bool) bool {
	if x := values[key]; x != "" {
		parsed, err := strconv.ParseBool(x)
		if err == nil {
			return parsed
		}
	}

	return defaultValue
}

func MergeMap(inputs ...map[string]string) map[string]string {
	res := map[string]string{}

	for _, m := range inputs {
		maps.Copy(res, m)
	}

	return res
}
