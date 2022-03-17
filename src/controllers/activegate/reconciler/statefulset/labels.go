package statefulset

import (
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
)

const (
	ActiveGateComponentName = "activegate"
)

func buildLabels(instance *dynatracev1beta1.DynaKube, feature string, capabilityProperties *dynatracev1beta1.CapabilityProperties) map[string]string {
	return kubeobjects.MergeLabels(instance.Labels,
		BuildLabelsFromInstance(instance, feature),
		capabilityProperties.Labels)
}

func BuildLabelsFromInstance(instance *dynatracev1beta1.DynaKube, feature string) map[string]string {
	labels := kubeobjects.CommonLabels(instance.Name, ActiveGateComponentName)
	labels[kubeobjects.FeatureLabel] = feature
	return labels
}
