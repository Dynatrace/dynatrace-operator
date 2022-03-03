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
	DeprecatedFeatureFlagPrefix                       = "alpha-"
	FeatureFlagAnnotationPrefix                       = "operator.dynatrace.com/"
	annotationFeaturePrefix                           = FeatureFlagAnnotationPrefix + "feature-"
	annotationFeatureDisableActiveGateUpdates         = annotationFeaturePrefix + "disable-activegate-updates"
	annotationFeatureDisableHostsRequests             = annotationFeaturePrefix + "disable-hosts-requests"
	annotationFeatureOneAgentMaxUnavailable           = annotationFeaturePrefix + "oneagent-max-unavailable"
	AnnotationFeatureEnableWebhookReinvocationPolicy  = annotationFeaturePrefix + "enable-webhook-reinvocation-policy"
	annotationFeatureIgnoreUnknownState               = annotationFeaturePrefix + "ignore-unknown-state"
	annotationFeatureIgnoredNamespaces                = annotationFeaturePrefix + "ignored-namespaces"
	annotationFeatureAutomaticKubernetesApiMonitoring = annotationFeaturePrefix + "automatic-kubernetes-api-monitoring"
	annotationFeatureDisableMetadataEnrichment        = annotationFeaturePrefix + "disable-metadata-enrichment"
	annotationFeatureUseActiveGateImageForStatsd      = annotationFeaturePrefix + "use-activegate-image-for-statsd"
	annotationFeatureCustomEecImage                   = annotationFeaturePrefix + "custom-eec-image"
	annotationFeatureCustomStatsdImage                = annotationFeaturePrefix + "custom-statsd-image"
	AnnotationFeatureDisableReadOnlyOneAgent          = annotationFeaturePrefix + "disable-oneagent-readonly-host-fs"
	AnnotationFeatureEnableActivegateRawImage         = annotationFeaturePrefix + "enable-activegate-raw-image"
	AnnotationFeatureEnableMultipleOsAgentsOnNode     = annotationFeaturePrefix + "multiple-osagents-on-node"
	AnnotationFeatureAgReadOnlyFilesystem             = annotationFeaturePrefix + "activegate-readonly-fs"
	AnnotationFeatureAgAppArmor                       = annotationFeaturePrefix + "activegate-apparmor"
)

var (
	log = logger.NewDTLogger().WithName("dynakube-api")
)

// FeatureDisableActiveGateUpdates is a feature flag to disable ActiveGate updates.
func (dk *DynaKube) FeatureDisableActiveGateUpdates() bool {
	return dk.getFeatureFlagRaw(annotationFeatureDisableActiveGateUpdates) == "true"
}

// FeatureDisableHostsRequests is a feature flag to disable queries to the Hosts API.
func (dk *DynaKube) FeatureDisableHostsRequests() bool {
	return dk.getFeatureFlagRaw(annotationFeatureDisableHostsRequests) == "true"
}

// FeatureOneAgentMaxUnavailable is a feature flag to configure maxUnavailable on the OneAgent DaemonSets rolling upgrades.
func (dk *DynaKube) FeatureOneAgentMaxUnavailable() int {
	raw := dk.getFeatureFlagRaw(annotationFeatureOneAgentMaxUnavailable)
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
	return dk.getFeatureFlagRaw(annotationFeatureIgnoreUnknownState) == "true"
}

// FeatureIgnoredNamespaces is a feature flag for ignoring certain namespaces.
// defaults to "[ \"^dynatrace$\", \"^kube-.*\", \"openshift(-.*)?\" ]"
func (dk *DynaKube) FeatureIgnoredNamespaces() []string {
	raw := dk.getFeatureFlagRaw(annotationFeatureIgnoredNamespaces)
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
	return dk.getFeatureFlagRaw(annotationFeatureAutomaticKubernetesApiMonitoring) == "true"
}

// FeatureDisableMetadataEnrichment is a feature flag to disable metadata enrichment,
func (dk *DynaKube) FeatureDisableMetadataEnrichment() bool {
	return dk.getFeatureFlagRaw(annotationFeatureDisableMetadataEnrichment) == "true"
}

// FeatureUseActiveGateImageForStatsd is a feature flag that makes the operator use ActiveGate image when initializing Extension Controller and Statsd containers
// (using special predefined entry points).
func (dk *DynaKube) FeatureUseActiveGateImageForStatsd() bool {
	return dk.getFeatureFlagRaw(annotationFeatureUseActiveGateImageForStatsd) == "true"
}

// FeatureCustomEecImage is a feature flag to specify custom Extension Controller Docker image path
func (dk *DynaKube) FeatureCustomEecImage() string {
	return dk.getFeatureFlagRaw(annotationFeatureCustomEecImage)
}

// FeatureCustomStatsdImage is a feature flag to specify custom StatsD Docker image path
func (dk *DynaKube) FeatureCustomStatsdImage() string {
	return dk.getFeatureFlagRaw(annotationFeatureCustomStatsdImage)
}

// FeatureDisableReadOnlyOneAgent is a feature flag to specify if the operator needs to deploy the oneagents in a readonly mode,
// where the csi-driver would provide the volume for logs and such
// Defaults to false
func (dk *DynaKube) FeatureDisableReadOnlyOneAgent() bool {
	return dk.getFeatureFlagRaw(AnnotationFeatureDisableReadOnlyOneAgent) == "true"
}

func (dk *DynaKube) getFeatureFlagRaw(annotation string) string {
	if raw, ok := dk.Annotations[annotation]; ok {
		return raw
	}
	if raw, ok := dk.Annotations[DeprecatedFeatureFlagPrefix+annotation]; ok {
		return raw
	}
	return ""
}

// FeatureEnableActivegateRawImage is a feature flag to specify if the operator should
// fetch from cluster and set in ActiveGet container: tenant UUID, token and communication endpoints
// instead of using embedded ones in the image
// Defaults to false
func (dk *DynaKube) FeatureEnableActivegateRawImage() bool {
	return dk.Annotations[AnnotationFeatureEnableActivegateRawImage] == "true"
}

// FeatureEnableMultipleOsAgentsOnNode is a feature flag to enable multiple osagents running on the same host
func (dk *DynaKube) FeatureEnableMultipleOsAgentsOnNode() bool {
	return dk.Annotations[AnnotationFeatureEnableMultipleOsAgentsOnNode] == "true"
}

// FeatureActiveGateReadOnlyFilesystem is a feature flag to enable RO mounted filesystem in ActiveGate container
func (dk *DynaKube) FeatureActiveGateReadOnlyFilesystem() bool {
	return dk.Annotations[AnnotationFeatureAgReadOnlyFilesystem] == "true"
}

// FeatureActiveGateAppArmor is a feature flag to enable AppArmor in ActiveGate container
func (dk *DynaKube) FeatureActiveGateAppArmor() bool {
	return dk.Annotations[AnnotationFeatureAgAppArmor] == "true"
}
