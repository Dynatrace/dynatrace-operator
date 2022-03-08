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

	"github.com/Dynatrace/dynatrace-operator/src/logger"
)

const (
	PublicAnnotationPrefix                            = "alpha.operator.dynatrace.com/"
	annotationFeaturePrefix                           = PublicAnnotationPrefix + "feature-"
	annotationFeatureDisableActiveGateUpdates         = annotationFeaturePrefix + "disable-activegate-updates"
	annotationFeatureDisableHostsRequests             = annotationFeaturePrefix + "disable-hosts-requests"
	annotationFeatureOneAgentMaxUnavailable           = annotationFeaturePrefix + "oneagent-max-unavailable"
	annotationFeatureEnableWebhookReinvocationPolicy  = annotationFeaturePrefix + "enable-webhook-reinvocation-policy"
	annotationFeatureIgnoreUnknownState               = annotationFeaturePrefix + "ignore-unknown-state"
	annotationFeatureIgnoredNamespaces                = annotationFeaturePrefix + "ignored-namespaces"
	annotationFeatureAutomaticKubernetesApiMonitoring = annotationFeaturePrefix + "automatic-kubernetes-api-monitoring"
	annotationFeatureDisableMetadataEnrichment        = annotationFeaturePrefix + "disable-metadata-enrichment"
	annotationFeatureUseActiveGateImageForStatsd      = annotationFeaturePrefix + "use-activegate-image-for-statsd"
	annotationFeatureCustomEecImage                   = annotationFeaturePrefix + "custom-eec-image"
	annotationFeatureCustomStatsdImage                = annotationFeaturePrefix + "custom-statsd-image"
	AnnotationFeatureDisableReadOnlyOneAgent          = annotationFeaturePrefix + "disable-oneagent-readonly-host-fs"
	AnnotationFeatureEnableMultipleOsAgentsOnNode     = annotationFeaturePrefix + "multiple-osagents-on-node"
)

var (
	log = logger.NewDTLogger().WithName("dynakube-api")
)

// FeatureDisableActiveGateUpdates is a feature flag to disable ActiveGate updates.
func (dk *DynaKube) FeatureDisableActiveGateUpdates() bool {
	return dk.Annotations[annotationFeatureDisableActiveGateUpdates] == "true"
}

// FeatureDisableHostsRequests is a feature flag to disable queries to the Hosts API.
func (dk *DynaKube) FeatureDisableHostsRequests() bool {
	return dk.Annotations[annotationFeatureDisableHostsRequests] == "true"
}

// FeatureOneAgentMaxUnavailable is a feature flag to configure maxUnavailable on the OneAgent DaemonSets rolling upgrades.
func (dk *DynaKube) FeatureOneAgentMaxUnavailable() int {
	raw := dk.Annotations[annotationFeatureOneAgentMaxUnavailable]
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
	return dk.Annotations[annotationFeatureEnableWebhookReinvocationPolicy] == "true"
}

// GetFeatureEnableWebhookReinvocationPolicy returns the annotation for FeatureEnableWebhookReinvocationPolicy
func (dk *DynaKube) GetFeatureEnableWebhookReinvocationPolicy() string {
	return annotationFeatureEnableWebhookReinvocationPolicy
}

// FeatureIgnoreUnknownState is a feature flag that makes the operator inject into applications even when the dynakube is in an UNKNOWN state,
// this may cause extra host to appear in the tenant for each process.
func (dk *DynaKube) FeatureIgnoreUnknownState() bool {
	return dk.Annotations[annotationFeatureIgnoreUnknownState] == "true"
}

// FeatureIgnoredNamespaces is a feature flag for ignoring certain namespaces.
// defaults to "[ \"^dynatrace$\", \"^kube-.*\", \"openshift(-.*)?\" ]"
func (dk *DynaKube) FeatureIgnoredNamespaces() []string {
	raw, ok := dk.Annotations[annotationFeatureIgnoredNamespaces]
	if !ok || raw == "" {
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
	return dk.Annotations[annotationFeatureAutomaticKubernetesApiMonitoring] == "true"
}

// FeatureDisableMetadataEnrichment is a feature flag to disable metadata enrichment,
func (dk *DynaKube) FeatureDisableMetadataEnrichment() bool {
	return dk.Annotations[annotationFeatureDisableMetadataEnrichment] == "true"
}

// FeatureUseActiveGateImageForStatsd is a feature flag that makes the operator use ActiveGate image when initializing Extension Controller and Statsd containers
// (using special predefined entry points).
func (dk *DynaKube) FeatureUseActiveGateImageForStatsd() bool {
	return dk.Annotations[annotationFeatureUseActiveGateImageForStatsd] == "true"
}

// FeatureCustomEecImage is a feature flag to specify custom Extension Controller Docker image path
func (dk *DynaKube) FeatureCustomEecImage() string {
	return dk.Annotations[annotationFeatureCustomEecImage]
}

// FeatureCustomStatsdImage is a feature flag to specify custom StatsD Docker image path
func (dk *DynaKube) FeatureCustomStatsdImage() string {
	return dk.Annotations[annotationFeatureCustomStatsdImage]
}

// FeatureDisableReadOnlyOneAgent is a feature flag to specify if the operator needs to deploy the oneagents in a readonly mode,
// where the csi-driver would provide the volume for logs and such
// Defaults to false
func (dk *DynaKube) FeatureDisableReadOnlyOneAgent() bool {
	return dk.Annotations[AnnotationFeatureDisableReadOnlyOneAgent] == "true"
}

// FeatureEnableMultipleOsAgentsOnNode is a feature flag to enable multiple osagents running on the same host
func (dk *DynaKube) FeatureEnableMultipleOsAgentsOnNode() bool {
	return dk.Annotations[AnnotationFeatureEnableMultipleOsAgentsOnNode] == "true"
}
