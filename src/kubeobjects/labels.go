package kubeobjects

import (
	"reflect"
)

const (
	AppNameLabel          = "app.kubernetes.io/name"
	AppCreatedByLabel     = "app.kubernetes.io/created-by"
	AppComponentLabel     = "app.kubernetes.io/component"
	AppVersionLabel       = "app.kubernetes.io/version"
	ComponentFeatureLabel = "component.dynatrace.com/feature"
	ComponentVersionLabel = "component.dynatrace.com/version"

	ActiveGateComponentLabel ComponentLabelValue = "activegate"
	OperatorComponentLabel   ComponentLabelValue = "operator"
	OneAgentComponentLabel   ComponentLabelValue = "oneagent"
	WebhookComponentLabel    ComponentLabelValue = "webhook"
)

type ComponentLabelValue string

type MatchLabels struct {
	AppName      string
	AppCreatedBy string
	AppComponent ComponentLabelValue
}

type PodLabels struct {
	MatchLabels
	AppVersion       string
	ComponentFeature string
	ComponentVersion string
}

func (labels *MatchLabels) buildCommonLabels() map[string]string {
	return map[string]string{
		AppNameLabel:      labels.AppName,
		AppCreatedByLabel: labels.AppCreatedBy,
		AppComponentLabel: string(labels.AppComponent),
	}
}

func (labels *PodLabels) BuildLabels() map[string]string {
	labelsMap := labels.buildCommonLabels()
	if labels.AppVersion != "" {
		labelsMap[AppVersionLabel] = labels.AppVersion
	}
	if labels.ComponentFeature != "" {
		labelsMap[ComponentFeatureLabel] = labels.ComponentFeature
	}
	if labels.ComponentVersion != "" {
		labelsMap[ComponentVersionLabel] = labels.ComponentVersion
	}
	return labelsMap
}

// BuildMatchLabels produces a set of labels that
// don't change when switching between modes
// or during operator version update
// as matchLabels are immutable
func (labels *MatchLabels) BuildMatchLabels() map[string]string {
	return labels.buildCommonLabels()
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

func LabelsNotEqual(currentLabels, desiredLabels map[string]string) bool {
	return !reflect.DeepEqual(
		currentLabels,
		desiredLabels,
	)
}
