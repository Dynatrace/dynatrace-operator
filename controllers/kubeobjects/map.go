package kubeobjects

import "strings"

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
		// i18n ignored on purpose :)
		trues := []string{"true", "yes", "1"}
		falses := []string{"false", "no", "0"}

		isInIgnoreCase := func(s string, d []string) bool {
			for _, word := range d {
				if strings.EqualFold(s, word) {
					return true
				}
			}
			return false
		}

		if isInIgnoreCase(x, trues) {
			return true
		}
		if isInIgnoreCase(x, falses) {
			return false
		}
	}
	return defaultValue
}
