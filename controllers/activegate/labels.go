package activegate

import "github.com/Dynatrace/dynatrace-operator/api/v1alpha1"

const (
	KeyDynatrace    = "dynatrace.com/component"
	KeyActiveGate   = "operator.dynatrace.com/instance"
	KeyFeature      = "operator.dynatrace.com/feature"
	ValueActiveGate = "activegate"
)

func BuildLabels(instance *v1alpha1.DynaKube, feature string, capabilityProperties *v1alpha1.CapabilityProperties) map[string]string {
	return MergeLabels(instance.Labels,
		BuildLabelsFromInstance(instance, feature),
		capabilityProperties.Labels)
}

func BuildLabelsFromInstance(instance *v1alpha1.DynaKube, feature string) map[string]string {
	return map[string]string{
		KeyDynatrace:  ValueActiveGate,
		KeyActiveGate: instance.Name,
		KeyFeature:    feature,
	}
}

func MergeLabels(labels ...map[string]string) map[string]string {
	res := map[string]string{}
	for _, m := range labels {
		for k, v := range m {
			res[k] = v
		}
	}

	return res
}
