package daemonset

import "github.com/Dynatrace/dynatrace-operator/src/kubeobjects"

const (
	componentName = "oneagent"
)

// buildLabels returns generic labels based on the name given for a Dynatrace OneAgent
func BuildLabels(name string, feature string) map[string]string {
	labels := kubeobjects.CommonLabels(name, componentName)
	labels[kubeobjects.FeatureLabel] = feature
	return labels
}
