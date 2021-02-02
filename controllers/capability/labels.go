package capability

import "github.com/Dynatrace/dynatrace-operator/api/v1alpha1"

const (
	KeyDynatrace    = "dynatrace"
	ValueActiveGate = "activegate"
	KeyActiveGate   = "activegate"
)

func BuildLabels(instance *v1alpha1.DynaKube, capabilityProperties *v1alpha1.CapabilityProperties) map[string]string {
	return mergeLabels(instance.Labels,
		BuildLabelsFromInstance(instance),
		capabilityProperties.Labels)
}

func BuildLabelsFromInstance(instance *v1alpha1.DynaKube) map[string]string {
	return map[string]string{
		KeyDynatrace:  ValueActiveGate,
		KeyActiveGate: instance.Name,
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
