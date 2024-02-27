/*
Copyright 2021 Dynatrace LLC.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package activegate

import (
	"strconv"
	"time"
)

const (
	AnnotationFeaturePrefix = "feature.dynatrace.com/"

	// General.
	AnnotationFeaturePublicRegistry = AnnotationFeaturePrefix + "public-registry"

	// activeGate.

	// Deprecated: AnnotationFeatureDisableActiveGateUpdates use AnnotationFeatureActiveGateUpdates instead.
	AnnotationFeatureDisableActiveGateUpdates = AnnotationFeaturePrefix + "disable-activegate-updates"

	AnnotationFeatureActiveGateUpdates = AnnotationFeaturePrefix + "activegate-updates"

	AnnotationFeatureActiveGateAppArmor                   = AnnotationFeaturePrefix + "activegate-apparmor"
	AnnotationFeatureAutomaticK8sApiMonitoring            = AnnotationFeaturePrefix + "automatic-kubernetes-api-monitoring"
	AnnotationFeatureAutomaticK8sApiMonitoringClusterName = AnnotationFeaturePrefix + "automatic-kubernetes-api-monitoring-cluster-name"
	AnnotationFeatureK8sAppEnabled                        = AnnotationFeaturePrefix + "k8s-app-enabled"
	AnnotationFeatureActiveGateIgnoreProxy                = AnnotationFeaturePrefix + "activegate-ignore-proxy"

	// dtClient.

	AnnotationFeatureNoProxy             = AnnotationFeaturePrefix + "no-proxy"
	AnnotationFeatureApiRequestThreshold = AnnotationFeaturePrefix + "dynatrace-api-request-threshold"

	falsePhrase = "false"
	truePhrase  = "true"
)

const (
	DefaultMinRequestThresholdMinutes = 15
)

// FeatureDisableActiveGateUpdates is a feature flag to disable ActiveGate updates.
func (activeGate *ActiveGate) FeatureDisableActiveGateUpdates() bool {
	return activeGate.getDisableFlagWithDeprecatedAnnotation(AnnotationFeatureActiveGateUpdates, AnnotationFeatureDisableActiveGateUpdates)
}

// FeatureAutomaticKubernetesApiMonitoring is a feature flag to enable automatic kubernetes api monitoring,
// which ensures that settings for this kubernetes cluster exist in Dynatrace.
func (activeGate *ActiveGate) FeatureAutomaticKubernetesApiMonitoring() bool {
	return activeGate.getFeatureFlagRaw(AnnotationFeatureAutomaticK8sApiMonitoring) != falsePhrase
}

// FeatureAutomaticKubernetesApiMonitoringClusterName is a feature flag to set custom cluster name for automatic-kubernetes-api-monitoring.
func (activeGate *ActiveGate) FeatureAutomaticKubernetesApiMonitoringClusterName() string {
	return activeGate.getFeatureFlagRaw(AnnotationFeatureAutomaticK8sApiMonitoringClusterName)
}

// FeatureEnableK8sAppEnabled is a feature flag to enable automatically enable current Kubernetes cluster for the Kubernetes app.
func (activeGate *ActiveGate) FeatureEnableK8sAppEnabled() bool {
	return activeGate.getFeatureFlagRaw(AnnotationFeatureK8sAppEnabled) == truePhrase
}

// FeatureActiveGateAppArmor is a feature flag to enable AppArmor in ActiveGate container.
func (activeGate *ActiveGate) FeatureActiveGateAppArmor() bool {
	return activeGate.getFeatureFlagRaw(AnnotationFeatureActiveGateAppArmor) == truePhrase
}

// FeatureActiveGateIgnoreProxy is a feature flag to ignore the proxy for ActiveGate when set in CR.
func (activeGate *ActiveGate) FeatureActiveGateIgnoreProxy() bool {
	return activeGate.getFeatureFlagRaw(AnnotationFeatureActiveGateIgnoreProxy) == truePhrase
}

func (activeGate *ActiveGate) FeaturePublicRegistry() bool {
	return activeGate.getFeatureFlagRaw(AnnotationFeaturePublicRegistry) == truePhrase
}

// dtClient

// FeatureNoProxy is a feature flag to set the NO_PROXY value to be used by the dtClient.
func (activeGate *ActiveGate) FeatureNoProxy() string {
	return activeGate.getFeatureFlagRaw(AnnotationFeatureNoProxy)
}

func (activeGate *ActiveGate) FeatureApiRequestThreshold() time.Duration {
	interval := activeGate.getFeatureFlagInt(AnnotationFeatureApiRequestThreshold, DefaultMinRequestThresholdMinutes)
	if interval < 0 {
		interval = DefaultMinRequestThresholdMinutes
	}

	return time.Duration(interval) * time.Minute
}

func (activeGate *ActiveGate) getDisableFlagWithDeprecatedAnnotation(annotation string, deprecatedAnnotation string) bool {
	return activeGate.getFeatureFlagRaw(annotation) == falsePhrase ||
		activeGate.getFeatureFlagRaw(deprecatedAnnotation) == truePhrase && activeGate.getFeatureFlagRaw(annotation) == ""
}

func (activeGate *ActiveGate) getFeatureFlagRaw(annotation string) string {
	if raw, ok := activeGate.Annotations[annotation]; ok {
		return raw
	}

	return ""
}

func (activeGate *ActiveGate) getFeatureFlagInt(annotation string, defaultVal int) int {
	raw := activeGate.getFeatureFlagRaw(annotation)
	if raw == "" {
		return defaultVal
	}

	val, err := strconv.Atoi(raw)
	if err != nil {
		return defaultVal
	}

	return val
}
