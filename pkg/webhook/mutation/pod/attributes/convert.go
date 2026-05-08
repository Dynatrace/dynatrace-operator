package attributes

type convertFunc func(string, string) string

func convert(attributes map[string]string, c convertFunc) []string {
	converted := make([]string, 0, len(attributes))
	for key, value := range attributes {
		if result := c(key, value); result != "" {
			converted = append(converted, result)
		}
	}

	return converted
}
