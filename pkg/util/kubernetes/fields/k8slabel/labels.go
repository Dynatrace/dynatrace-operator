// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package k8slabel

import (
	"maps"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/version"
	"k8s.io/apimachinery/pkg/api/validate/content"
	"k8s.io/apimachinery/pkg/util/validation"
)

const (
	AppNameLabel     = "app.kubernetes.io/name"
	AppInstanceLabel = "app.kubernetes.io/instance"
	// Deprecated: Use AppInstanceLabel instead. See https://kubernetes.io/docs/reference/labels-annotations-taints/#app-kubernetes-io-created-by-deprecated
	AppCreatedByLabel    = "app.kubernetes.io/created-by"
	AppManagedByLabel    = "app.kubernetes.io/managed-by"
	AppComponentLabel    = "app.kubernetes.io/component"
	AppVersionLabel      = "app.kubernetes.io/version"
	OperatorVersionLabel = "internal.dynatrace.com/operator-version"

	OneAgentComponentLabel      = "oneagent"
	CodeModuleComponentLabel    = "codemodule"
	LogMonitoringComponentLabel = "logmonitoring"
	KSPMComponentLabel          = "kspm"
	ActiveGateComponentLabel    = "activegate"
	KubeMonComponentLabel       = "kubemon"
	WebhookComponentLabel       = "webhook"
	EdgeConnectComponentLabel   = "edgeconnect"
	ExtensionComponentLabel     = "dynatrace-extension-controller"
	OTelColComponentLabel       = "dynatrace-otel-collector"
	DatabaseSQLExecutorLabel    = "dynatrace-sql-extension-executor"
	NodeControllerLabel         = "node-controller"
	OperatorComponentLabel      = "operator"
)

const (
	KubeMonAppLabel = "kubemon"
)

func OTelTargetAllocator() *Labels {
	return New("opentelemetry-target-allocator", "otel-allocator", "")
}

func OTelScraper() *Labels {
	return New("opentelemetry-collector", "otel-scraper", "")
}

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

type Labels struct {
	Name            string
	Instance        string
	ManagedBy       string
	Version         string
	OperatorVersion string
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

// New will return a simplified Labels struct that contains all the necessary info related to the ownership of a resource.
// It should be used instead of NewAppLabels and NewCoreLabels, as those are overcomplicating things.
// If appVersion is empty the related `version` label will be omitted. This should be done for resources that have no version in general, like a `Secret`.
func New(appName, instanceName, appVersion string) *Labels {
	return &Labels{
		Name:            appName,
		Instance:        instanceName,
		ManagedBy:       version.AppName,
		Version:         truncateVersion(appVersion),
		OperatorVersion: truncateVersion(version.Version),
	}
}

// NewAppLabels abstracts labels that are specific to an application managed by the operator
// which have their own version separate from the operator version.
// Follows the recommended label pattern: https://kubernetes.io/docs/concepts/overview/working-with-objects/common-labels
//
// Deprecated: Use New instead.
func NewAppLabels(appName, name, component, ver string) *AppLabels {
	if len(ver) > validation.DNS1035LabelMaxLength {
		ver = ver[:validation.DNS1035LabelMaxLength]
	}

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
//
// Deprecated: Use New instead.
func NewCoreLabels(dynakubeName, component string) *CoreLabels {
	ver := version.Version
	if len(ver) > validation.DNS1035LabelMaxLength {
		ver = ver[:validation.DNS1035LabelMaxLength]
	}

	return &CoreLabels{
		coreMatchLabels: coreMatchLabels{
			Name:      version.AppName,
			CreatedBy: dynakubeName,
			Component: component,
		},
		Version: ver,
	}
}

// AsMap returns all labels, including version metadata
func (labels *Labels) AsMap() map[string]string {
	labelsMap := labels.AsSelector()

	if labels.Version != "" {
		labelsMap[AppVersionLabel] = labels.Version
	}

	if labels.OperatorVersion != "" {
		labelsMap[OperatorVersionLabel] = labels.OperatorVersion
	}

	return labelsMap
}

// AsSelector returns the stable labels used to select resources
func (labels *Labels) AsSelector() map[string]string {
	return map[string]string{
		AppNameLabel:      labels.Name,
		AppInstanceLabel:  labels.Instance,
		AppManagedByLabel: labels.ManagedBy,
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
	return !maps.Equal(currentLabels, desiredLabels)
}

func truncateVersion(ver string) string {
	if len(ver) > validation.DNS1035LabelMaxLength {
		ver = ver[:validation.DNS1035LabelMaxLength]
	}

	if len(content.IsLabelValue(ver)) > 0 {
		return ""
	}

	return ver
}
