package statefulset

import (
	dynatracev1 "github.com/Dynatrace/dynatrace-operator/api/v1"
	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
)

const (
	KeyDynatrace    = "dynatrace.com/component"
	KeyActiveGate   = "operator.dynatrace.com/instance"
	KeyFeature      = "operator.dynatrace.com/feature"
	ValueActiveGate = "activegate"
)

func buildLabels(instance *dynatracev1.DynaKube, feature string, capabilityProperties *dynatracev1alpha1.CapabilityProperties) map[string]string {
	return mergeLabels(instance.Labels,
		BuildLabelsFromInstance(instance, feature),
		capabilityProperties.Labels)
}

func BuildLabelsFromInstance(instance *dynatracev1.DynaKube, feature string) map[string]string {
	return map[string]string{
		KeyDynatrace:  ValueActiveGate,
		KeyActiveGate: instance.Name,
		KeyFeature:    feature,
	}
}

func mergeLabels(labels ...map[string]string) map[string]string {
	res := map[string]string{}
	for _, m := range labels {
		for k, v := range m {
			res[k] = v
		}
	}

	return res
}
