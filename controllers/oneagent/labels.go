package oneagent

// buildLabels returns generic labels based on the name given for a Dynatrace OneAgent
func buildLabels(name string, feature string) map[string]string {
	return map[string]string{
		"dynatrace.com/component":         "operator",
		"operator.dynatrace.com/instance": name,
		"operator.dynatrace.com/feature":  feature,
	}
}
