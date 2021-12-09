package kubeobjects

import "strconv"

func GetField(values map[string]string, key string, defaultValue string) string {
	if values == nil {
		return defaultValue
	}
	if x := values[key]; x != "" {
		return x
	}
	return defaultValue
}

func GetFieldBool(values map[string]string, key string, defaultValue bool) bool {
	if values == nil {
		return defaultValue
	}
	if x := values[key]; x != "" {
		parsed, err := strconv.ParseBool(x)
		if err == nil {
			return parsed
		}
	}
	return defaultValue
}
