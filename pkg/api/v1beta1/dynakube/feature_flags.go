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

package dynakube

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/util/logger"
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

	// oneAgent.

	AnnotationFeatureMultipleOsAgentsOnNode         = AnnotationFeaturePrefix + "multiple-osagents-on-node"
	AnnotationFeatureOneAgentMaxUnavailable         = AnnotationFeaturePrefix + "oneagent-max-unavailable"
	AnnotationFeatureOneAgentIgnoreProxy            = AnnotationFeaturePrefix + "oneagent-ignore-proxy"
	AnnotationFeatureOneAgentInitialConnectRetry    = AnnotationFeaturePrefix + "oneagent-initial-connect-retry-ms"
	AnnotationFeatureRunOneAgentContainerPrivileged = AnnotationFeaturePrefix + "oneagent-privileged"
	AnnotationFeatureOneAgentSecCompProfile         = AnnotationFeaturePrefix + "oneagent-seccomp-profile"

	// injection (webhook).

	// Deprecated: AnnotationFeatureDisableMetadataEnrichment use AnnotationFeatureMetadataEnrichment instead.
	AnnotationFeatureDisableMetadataEnrichment = AnnotationFeaturePrefix + "disable-metadata-enrichment"

	AnnotationFeatureMetadataEnrichment = AnnotationFeaturePrefix + "metadata-enrichment"

	AnnotationFeatureIgnoreUnknownState    = AnnotationFeaturePrefix + "ignore-unknown-state"
	AnnotationFeatureIgnoredNamespaces     = AnnotationFeaturePrefix + "ignored-namespaces"
	AnnotationFeatureAutomaticInjection    = AnnotationFeaturePrefix + "automatic-injection"
	AnnotationFeatureLabelVersionDetection = AnnotationFeaturePrefix + "label-version-detection"
	AnnotationInjectionFailurePolicy       = AnnotationFeaturePrefix + "injection-failure-policy"
	AnnotationFeatureInitContainerSeccomp  = AnnotationFeaturePrefix + "init-container-seccomp-profile"
	AnnotationFeatureEnforcementMode       = AnnotationFeaturePrefix + "enforcement-mode"

	// CSI.
	AnnotationFeatureMaxFailedCsiMountAttempts = AnnotationFeaturePrefix + "max-csi-mount-attempts"
	AnnotationFeatureReadOnlyCsiVolume         = AnnotationFeaturePrefix + "injection-readonly-volume"

	falsePhrase  = "false"
	truePhrase   = "true"
	silentPhrase = "silent"
	failPhrase   = "fail"
)

const (
	DefaultMaxFailedCsiMountAttempts        = 10
	DefaultMinRequestThresholdMinutes       = 15
	IstioDefaultOneAgentInitialConnectRetry = 6000
)

var (
	log = logger.Get().WithName("dynakube-api")
)

func (dk *DynaKube) getDisableFlagWithDeprecatedAnnotation(annotation string, deprecatedAnnotation string) bool {
	return dk.getFeatureFlagRaw(annotation) == falsePhrase ||
		dk.getFeatureFlagRaw(deprecatedAnnotation) == truePhrase && dk.getFeatureFlagRaw(annotation) == ""
}

func (dk *DynaKube) getFeatureFlagRaw(annotation string) string {
	if raw, ok := dk.Annotations[annotation]; ok {
		return raw
	}

	return ""
}

func (dk *DynaKube) getFeatureFlagInt(annotation string, defaultVal int) int {
	raw := dk.getFeatureFlagRaw(annotation)
	if raw == "" {
		return defaultVal
	}

	val, err := strconv.Atoi(raw)
	if err != nil {
		return defaultVal
	}

	return val
}

// FeatureDisableActiveGateUpdates is a feature flag to disable ActiveGate updates.
func (dk *DynaKube) FeatureDisableActiveGateUpdates() bool {
	return dk.getDisableFlagWithDeprecatedAnnotation(AnnotationFeatureActiveGateUpdates, AnnotationFeatureDisableActiveGateUpdates)
}

