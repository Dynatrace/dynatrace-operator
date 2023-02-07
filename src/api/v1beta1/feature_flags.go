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

	"github.com/Dynatrace/dynatrace-operator/src/logger"
)

const (
	DeprecatedFeatureFlagPrefix = "alpha.operator.dynatrace.com/feature-"

	AnnotationFeaturePrefix = "feature.dynatrace.com/"

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

	AnnotationFeatureCustomSyntheticImage  = AnnotationFeaturePrefix + "custom-synthetic-image"
	AnnotationFeatureCustomDynaMetricImage = AnnotationFeaturePrefix + "custom-dynametric-image"

	// dtClient

	// Deprecated: AnnotationFeatureDisableHostsRequests use AnnotationFeatureHostsRequests instead
	AnnotationFeatureDisableHostsRequests = AnnotationFeaturePrefix + "disable-hosts-requests"
	AnnotationFeatureHostsRequests        = AnnotationFeaturePrefix + "hosts-requests"

	// oneAgent

	// Deprecated: AnnotationFeatureDisableReadOnlyOneAgent use AnnotationFeatureReadOnlyOneAgent instead
	AnnotationFeatureDisableReadOnlyOneAgent = AnnotationFeaturePrefix + "disable-oneagent-readonly-host-fs"

	AnnotationFeatureReadOnlyOneAgent = AnnotationFeaturePrefix + "oneagent-readonly-host-fs"

	AnnotationFeatureMultipleOsAgentsOnNode         = AnnotationFeaturePrefix + "multiple-osagents-on-node"
	AnnotationFeatureOneAgentMaxUnavailable         = AnnotationFeaturePrefix + "oneagent-max-unavailable"
	AnnotationFeatureOneAgentIgnoreProxy            = AnnotationFeaturePrefix + "oneagent-ignore-proxy"
	AnnotationFeatureOneAgentInitialConnectRetry    = AnnotationFeaturePrefix + "oneagent-initial-connect-retry-ms"
	AnnotationFeatureRunOneAgentContainerPrivileged = AnnotationFeaturePrefix + "oneagent-privileged"

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

	// CSI
	AnnotationFeatureMaxFailedCsiMountAttempts = AnnotationFeaturePrefix + "max-csi-mount-attempts"

	falsePhrase = "false"
	truePhrase  = "true"

	// synthetic location
	AnnotationFeatureSyntheticLocationEntityId = AnnotationFeaturePrefix + "synthetic-location-entity-id"

	// synthetic node type
	AnnotationFeatureSyntheticNodeType = AnnotationFeaturePrefix + "synthetic-node-type"

	SyntheticNodeXs = "XS"
	SyntheticNodeS  = "S"
	SyntheticNodeM  = "M"

	// synthetic autoscaler
	AnnotationFeatureSyntheticAutoscalerMinReplicas = AnnotationFeaturePrefix + "synthetic-autoscaler-min-replicas"
	AnnotationFeatureSyntheticAutoscalerMaxReplicas = AnnotationFeaturePrefix + "synthetic-autoscaler-max-replicas"
	AnnotationFeatureSyntheticAutoscalerDynaQuery   = AnnotationFeaturePrefix + "synthetic-autoscaler-dynaquery"

	// dynaMetrics token
	AnnotationFeatureDynaMetricsToken = AnnotationFeaturePrefix + "dynametrics-token"
)

const (
	DefaultMaxFailedCsiMountAttempts = 10

	defaultSyntheticAutoscalerDynaQuery = `dsfm:synthetic.engine_utilization:filter(eq("dt.entity.synthetic_location","%s")):merge("host.name","dt.active_gate.working_mode","dt.active_gate.id","location.name"):fold(avg)`
)

var (
	log = logger.Factory.GetLogger("dynakube-api")

	DefaultSyntheticAutoscalerMinReplicas = int32(1)
	defaultSyntheticAutoscalerMaxReplicas = int32(2)
)

func (dk *DynaKube) getDisableFlagWithDeprecatedAnnotation(annotation string, deprecatedAnnotation string) bool {
	return dk.getFeatureFlagRaw(annotation) == falsePhrase ||
		dk.getFeatureFlagRaw(deprecatedAnnotation) == truePhrase && dk.getFeatureFlagRaw(annotation) == ""
}

// FeatureDisableActiveGateUpdates is a feature flag to disable ActiveGate updates.
func (dk *DynaKube) FeatureDisableActiveGateUpdates() bool {
	return dk.getDisableFlagWithDeprecatedAnnotation(AnnotationFeatureActiveGateUpdates, AnnotationFeatureDisableActiveGateUpdates)
}

