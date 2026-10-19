// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package k8slabel

import (
	"maps"

	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/Dynatrace/dynatrace-operator/pkg/version"
	"k8s.io/apimachinery/pkg/api/validate/content"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation"
)

var log = logd.Get().WithName("k8slabel")

const (
	AppName         = "app.kubernetes.io/name"
	AppInstance     = "app.kubernetes.io/instance"
	AppManagedBy    = "app.kubernetes.io/managed-by"
	AppVersion      = "app.kubernetes.io/version"
	OperatorVersion = "internal.dynatrace.com/operator-version"

	OneAgent            = "oneagent"
	CodeModule          = "codemodule"
	LogMonitoring       = "logmonitoring"
	KSPM                = "kspm"
	ActiveGate          = "activegate"
	KubeMon             = "kubemon"
	Webhook             = "webhook"
	EdgeConnect         = "edgeconnect"
	Extension           = "dynatrace-extension-controller"
	OTelCol             = "dynatrace-otel-collector"
	DatabaseSQLExecutor = "dynatrace-sql-extension-executor"
	NodeController      = "node-controller"
	Operator            = "operator"
	OTelTargetAllocator = "opentelemetry-target-allocator"
	OTelScraper         = "opentelemetry-collector"
	OTelGateway         = "opentelemetry-gateway"
)

type Labels struct {
	Name            string
	Instance        string
	ManagedBy       string
	Version         string
	OperatorVersion string
}

// New will return a simplified Labels struct that contains all the necessary info related to the ownership of a resource.
// If appVersion is empty the related `version` label will be omitted. This should be done for resources that have no version in general, like a `Secret`.
func New(appName, instanceName, appVersion string) *Labels {
	return &Labels{
		Name:            appName,
		Instance:        instanceName,
		ManagedBy:       version.AppName,
		Version:         sanitizeVersion(appVersion),
		OperatorVersion: truncateVersion(version.Version),
	}
}

// AsMap returns all labels, including version metadata
func (labels *Labels) AsMap() map[string]string {
	labelsMap := labels.AsSelector()

	if labels.Version != "" {
		labelsMap[AppVersion] = labels.Version
	}

	if labels.OperatorVersion != "" {
		labelsMap[OperatorVersion] = labels.OperatorVersion
	}

	return labelsMap
}

// MergeInto merges the labels from AsMap into obj, keeping any labels obj already
// has that are not part of this set.
func (labels *Labels) MergeInto(obj metav1.Object) {
	objLabels := obj.GetLabels()
	if objLabels == nil {
		objLabels = make(map[string]string)
	}

	maps.Copy(objLabels, labels.AsMap())
	obj.SetLabels(objLabels)
}

// AsSelector returns the stable labels used to select resources
func (labels *Labels) AsSelector() map[string]string {
	return map[string]string{
		AppName:      labels.Name,
		AppInstance:  labels.Instance,
		AppManagedBy: labels.ManagedBy,
	}
}

func NotEqual(currentLabels, desiredLabels map[string]string) bool {
	return !maps.Equal(currentLabels, desiredLabels)
}

func truncateVersion(ver string) string {
	if len(ver) > validation.DNS1035LabelMaxLength {
		ver = ver[:validation.DNS1035LabelMaxLength]
	}

	return ver
}

func sanitizeVersion(ver string) string {
	ver = truncateVersion(ver)

	if errs := content.IsLabelValue(ver); len(errs) > 0 {
		log.Info("omitting invalid app version label", "version", ver, "errors", errs)

		return ""
	}

	return ver
}
