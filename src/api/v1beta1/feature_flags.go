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

package v1beta1

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/Dynatrace/dynatrace-operator/src/logger"
)

const (
	DeprecatedFeatureFlagPrefix = "alpha.operator.dynatrace.com/feature-"

	AnnotationFeaturePrefix = "feature.dynatrace.com/"

	// General
	AnnotationFeaturePublicRegistry = AnnotationFeaturePrefix + "public-registry"

	// activeGate

	// Deprecated: AnnotationFeatureDisableActiveGateUpdates use AnnotationFeatureActiveGateUpdates instead
	AnnotationFeatureDisableActiveGateUpdates = AnnotationFeaturePrefix + "disable-activegate-updates"
	// Deprecated: AnnotationFeatureDisableActiveGateRawImage use AnnotationFeatureActiveGateRawImage instead
	AnnotationFeatureDisableActiveGateRawImage = AnnotationFeaturePrefix + "disable-activegate-raw-image"

	AnnotationFeatureActiveGateUpdates   = AnnotationFeaturePrefix + "activegate-updates"
	AnnotationFeatureActiveGateRawImage  = AnnotationFeaturePrefix + "activegate-raw-image"
	AnnotationFeatureActiveGateAuthToken = AnnotationFeaturePrefix + "activegate-authtoken"

	AnnotationFeatureActiveGateAppArmor                   = AnnotationFeaturePrefix + "activegate-apparmor"
	AnnotationFeatureActiveGateReadOnlyFilesystem         = AnnotationFeaturePrefix + "activegate-readonly-fs"
	AnnotationFeatureAutomaticK8sApiMonitoring            = AnnotationFeaturePrefix + "automatic-kubernetes-api-monitoring"
	AnnotationFeatureAutomaticK8sApiMonitoringClusterName = AnnotationFeaturePrefix + "automatic-kubernetes-api-monitoring-cluster-name"
	AnnotationFeatureActiveGateIgnoreProxy                = AnnotationFeaturePrefix + "activegate-ignore-proxy"

	AnnotationFeatureCustomSyntheticImage = AnnotationFeaturePrefix + "custom-synthetic-image"

	// dtClient

	// Deprecated: AnnotationFeatureDisableHostsRequests use AnnotationFeatureHostsRequests instead
	AnnotationFeatureDisableHostsRequests = AnnotationFeaturePrefix + "disable-hosts-requests"
	AnnotationFeatureHostsRequests        = AnnotationFeaturePrefix + "hosts-requests"
	AnnotationFeatureNoProxy              = AnnotationFeaturePrefix + "no-proxy"
	AnnotationFeatureApiRequestThreshold  = AnnotationFeaturePrefix + "dynatrace-api-request-threshold"

	// oneAgent

	// Deprecated: AnnotationFeatureDisableReadOnlyOneAgent use AnnotationFeatureReadOnlyOneAgent instead
	AnnotationFeatureDisableReadOnlyOneAgent = AnnotationFeaturePrefix + "disable-oneagent-readonly-host-fs"

	AnnotationFeatureReadOnlyOneAgent = AnnotationFeaturePrefix + "oneagent-readonly-host-fs"

	AnnotationFeatureMultipleOsAgentsOnNode         = AnnotationFeaturePrefix + "multiple-osagents-on-node"
	AnnotationFeatureOneAgentMaxUnavailable         = AnnotationFeaturePrefix + "oneagent-max-unavailable"
	AnnotationFeatureOneAgentIgnoreProxy            = AnnotationFeaturePrefix + "oneagent-ignore-proxy"
	AnnotationFeatureOneAgentInitialConnectRetry    = AnnotationFeaturePrefix + "oneagent-initial-connect-retry-ms"
	AnnotationFeatureRunOneAgentContainerPrivileged = AnnotationFeaturePrefix + "oneagent-privileged"
	AnnotationFeatureOneAgentSecCompProfile         = AnnotationFeaturePrefix + "oneagent-seccomp-profile"

	// injection (webhook)

	// Deprecated: AnnotationFeatureDisableWebhookReinvocationPolicy use AnnotationFeatureWebhookReinvocationPolicy instead
	AnnotationFeatureDisableWebhookReinvocationPolicy = AnnotationFeaturePrefix + "disable-webhook-reinvocation-policy"
	// Deprecated: AnnotationFeatureDisableMetadataEnrichment use AnnotationFeatureMetadataEnrichment instead
	AnnotationFeatureDisableMetadataEnrichment = AnnotationFeaturePrefix + "disable-metadata-enrichment"

	AnnotationFeatureWebhookReinvocationPolicy = AnnotationFeaturePrefix + "webhook-reinvocation-policy"
	AnnotationFeatureMetadataEnrichment        = AnnotationFeaturePrefix + "metadata-enrichment"

	AnnotationFeatureIgnoreUnknownState    = AnnotationFeaturePrefix + "ignore-unknown-state"
	AnnotationFeatureIgnoredNamespaces     = AnnotationFeaturePrefix + "ignored-namespaces"
	AnnotationFeatureAutomaticInjection    = AnnotationFeaturePrefix + "automatic-injection"
	AnnotationFeatureLabelVersionDetection = AnnotationFeaturePrefix + "label-version-detection"
	AnnotationInjectionFailurePolicy       = AnnotationFeaturePrefix + "injection-failure-policy"
	AnnotationFeatureInitContainerSeccomp  = AnnotationFeaturePrefix + "init-container-seccomp-profile"

	// CSI
	AnnotationFeatureMaxFailedCsiMountAttempts = AnnotationFeaturePrefix + "max-csi-mount-attempts"
	AnnotationFeatureReadOnlyCsiVolume         = AnnotationFeaturePrefix + "injection-readonly-volume"

	// synthetic location
	AnnotationFeatureSyntheticLocationEntityId = AnnotationFeaturePrefix + "synthetic-location-entity-id"

	// synthetic node type
	AnnotationFeatureSyntheticNodeType = AnnotationFeaturePrefix + "synthetic-node-type"

	// replicas for the synthetic monitoring
	AnnotationFeatureSyntheticReplicas = AnnotationFeaturePrefix + "synthetic-replicas"

	falsePhrase  = "false"
	truePhrase   = "true"
	silentPhrase = "silent"
	failPhrase   = "fail"

	// synthetic node types
	SyntheticNodeXs = "XS"
	SyntheticNodeS  = "S"
	SyntheticNodeM  = "M"
)

