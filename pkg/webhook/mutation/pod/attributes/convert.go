package attributes

type convertFunc func(string, string) string

func convert(attributes map[string]string, c convertFunc) []string {
	converted := make([]string, 0)
	for key, value := range attributes {
		converted = append(converted, c(key, value))
	}
	return converted
}
