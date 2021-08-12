package kubeobjects

func GetField(values map[string]string, key, defaultValue string) string {
	if values == nil {
		return defaultValue
	}
	if x := values[key]; x != "" {
		return x
	}
	return defaultValue
}