// FeatureNoProxy is a feature flag to set the NO_PROXY value to be used by the dtClient.
func (dk *DynaKube) FeatureNoProxy() string {
	return dk.getFeatureFlagRaw(AnnotationFeatureNoProxy)
}

func (dk *DynaKube) FeatureApiRequestThreshold() time.Duration {
	interval := dk.getFeatureFlagInt(AnnotationFeatureApiRequestThreshold, DefaultMinRequestThresholdMinutes)
	if interval < 0 {
		interval = DefaultMinRequestThresholdMinutes
	}

	return time.Duration(interval) * time.Minute
}

// FeatureOneAgentMaxUnavailable is a feature flag to configure maxUnavailable on the OneAgent DaemonSets rolling upgrades.
func (dk *DynaKube) FeatureOneAgentMaxUnavailable() int {
	return dk.getFeatureFlagInt(AnnotationFeatureOneAgentMaxUnavailable, 1)
}

// FeatureIgnoreUnknownState is a feature flag that makes the operator inject into applications even when the dynakube is in an UNKNOWN state,
// this may cause extra host to appear in the tenant for each process.
func (dk *DynaKube) FeatureIgnoreUnknownState() bool {
	return dk.getFeatureFlagRaw(AnnotationFeatureIgnoreUnknownState) == truePhrase
}

// FeatureIgnoredNamespaces is a feature flag for ignoring certain namespaces.
// defaults to "[ \"^dynatrace$\", \"^kube-.*\", \"openshift(-.*)?\" ]".
func (dk *DynaKube) FeatureIgnoredNamespaces() []string {
	raw := dk.getFeatureFlagRaw(AnnotationFeatureIgnoredNamespaces)
	if raw == "" {
		return dk.getDefaultIgnoredNamespaces()
	}

	ignoredNamespaces := &[]string{}

	err := json.Unmarshal([]byte(raw), ignoredNamespaces)
	if err != nil {
		log.Error(err, "failed to unmarshal ignoredNamespaces feature-flag")

		return dk.getDefaultIgnoredNamespaces()
	}

	return *ignoredNamespaces
}

func (dk *DynaKube) getDefaultIgnoredNamespaces() []string {
	defaultIgnoredNamespaces := []string{
		fmt.Sprintf("^%s$", dk.Namespace),
		"^kube-.*",
		"^openshift(-.*)?",
		"^gke-.*",
		"^gmp-.*",
	}

	return defaultIgnoredNamespaces
}

// FeatureAutomaticKubernetesApiMonitoring is a feature flag to enable automatic kubernetes api monitoring,
// which ensures that settings for this kubernetes cluster exist in Dynatrace.
func (dk *DynaKube) FeatureAutomaticKubernetesApiMonitoring() bool {
	return dk.getFeatureFlagRaw(AnnotationFeatureAutomaticK8sApiMonitoring) != falsePhrase
}

// FeatureAutomaticKubernetesApiMonitoringClusterName is a feature flag to set custom cluster name for automatic-kubernetes-api-monitoring.
func (dk *DynaKube) FeatureAutomaticKubernetesApiMonitoringClusterName() string {
	return dk.getFeatureFlagRaw(AnnotationFeatureAutomaticK8sApiMonitoringClusterName)
}

// FeatureEnableK8sAppEnabled is a feature flag to enable automatically enable current Kubernetes cluster for the Kubernetes app.
func (dk *DynaKube) FeatureEnableK8sAppEnabled() bool {
	return dk.getFeatureFlagRaw(AnnotationFeatureK8sAppEnabled) == truePhrase
}

// FeatureDisableMetadataEnrichment is a feature flag to disable metadata enrichment,.
func (dk *DynaKube) FeatureDisableMetadataEnrichment() bool {
	return dk.getDisableFlagWithDeprecatedAnnotation(AnnotationFeatureMetadataEnrichment, AnnotationFeatureDisableMetadataEnrichment)
}

// FeatureAutomaticInjection controls OneAgent is injected to pods in selected namespaces automatically ("automatic-injection=true" or flag not set)
// or if pods need to be opted-in one by one ("automatic-injection=false").
func (dk *DynaKube) FeatureAutomaticInjection() bool {
	return dk.getFeatureFlagRaw(AnnotationFeatureAutomaticInjection) != falsePhrase
}

