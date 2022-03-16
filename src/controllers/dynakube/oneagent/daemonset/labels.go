package daemonset

import "github.com/Dynatrace/dynatrace-operator/src/kubeobjects"

// buildLabels returns generic labels based on the name given for a Dynatrace OneAgent
func BuildLabels(name string, feature string) map[string]string {
	labels := kubeobjects.CommonLabels(name, kubeobjects.OneAgentComponentLabel)
	labels[kubeobjects.FeatureLabel] = feature
	return labels
}
