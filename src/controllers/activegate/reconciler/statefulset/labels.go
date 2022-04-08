package statefulset

import (
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
)

const (
	ActiveGateComponentName = "activegate"
)

func (stsProperties *statefulSetProperties) buildLabels() map[string]string {
	labels := kubeobjects.MergeLabels(stsProperties.DynaKube.Labels,
		BuildLabelsFromInstance(stsProperties.DynaKube, stsProperties.feature),
		stsProperties.CapabilityProperties.Labels)
	return labels
}

// buildMatchLabels produces a set of labels that
// don't change when switching between oneagent modes
// or during operator version update
// as matchLabels are not mutable on a StatefulSet
func (stsProperties *statefulSetProperties) buildMatchLabels() map[string]string {
	labels := kubeobjects.CommonLabels(stsProperties.DynaKube.Name, ActiveGateComponentName)
	delete(labels, kubeobjects.AppVersionLabel)
	delete(labels, kubeobjects.FeatureLabel)
	return labels
}

func BuildLabelsFromInstance(instance *dynatracev1beta1.DynaKube, feature string) map[string]string {
	labels := kubeobjects.CommonLabels(instance.Name, ActiveGateComponentName)
	labels[kubeobjects.FeatureLabel] = feature
	labels[kubeobjects.AppVersionLabel] = instance.Status.ActiveGate.Version
	return labels
}
