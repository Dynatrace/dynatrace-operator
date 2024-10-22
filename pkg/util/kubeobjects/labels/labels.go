package labels

import (
	"reflect"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/version"
)

const (
	AppNameLabel      = "app.kubernetes.io/name"
	AppCreatedByLabel = "app.kubernetes.io/created-by"
	AppManagedByLabel = "app.kubernetes.io/managed-by"
	AppComponentLabel = "app.kubernetes.io/component"
	AppVersionLabel   = "app.kubernetes.io/version"

	OneAgentComponentLabel      = "oneagent"
	LogMonitoringComponentLabel = "logmonitoring"
	ActiveGateComponentLabel    = "activegate"
	WebhookComponentLabel       = "webhook"
	EdgeConnectComponentLabel   = "edgeconnect"
	ExtensionComponentLabel     = "dynatrace-extensions-controller"
	CollectorComponentLabel     = "dynatrace-extensions-collector"
)

type AppMatchLabels struct {
	Name      string
	CreatedBy string
	ManagedBy string
}

type coreMatchLabels struct {
	Name      string
	CreatedBy string
	Component string
}

type AppLabels struct {
	AppMatchLabels
	Component string
	Version   string
}

type CoreLabels struct {
	coreMatchLabels
	Version string
}

// NewAppLabels abstracts labels that are specific to an application managed by the operator
// which have their own version separate from the operator version.
// Follows the recommended label pattern: https://kubernetes.io/docs/concepts/overview/working-with-objects/common-labels
func NewAppLabels(appName, name, component, ver string) *AppLabels {
	return &AppLabels{
		AppMatchLabels: AppMatchLabels{
			Name:      appName,
			CreatedBy: name,
			ManagedBy: version.AppName,
		},
		Component: strings.ReplaceAll(component, "_", ""),
		Version:   ver,
	}
}

// NewCoreLabels abstracts labels that are used for statefulsetreconciler functionality in the operator
// which are not specific to an application's version
// Follows the recommended label pattern: https://kubernetes.io/docs/concepts/overview/working-with-objects/common-labels
func NewCoreLabels(dynakubeName, component string) *CoreLabels {
	return &CoreLabels{
		coreMatchLabels: coreMatchLabels{
			Name:      version.AppName,
			CreatedBy: dynakubeName,
			Component: component,
		},
		Version: version.Version,
	}
}

// BuildLabels creates labels that
// include operator version
func (labels *CoreLabels) BuildLabels() map[string]string {
	labelsMap := labels.BuildMatchLabels()
	labelsMap[AppVersionLabel] = labels.Version

	return labelsMap
}

// BuildLabels creates labels that
// include oneagent or activegate mode and version
func (labels *AppLabels) BuildLabels() map[string]string {
	labelsMap := labels.BuildMatchLabels()
	labelsMap[AppVersionLabel] = labels.Version
	labelsMap[AppComponentLabel] = labels.Component

	return labelsMap
}

// BuildMatchLabels creates labels that
// don't change when switching operator versions
func (labels *coreMatchLabels) BuildMatchLabels() map[string]string {
	return map[string]string{
		AppNameLabel:      labels.Name,
		AppCreatedByLabel: labels.CreatedBy,
		AppComponentLabel: labels.Component,
	}
}

// BuildMatchLabels creates labels that
// don't change when switching oneagent or activegate mode
func (labels *AppLabels) BuildMatchLabels() map[string]string {
	return map[string]string{
		AppNameLabel:      labels.Name,
		AppCreatedByLabel: labels.CreatedBy,
		AppManagedByLabel: labels.ManagedBy,
	}
}

func NotEqual(currentLabels, desiredLabels map[string]string) bool {
	return !reflect.DeepEqual(
		currentLabels,
		desiredLabels,
	)
}
