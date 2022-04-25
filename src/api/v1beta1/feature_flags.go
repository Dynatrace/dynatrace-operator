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
	AnnotationFeatureDisableActiveGateUpdates         = AnnotationFeaturePrefix + "disable-activegate-updates"
	AnnotationFeatureDisableActiveGateRawImage        = AnnotationFeaturePrefix + "disable-activegate-raw-image"
	AnnotationFeatureActiveGateAppArmor               = AnnotationFeaturePrefix + "activegate-apparmor"
	AnnotationFeatureActiveGateReadOnlyFilesystem     = AnnotationFeaturePrefix + "activegate-readonly-fs"
	AnnotationFeatureAutomaticKubernetesApiMonitoring = AnnotationFeaturePrefix + "automatic-kubernetes-api-monitoring"
	AnnotationFeatureActiveGateIgnoreProxy            = AnnotationFeaturePrefix + "activegate-ignore-proxy"

	// statsD
	AnnotationFeatureUseActiveGateImageForStatsd = AnnotationFeaturePrefix + "use-activegate-image-for-statsd"
	AnnotationFeatureCustomEecImage              = AnnotationFeaturePrefix + "custom-eec-image"
	AnnotationFeatureCustomStatsdImage           = AnnotationFeaturePrefix + "custom-statsd-image"

	// dtClient
	AnnotationFeatureDisableHostsRequests = AnnotationFeaturePrefix + "disable-hosts-requests"

	// oneAgent
	AnnotationFeatureOneAgentMaxUnavailable       = AnnotationFeaturePrefix + "oneagent-max-unavailable"
	AnnotationFeatureDisableReadOnlyOneAgent      = AnnotationFeaturePrefix + "disable-oneagent-readonly-host-fs"
	AnnotationFeatureEnableMultipleOsAgentsOnNode = AnnotationFeaturePrefix + "multiple-osagents-on-node"
	AnnotationFeatureOneAgentIgnoreProxy          = AnnotationFeaturePrefix + "oneagent-ignore-proxy"

	// injection (webhook)
	AnnotationFeatureEnableWebhookReinvocationPolicy = AnnotationFeaturePrefix + "enable-webhook-reinvocation-policy"
	AnnotationFeatureIgnoreUnknownState              = AnnotationFeaturePrefix + "ignore-unknown-state"
	AnnotationFeatureIgnoredNamespaces               = AnnotationFeaturePrefix + "ignored-namespaces"
	AnnotationFeatureDisableMetadataEnrichment       = AnnotationFeaturePrefix + "disable-metadata-enrichment"
)

var (
	log = logger.NewDTLogger().WithName("dynakube-api")
)

// FeatureDisableActiveGateUpdates is a feature flag to disable ActiveGate updates.
func (dk *DynaKube) FeatureDisableActiveGateUpdates() bool {
	return dk.getFeatureFlagRaw(AnnotationFeatureDisableActiveGateUpdates) == "true"
}

// FeatureDisableHostsRequests is a feature flag to disable queries to the Hosts API.
func (dk *DynaKube) FeatureDisableHostsRequests() bool {
	return dk.getFeatureFlagRaw(AnnotationFeatureDisableHostsRequests) == "true"
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

// FeatureEnableWebhookReinvocationPolicy is a feature flag to enable instrumenting missing containers
// by enabling reinvocation for webhook.
func (dk *DynaKube) FeatureEnableWebhookReinvocationPolicy() bool {
	return dk.getFeatureFlagRaw(AnnotationFeatureEnableWebhookReinvocationPolicy) == "true"
}

// FeatureIgnoreUnknownState is a feature flag that makes the operator inject into applications even when the dynakube is in an UNKNOWN state,
// this may cause extra host to appear in the tenant for each process.
func (dk *DynaKube) FeatureIgnoreUnknownState() bool {
	return dk.getFeatureFlagRaw(AnnotationFeatureIgnoreUnknownState) == "true"
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
	return dk.getFeatureFlagRaw(AnnotationFeatureAutomaticKubernetesApiMonitoring) == "true"
}

// FeatureDisableMetadataEnrichment is a feature flag to disable metadata enrichment,
func (dk *DynaKube) FeatureDisableMetadataEnrichment() bool {
	return dk.getFeatureFlagRaw(AnnotationFeatureDisableMetadataEnrichment) == "true"
}

// FeatureUseActiveGateImageForStatsd is a feature flag that makes the operator use ActiveGate image when initializing Extension Controller and Statsd containers
// (using special predefined entry points).
func (dk *DynaKube) FeatureUseActiveGateImageForStatsd() bool {
	return dk.getFeatureFlagRaw(AnnotationFeatureUseActiveGateImageForStatsd) == "true"
}

// FeatureCustomEecImage is a feature flag to specify custom Extension Controller Docker image path
func (dk *DynaKube) FeatureCustomEecImage() string {
	return dk.getFeatureFlagRaw(AnnotationFeatureCustomEecImage)
}

// FeatureCustomStatsdImage is a feature flag to specify custom StatsD Docker image path
func (dk *DynaKube) FeatureCustomStatsdImage() string {
	return dk.getFeatureFlagRaw(AnnotationFeatureCustomStatsdImage)
}

// FeatureDisableReadOnlyOneAgent is a feature flag to specify if the operator needs to deploy the oneagents in a readonly mode,
// where the csi-driver would provide the volume for logs and such
// Defaults to false
func (dk *DynaKube) FeatureDisableReadOnlyOneAgent() bool {
	return dk.getFeatureFlagRaw(AnnotationFeatureDisableReadOnlyOneAgent) == "true"
}

// FeatureDisableActivegateRawImage is a feature flag to specify if the operator should
// fetch from cluster and set in ActiveGet container: tenant UUID, token and communication endpoints
// instead of using embedded ones in the image
// Defaults to false
func (dk *DynaKube) FeatureDisableActivegateRawImage() bool {
	return dk.getFeatureFlagRaw(AnnotationFeatureDisableActiveGateRawImage) == "true"
}

// FeatureEnableMultipleOsAgentsOnNode is a feature flag to enable multiple osagents running on the same host
func (dk *DynaKube) FeatureEnableMultipleOsAgentsOnNode() bool {
	return dk.getFeatureFlagRaw(AnnotationFeatureEnableMultipleOsAgentsOnNode) == "true"
}

// FeatureActiveGateReadOnlyFilesystem is a feature flag to enable RO mounted filesystem in ActiveGate container
func (dk *DynaKube) FeatureActiveGateReadOnlyFilesystem() bool {
	return dk.getFeatureFlagRaw(AnnotationFeatureActiveGateReadOnlyFilesystem) == "true"
}

// FeatureActiveGateAppArmor is a feature flag to enable AppArmor in ActiveGate container
func (dk *DynaKube) FeatureActiveGateAppArmor() bool {
	return dk.getFeatureFlagRaw(AnnotationFeatureActiveGateAppArmor) == "true"
}

// FeatureOneAgentIgnoreProxy is a feature flag to ignore the proxy for oneAgents when set in CR
func (dk *DynaKube) FeatureOneAgentIgnoreProxy() bool {
	return dk.getFeatureFlagRaw(AnnotationFeatureOneAgentIgnoreProxy) == "true"
}

// FeatureActiveGateIgnoreProxy is a feature flag to ignore the proxy for ActiveGate when set in CR
func (dk *DynaKube) FeatureActiveGateIgnoreProxy() bool {
	return dk.getFeatureFlagRaw(AnnotationFeatureActiveGateIgnoreProxy) == "true"
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
