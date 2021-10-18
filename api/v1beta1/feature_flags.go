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
	"strconv"

	"github.com/Dynatrace/dynatrace-operator/logger"
	corev1 "k8s.io/api/core/v1"
)

const (
	annotationFeaturePrefix                          = "alpha.operator.dynatrace.com/feature-"
	annotationFeatureDisableActiveGateUpdates        = annotationFeaturePrefix + "disable-activegate-updates"
	annotationFeatureDisableHostsRequests            = annotationFeaturePrefix + "disable-hosts-requests"
	annotationFeatureOneAgentMaxUnavailable          = annotationFeaturePrefix + "oneagent-max-unavailable"
	annotationFeatureEnableWebhookReinvocationPolicy = annotationFeaturePrefix + "enable-webhook-reinvocation-policy"
	annotationFeatureIgnoreUnknownState              = annotationFeaturePrefix + "ignore-unknown-state"
	annotationFeatureCSITolerations                  = annotationFeaturePrefix + "csi-tolerations"
	annotationFeatureIgnoredNamespaces               = annotationFeaturePrefix + "ignored-namespaces"
)

var (
	log = logger.NewDTLogger()

	defaultIgnoredNamespaces = []string{
		"^dynatrace$",
		"^kube-.*",
		"^openshift(-.*)?",
	}
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

// FeatureCSITolerations is a feature flag to set tolerations for the csi drive daemonset.
// example: [{\"key\":\"node-role.kubernetes.io/master\",\"operator\":\"Exists\",\"effect\":\"NoSchedule\"}]
func (dk *DynaKube) FeatureCSITolerations() []corev1.Toleration {
	raw, ok := dk.Annotations[annotationFeatureCSITolerations]
	if !ok || raw == "" {
		return nil
	}
	tolerations := &[]corev1.Toleration{}
	err := json.Unmarshal([]byte(raw), tolerations)
	if err != nil {
		log.Error(err, "failed to unmarshal csi tolerations feature-flag")
		return nil
	}
	return *tolerations
}

// FeatureIgnoredNamespaces is a feature flag for ignoreing certain namespaces.
// defaults to "[ \"^dynatrace$\", \"^kube-.*\", \"openshift(-.*)?\" ]"
func (dk *DynaKube) FeatureIgnoredNamespaces() []string {
	raw, ok := dk.Annotations[annotationFeatureIgnoredNamespaces]
	if !ok || raw == "" {
		return defaultIgnoredNamespaces
	}
	ignoredNamespaces := &[]string{}
	err := json.Unmarshal([]byte(raw), ignoredNamespaces)
	if err != nil {
		log.Error(err, "failed to unmarshal ignoredNamespaces feature-flag")
		return defaultIgnoredNamespaces
	}
	return *ignoredNamespaces
}
