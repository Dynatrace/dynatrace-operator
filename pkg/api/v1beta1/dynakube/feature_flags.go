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
	"strconv"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
)

const (
	AnnotationFeaturePrefix = "feature.dynatrace.com/"

	// activeGate.

	// Deprecated: AnnotationFeatureDisableActiveGateUpdates use AnnotationFeatureActiveGateUpdates instead.
	AnnotationFeatureAutomaticK8sApiMonitoring = AnnotationFeaturePrefix + "automatic-kubernetes-api-monitoring"
	AnnotationFeatureActiveGateIgnoreProxy     = AnnotationFeaturePrefix + "activegate-ignore-proxy"

	// dtClient.
	AnnotationFeatureApiRequestThreshold = AnnotationFeaturePrefix + "dynatrace-api-request-threshold"

	// oneAgent.
	AnnotationFeatureOneAgentSecCompProfile = AnnotationFeaturePrefix + "oneagent-seccomp-profile"

	// injection (webhook).
	// Deprecated: AnnotationFeatureDisableMetadataEnrichment use AnnotationFeatureMetadataEnrichment instead.
	AnnotationFeatureDisableMetadataEnrichment = AnnotationFeaturePrefix + "disable-metadata-enrichment"
	AnnotationFeatureMetadataEnrichment        = AnnotationFeaturePrefix + "metadata-enrichment"

	// CSI.
	AnnotationFeatureMaxFailedCsiMountAttempts = AnnotationFeaturePrefix + "max-csi-mount-attempts"

	falsePhrase = "false"
	truePhrase  = "true"
)

const (
	DefaultMinRequestThresholdMinutes = 15
	DefaultMaxFailedCsiMountAttempts  = 10
)

var (
	log = logd.Get().WithName("dynakube-api")
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

func (dk *DynaKube) FeatureMaxFailedCsiMountAttempts() int {
	maxCsiMountAttemptsValue := dk.getFeatureFlagInt(AnnotationFeatureMaxFailedCsiMountAttempts, DefaultMaxFailedCsiMountAttempts)
	if maxCsiMountAttemptsValue < 0 {
		return DefaultMaxFailedCsiMountAttempts
	}

	return maxCsiMountAttemptsValue
}

func (dk *DynaKube) FeatureApiRequestThreshold() time.Duration {
	interval := dk.getFeatureFlagInt(AnnotationFeatureApiRequestThreshold, DefaultMinRequestThresholdMinutes)
	if interval < 0 {
		interval = DefaultMinRequestThresholdMinutes
	}

	return time.Duration(interval) * time.Minute
}

// FeatureDisableMetadataEnrichment is a feature flag to disable metadata enrichment,.
func (dk *DynaKube) FeatureDisableMetadataEnrichment() bool {
	return dk.getDisableFlagWithDeprecatedAnnotation(AnnotationFeatureMetadataEnrichment, AnnotationFeatureDisableMetadataEnrichment)
}

func (dk *DynaKube) FeatureOneAgentSecCompProfile() string {
	return dk.getFeatureFlagRaw(AnnotationFeatureOneAgentSecCompProfile)
}
