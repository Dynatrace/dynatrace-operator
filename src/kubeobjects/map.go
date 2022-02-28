package kubeobjects

import (
	"reflect"
	"strconv"

	corev1 "k8s.io/api/core/v1"
)

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

func ConfigMapDataEqual(map1, map2 *corev1.ConfigMap) bool {
	if map1 == nil || map2 == nil {
		return map1 == nil && map2 == nil
	}

	return reflect.DeepEqual(map1.Data, map2.Data) &&
		reflect.DeepEqual(map1.BinaryData, map2.BinaryData)
}
