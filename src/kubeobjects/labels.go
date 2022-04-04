package kubeobjects

import (
	"reflect"

	"github.com/Dynatrace/dynatrace-operator/src/version"
)

const (
	AppNameLabel      = "app.kubernetes.io/name"
	AppCreatedByLabel = "app.kubernetes.io/created-by"
	AppComponentLabel = "app.kubernetes.io/component"
	AppVersionLabel   = "app.kubernetes.io/version"
	FeatureLabel      = "component.dynatrace.com/feature"

	ActiveGateComponentLabel ComponentLabelValue = "activegate"
	OperatorComponentLabel   ComponentLabelValue = "operator"
	OneAgentComponentLabel   ComponentLabelValue = "oneagent"
	WebhookComponentLabel    ComponentLabelValue = "webhook"
)

type ComponentLabelValue string

func MergeLabels(labels ...map[string]string) map[string]string {
	res := map[string]string{}
	for _, m := range labels {
		for k, v := range m {
			res[k] = v
		}
	}

	return res
}

func CommonLabels(dynakubeName string, componentName ComponentLabelValue) map[string]string {
	return map[string]string{
		AppNameLabel:      version.AppName,
		AppComponentLabel: string(componentName),
		AppCreatedByLabel: dynakubeName,
		AppVersionLabel:   version.Version,
	}
}

func MatchLabelsChanged(currentMatchLabels, desiredMatchLabels map[string]string) bool {
	return !reflect.DeepEqual(
		currentMatchLabels,
		desiredMatchLabels,
	)
}
