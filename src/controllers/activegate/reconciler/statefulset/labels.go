package statefulset

import (
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
)

const (
	KeyDynatrace    = "dynatrace.com/component"
	KeyActiveGate   = "operator.dynatrace.com/instance"
	KeyFeature      = "operator.dynatrace.com/feature"
	ValueActiveGate = "activegate"
)

func buildLabels(instance *dynatracev1beta1.DynaKube, feature string, capabilityProperties *dynatracev1beta1.CapabilityProperties) map[string]string {
	return mergeLabels(instance.Labels,
		BuildLabelsFromInstance(instance, feature),
		capabilityProperties.Labels)
}

func BuildLabelsFromInstance(instance *dynatracev1beta1.DynaKube, feature string) map[string]string {
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