// FeatureEnableMultipleOsAgentsOnNode is a feature flag to enable multiple osagents running on the same host.
func (dk *DynaKube) FeatureEnableMultipleOsAgentsOnNode() bool {
	return dk.getFeatureFlagRaw(AnnotationFeatureMultipleOsAgentsOnNode) == truePhrase
}

// FeatureActiveGateAppArmor is a feature flag to enable AppArmor in ActiveGate container.
func (dk *DynaKube) FeatureActiveGateAppArmor() bool {
	return dk.getFeatureFlagRaw(AnnotationFeatureActiveGateAppArmor) == truePhrase
}

// FeatureOneAgentIgnoreProxy is a feature flag to ignore the proxy for oneAgents when set in CR.
func (dk *DynaKube) FeatureOneAgentIgnoreProxy() bool {
	return dk.getFeatureFlagRaw(AnnotationFeatureOneAgentIgnoreProxy) == truePhrase
}

// FeatureActiveGateIgnoreProxy is a feature flag to ignore the proxy for ActiveGate when set in CR.
func (dk *DynaKube) FeatureActiveGateIgnoreProxy() bool {
	return dk.getFeatureFlagRaw(AnnotationFeatureActiveGateIgnoreProxy) == truePhrase
}

// FeatureLabelVersionDetection is a feature flag to enable injecting additional environment variables based on user labels.
func (dk *DynaKube) FeatureLabelVersionDetection() bool {
	return dk.getFeatureFlagRaw(AnnotationFeatureLabelVersionDetection) == truePhrase
}

// FeatureAgentInitialConnectRetry is a feature flag to configure startup delay of standalone agents.
func (dk *DynaKube) FeatureAgentInitialConnectRetry() int {
	defaultValue := -1
	ffValue := dk.getFeatureFlagInt(AnnotationFeatureOneAgentInitialConnectRetry, defaultValue)

	// In case of istio, we want to have a longer initial delay for codemodules to ensure the DT service is created consistently
	if ffValue == defaultValue && dk.Spec.EnableIstio {
		ffValue = IstioDefaultOneAgentInitialConnectRetry
	}

	return ffValue
}

func (dk *DynaKube) FeatureOneAgentPrivileged() bool {
	return dk.getFeatureFlagRaw(AnnotationFeatureRunOneAgentContainerPrivileged) == truePhrase
}

func (dk *DynaKube) FeatureOneAgentSecCompProfile() string {
	return dk.getFeatureFlagRaw(AnnotationFeatureOneAgentSecCompProfile)
}

func (dk *DynaKube) FeatureMaxFailedCsiMountAttempts() int {
	maxCsiMountAttemptsValue := dk.getFeatureFlagInt(AnnotationFeatureMaxFailedCsiMountAttempts, DefaultMaxFailedCsiMountAttempts)
	if maxCsiMountAttemptsValue < 0 {
		return DefaultMaxFailedCsiMountAttempts
	}

	return maxCsiMountAttemptsValue
}

func (dk *DynaKube) FeatureReadOnlyCsiVolume() bool {
	return dk.getFeatureFlagRaw(AnnotationFeatureReadOnlyCsiVolume) == truePhrase
}

func (dk *DynaKube) FeatureInjectionFailurePolicy() string {
	if dk.getFeatureFlagRaw(AnnotationInjectionFailurePolicy) == failPhrase {
		return failPhrase
	}

	return silentPhrase
}

func (dk *DynaKube) FeaturePublicRegistry() bool {
	return dk.getFeatureFlagRaw(AnnotationFeaturePublicRegistry) == truePhrase
}

func (dk *DynaKube) FeatureInitContainerSeccomp() bool {
	return dk.getFeatureFlagRaw(AnnotationFeatureInitContainerSeccomp) == truePhrase
}

// FeatureEnforcementMode is a feature flag to control how the initContainer
// sets the tenantUUID to the container.conf file (always vs if oneAgent is present).
func (dk *DynaKube) FeatureEnforcementMode() bool {
	return dk.getFeatureFlagRaw(AnnotationFeatureEnforcementMode) != falsePhrase
}
