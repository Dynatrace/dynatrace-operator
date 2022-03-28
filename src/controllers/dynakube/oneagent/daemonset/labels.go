package daemonset

import "github.com/Dynatrace/dynatrace-operator/src/kubeobjects"

func BuildLabels(name string, feature string) map[string]string {
	labels := kubeobjects.CommonLabels(name, kubeobjects.OneAgentComponentLabel)
	labels[kubeobjects.FeatureLabel] = feature
	return labels
}
