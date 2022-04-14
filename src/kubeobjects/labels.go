package kubeobjects

import (
	"reflect"

	"github.com/Dynatrace/dynatrace-operator/src/version"
)

const (
	AppNameLabel          = "app.kubernetes.io/name"
	AppCreatedByLabel     = "app.kubernetes.io/created-by"
	AppComponentLabel     = "app.kubernetes.io/component"
	AppVersionLabel       = "app.kubernetes.io/version"
	ComponentFeatureLabel = "app.kubernetes.io/component-feature"
	ComponentVersionLabel = "app.kubernetes.io/component-version"

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

func NewMatchLabels(createdBy string, component ComponentLabelValue) *MatchLabels {
	return &MatchLabels{
		AppName:      version.AppName,
		AppCreatedBy: createdBy,
		AppComponent: component,
	}
}

func NewComponentLabels(createdBy string, component ComponentLabelValue, componentFeature string, componentVersion string) *PodLabels {
	return &PodLabels{
		MatchLabels:      *NewMatchLabels(createdBy, component),
		AppVersion:       version.Version,
		ComponentFeature: componentFeature,
		ComponentVersion: componentVersion,
	}
}

func NewPodLabels(createdBy string, component ComponentLabelValue) *PodLabels {
	return NewComponentLabels(createdBy, component, "", "")
}

// BuildLabels produces a set of labels that
// include versions of operator and component
// and component feature, if set
func (labels *PodLabels) BuildLabels() map[string]string {
	labelsMap := labels.BuildMatchLabels()
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
	return map[string]string{
		AppNameLabel:      labels.AppName,
		AppCreatedByLabel: labels.AppCreatedBy,
		AppComponentLabel: string(labels.AppComponent),
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

func LabelsNotEqual(currentLabels, desiredLabels map[string]string) bool {
	return !reflect.DeepEqual(
		currentLabels,
		desiredLabels,
	)
}