const (
	DefaultMaxFailedCsiMountAttempts  = 10
	DefaultMinRequestThresholdMinutes = 15
)

var (
	log = logger.Factory.GetLogger("dynakube-api")

	defaultSyntheticReplicas = int32(1)
)

func (dk *DynaKube) getDisableFlagWithDeprecatedAnnotation(annotation string, deprecatedAnnotation string) bool {
	return dk.getFeatureFlagRaw(annotation) == falsePhrase ||
		dk.getFeatureFlagRaw(deprecatedAnnotation) == truePhrase && dk.getFeatureFlagRaw(annotation) == ""
}

func (dk *DynaKube) getFeatureFlagRaw(annotation string) string {
	if raw, ok := dk.Annotations[annotation]; ok {
		return raw
	}
	split := strings.Split(annotation, "/")
	postFix := split[1]
	if raw, ok := dk.Annotations[DeprecatedFeatureFlagPrefix+postFix]; ok {
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

// FeatureDisableHostsRequests is a feature flag to disable queries to the Hosts API.
func (dk *DynaKube) FeatureDisableHostsRequests() bool {
	return dk.getDisableFlagWithDeprecatedAnnotation(AnnotationFeatureHostsRequests, AnnotationFeatureDisableHostsRequests)
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

// FeatureDisableWebhookReinvocationPolicy disables the reinvocation for the Operator's webhooks.
// This disables instrumenting containers injected by other webhooks following the admission to the Operator's webhook.
func (dk *DynaKube) FeatureDisableWebhookReinvocationPolicy() bool {
	return dk.getDisableFlagWithDeprecatedAnnotation(AnnotationFeatureWebhookReinvocationPolicy, AnnotationFeatureDisableWebhookReinvocationPolicy)
}

// FeatureIgnoreUnknownState is a feature flag that makes the operator inject into applications even when the dynakube is in an UNKNOWN state,
// this may cause extra host to appear in the tenant for each process.
func (dk *DynaKube) FeatureIgnoreUnknownState() bool {
	return dk.getFeatureFlagRaw(AnnotationFeatureIgnoreUnknownState) == truePhrase
}

// FeatureIgnoredNamespaces is a feature flag for ignoring certain namespaces.
// defaults to "[ \"^dynatrace$\", \"^kube-.*\", \"openshift(-.*)?\" ]"
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
	}
	return defaultIgnoredNamespaces
}

// FeatureAutomaticKubernetesApiMonitoring is a feature flag to enable automatic kubernetes api monitoring,
// which ensures that settings for this kubernetes cluster exist in Dynatrace
func (dk *DynaKube) FeatureAutomaticKubernetesApiMonitoring() bool {
	return dk.getFeatureFlagRaw(AnnotationFeatureAutomaticK8sApiMonitoring) != falsePhrase
}

// FeatureAutomaticKubernetesApiMonitoringClusterName is a feature flag to set custom cluster name for automatic-kubernetes-api-monitoring
func (dk *DynaKube) FeatureAutomaticKubernetesApiMonitoringClusterName() string {
	return dk.getFeatureFlagRaw(AnnotationFeatureAutomaticK8sApiMonitoringClusterName)
}

// FeatureDisableMetadataEnrichment is a feature flag to disable metadata enrichment,
func (dk *DynaKube) FeatureDisableMetadataEnrichment() bool {
	return dk.getDisableFlagWithDeprecatedAnnotation(AnnotationFeatureMetadataEnrichment, AnnotationFeatureDisableMetadataEnrichment)
}

// FeatureAutomaticInjection controls OneAgent is injected to pods in selected namespaces automatically ("automatic-injection=true" or flag not set)
// or if pods need to be opted-in one by one ("automatic-injection=false")
func (dk *DynaKube) FeatureAutomaticInjection() bool {
	return dk.getFeatureFlagRaw(AnnotationFeatureAutomaticInjection) != falsePhrase
}

func (dk *DynaKube) FeatureCustomSyntheticImage() string {
	return dk.getFeatureFlagRaw(AnnotationFeatureCustomSyntheticImage)
}

// FeatureDisableReadOnlyOneAgent is a feature flag to specify if the operator needs to deploy the oneagents in a readonly mode,
// where the csi-driver would provide the volume for logs and such
// Defaults to false
func (dk *DynaKube) FeatureDisableReadOnlyOneAgent() bool {
	return dk.getDisableFlagWithDeprecatedAnnotation(AnnotationFeatureReadOnlyOneAgent, AnnotationFeatureDisableReadOnlyOneAgent)
}

// FeatureDisableActivegateRawImage is a feature flag to specify if the operator should
// fetch from cluster and set in ActiveGet container: tenant UUID, token and communication endpoints
// instead of using embedded ones in the image
// Defaults to false
func (dk *DynaKube) FeatureDisableActivegateRawImage() bool {
	return dk.getDisableFlagWithDeprecatedAnnotation(AnnotationFeatureActiveGateRawImage, AnnotationFeatureDisableActiveGateRawImage)
}

// FeatureEnableMultipleOsAgentsOnNode is a feature flag to enable multiple osagents running on the same host
func (dk *DynaKube) FeatureEnableMultipleOsAgentsOnNode() bool {
	return dk.getFeatureFlagRaw(AnnotationFeatureMultipleOsAgentsOnNode) == truePhrase
}

// FeatureActiveGateReadOnlyFilesystem is a feature flag to enable RO mounted filesystem in ActiveGate container
func (dk *DynaKube) FeatureActiveGateReadOnlyFilesystem() bool {
	return dk.getFeatureFlagRaw(AnnotationFeatureActiveGateReadOnlyFilesystem) != falsePhrase
}

// FeatureActiveGateAppArmor is a feature flag to enable AppArmor in ActiveGate container
func (dk *DynaKube) FeatureActiveGateAppArmor() bool {
	return dk.getFeatureFlagRaw(AnnotationFeatureActiveGateAppArmor) == truePhrase
}

// FeatureOneAgentIgnoreProxy is a feature flag to ignore the proxy for oneAgents when set in CR
func (dk *DynaKube) FeatureOneAgentIgnoreProxy() bool {
	return dk.getFeatureFlagRaw(AnnotationFeatureOneAgentIgnoreProxy) == truePhrase
}

// FeatureActiveGateIgnoreProxy is a feature flag to ignore the proxy for ActiveGate when set in CR
func (dk *DynaKube) FeatureActiveGateIgnoreProxy() bool {
	return dk.getFeatureFlagRaw(AnnotationFeatureActiveGateIgnoreProxy) == truePhrase
}

// FeatureActiveGateAuthToken is a feature flag to enable authToken usage in the activeGate
func (dk *DynaKube) FeatureActiveGateAuthToken() bool {
	return dk.getFeatureFlagRaw(AnnotationFeatureActiveGateAuthToken) != falsePhrase
}

// FeatureLabelVersionDetection is a feature flag to enable injecting additional environment variables based on user labels
func (dk *DynaKube) FeatureLabelVersionDetection() bool {
	return dk.getFeatureFlagRaw(AnnotationFeatureLabelVersionDetection) != falsePhrase
}

// FeatureAgentInitialConnectRetry is a feature flag to configure startup delay of standalone agents
func (dk *DynaKube) FeatureAgentInitialConnectRetry() int {
	return dk.getFeatureFlagInt(AnnotationFeatureOneAgentInitialConnectRetry, -1)
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

func (dk *DynaKube) FeatureSyntheticNodeType() string {
	node := dk.getFeatureFlagRaw(AnnotationFeatureSyntheticNodeType)
	if node == "" {
		return SyntheticNodeS
	}
	return node
}

func (dk *DynaKube) FeatureSyntheticLocationEntityId() string {
	return dk.getFeatureFlagRaw(AnnotationFeatureSyntheticLocationEntityId)
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

func (dk *DynaKube) FeatureSyntheticReplicas() int32 {
	value := dk.getFeatureFlagRaw(AnnotationFeatureSyntheticReplicas)
	if value == "" {
		return defaultSyntheticReplicas
	}

	parsed, err := strconv.ParseInt(value, 0, 32)
	if err != nil {
		return defaultSyntheticReplicas
	}

	return int32(parsed)
}

func (dk *DynaKube) FeatureInitContainerSeccomp() bool {
	return dk.getFeatureFlagRaw(AnnotationFeatureInitContainerSeccomp) == truePhrase
}
