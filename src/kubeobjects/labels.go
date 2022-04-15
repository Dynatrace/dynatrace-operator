package kubeobjects

import (
	"reflect"

	"github.com/Dynatrace/dynatrace-operator/src/version"
)

const (
	AppNameLabel      = "app.kubernetes.io/name"
	AppCreatedByLabel = "app.kubernetes.io/created-by"
	AppManagedByLabel = "app.kubernetes.io/managed-by"
	AppComponentLabel = "app.kubernetes.io/component"
	AppVersionLabel   = "app.kubernetes.io/version"

	OneAgentComponentLabel   = "oneagent"
	ActiveGateComponentLabel = "activegate"
	WebhookComponentLabel    = "webhook"
)

type AppFeatureLabel string

type matchLabels struct {
	AppName      string
	AppCreatedBy string
	AppComponent string
}

type VersionedLabels struct {
	matchLabels
	AppVersion   string
	AppManagedBy string
}

func newMatchLabels(name, createdBy, component string) *matchLabels {
	return &matchLabels{
		AppName:      name,
		AppCreatedBy: createdBy,
		AppComponent: component,
	}
}

// NewAppLabels abstracts labels that are specific to an application managed by the operator
// which have their own version separate from the operator version.
// Follows the recommended label pattern: https://kubernetes.io/docs/concepts/overview/working-with-objects/common-labels
func NewAppLabels(name, createdBy, appVersion, feature string) *VersionedLabels {
	return &VersionedLabels{
		matchLabels:  *newMatchLabels(name, createdBy, feature),
		AppVersion:   appVersion,
		AppManagedBy: version.AppName,
	}
}

// NewCoreLabels abstracts labels that are used for core functionality in the operator
// which are not specific to an application's version
// Follows the recommended label pattern: https://kubernetes.io/docs/concepts/overview/working-with-objects/common-labels
func NewCoreLabels(createdBy, feature string) *VersionedLabels {
	return NewAppLabels(version.AppName, createdBy, version.Version, feature)
}

// BuildLabels produces a set of labels that
// include versions of operator and component
// and component feature, if set
func (labels *VersionedLabels) BuildLabels() map[string]string {
	labelsMap := labels.BuildMatchLabels()
	if labels.AppVersion != "" {
		labelsMap[AppVersionLabel] = labels.AppVersion
	}
	if labels.AppManagedBy != "" {
		labelsMap[AppManagedByLabel] = labels.AppManagedBy
	}
	return labelsMap
}

// BuildMatchLabels produces a set of labels that
// don't change when switching between modes
// or during operator version update
// as matchLabels are immutable
func (labels *matchLabels) BuildMatchLabels() map[string]string {
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