// FeatureDisableHostsRequests is a feature flag to disable queries to the Hosts API.
func (dk *DynaKube) FeatureDisableHostsRequests() bool {
	return dk.getDisableFlagWithDeprecatedAnnotation(AnnotationFeatureHostsRequests, AnnotationFeatureDisableHostsRequests)
}

// FeatureOneAgentMaxUnavailable is a feature flag to configure maxUnavailable on the OneAgent DaemonSets rolling upgrades.
func (dk *DynaKube) FeatureOneAgentMaxUnavailable() int {
	raw := dk.getFeatureFlagRaw(AnnotationFeatureOneAgentMaxUnavailable)
	if raw == "" {
		return 1
	}

	val, err := strconv.Atoi(raw)
	if err != nil {
		return 1
	}

	return val
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
	return dk.getFeatureFlagRaw(AnnotationFeatureAutomaticK8sApiMonitoring) == truePhrase
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

func (dk *DynaKube) FeatureCustomDynaMetricImage() string {
	return dk.getFeatureFlagRaw(AnnotationFeatureCustomDynaMetricImage)
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
	return dk.getFeatureFlagRaw(AnnotationFeatureActiveGateReadOnlyFilesystem) == truePhrase
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
	return dk.getFeatureFlagRaw(AnnotationFeatureLabelVersionDetection) == truePhrase
}

// FeatureAgentInitialConnectRetry is a feature flag to configure startup delay of standalone agents
func (dk *DynaKube) FeatureAgentInitialConnectRetry() int {
	raw := dk.getFeatureFlagRaw(AnnotationFeatureOneAgentInitialConnectRetry)
	if raw == "" {
		return -1
	}

	val, err := strconv.Atoi(raw)
	if err != nil {
		log.Error(err, "failed to parse agentInitialConnectRetry feature-flag")
		return -1
	}

	return val
}

func (dk *DynaKube) FeatureOneAgentPrivileged() bool {
	return dk.getFeatureFlagRaw(AnnotationFeatureRunOneAgentContainerPrivileged) == truePhrase
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

func (dk *DynaKube) FeatureMaxFailedCsiMountAttempts() int {
	maxCsiMountAttemptsValue := dk.getFeatureFlagRaw(AnnotationFeatureMaxFailedCsiMountAttempts)

	if maxCsiMountAttemptsValue == "" {
		return DefaultMaxFailedCsiMountAttempts
	}

	maxCsiMountAttempts, err := strconv.Atoi(maxCsiMountAttemptsValue)

	if err != nil || maxCsiMountAttempts < 0 {
		return DefaultMaxFailedCsiMountAttempts
	}

	return maxCsiMountAttempts
}

func (dynaKube *DynaKube) FeatureSyntheticLocationEntityId() string {
	return dynaKube.Annotations[AnnotationFeatureSyntheticLocationEntityId]
}

func (dynaKube *DynaKube) FeatureSyntheticNodeType() string {
	node, containsKey := dynaKube.Annotations[AnnotationFeatureSyntheticNodeType]
	if !containsKey {
		return SyntheticNodeS
	}
	return node
}

func (dynaKube *DynaKube) FeatureSyntheticAutoscalerMinReplicas() int32 {
	return dynaKube.getSyntheticAutoscalerReplicas(
		AnnotationFeatureSyntheticAutoscalerMinReplicas,
		DefaultSyntheticAutoscalerMinReplicas)
}

func (dynaKube *DynaKube) getSyntheticAutoscalerReplicas(feature string, defaultReplicas int32) int32 {
	replicas := defaultReplicas
	value, containsKey := dynaKube.Annotations[feature]
	if containsKey {
		parsed, err := strconv.ParseInt(value, 0, 32)
		if err == nil {
			replicas = int32(parsed)
		}
	}

	return replicas
}

func (dynaKube *DynaKube) FeatureSyntheticAutoscalerMaxReplicas() int32 {
	return dynaKube.getSyntheticAutoscalerReplicas(
		AnnotationFeatureSyntheticAutoscalerMaxReplicas,
		defaultSyntheticAutoscalerMaxReplicas)
}

func (dynaKube *DynaKube) FeatureSyntheticAutoscalerDynaQuery() string {
	query, containsKey := dynaKube.Annotations[AnnotationFeatureSyntheticAutoscalerDynaQuery]
	if !containsKey {
		query = defaultSyntheticAutoscalerDynaQuery
	}

	return fmt.Sprintf(query, dynaKube.FeatureSyntheticLocationEntityId())
}

// a token to offer external metrics from Dynatrace
func (dynaKube *DynaKube) FeatureDynaMetricsToken() string {
	return dynaKube.Annotations[AnnotationFeatureDynaMetricsToken]
}
