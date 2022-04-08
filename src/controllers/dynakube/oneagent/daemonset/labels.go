package daemonset

import (
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
)

func BuildLabels(instance *dynatracev1beta1.DynaKube, feature string) map[string]string {
	labels := kubeobjects.CommonLabels(instance.Name, kubeobjects.OneAgentComponentLabel)
	labels[kubeobjects.FeatureLabel] = feature
	labels[kubeobjects.AppVersionLabel] = instance.Status.OneAgent.Version
	return labels
}

// buildMatchLabels produces a set of labels that
// don't change when switching between oneagent modes
// or during operator version update
// as matchLabels are not mutable on a Daemonset
func (dsInfo *builderInfo) buildMatchLabels() map[string]string {
	labels := BuildLabels(dsInfo.instance, "")
	delete(labels, kubeobjects.AppVersionLabel)
	delete(labels, kubeobjects.FeatureLabel)
	return labels
}
