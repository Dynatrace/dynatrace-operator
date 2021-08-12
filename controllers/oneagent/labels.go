package oneagent

func mergeLabels(labels ...map[string]string) map[string]string {
	res := map[string]string{}
	for _, m := range labels {
		for k, v := range m {
			res[k] = v
		}
	}

	return res
}

// buildLabels returns generic labels based on the name given for a Dynatrace OneAgent
func buildLabels(name string, feature string) map[string]string {
	return map[string]string{
		"dynatrace.com/component":         "operator",
		"operator.dynatrace.com/instance": name,
		"operator.dynatrace.com/feature":  feature,
	}
}
