package exp

import (
	"strconv"
	"time"
)

const (
	FFPrefix = "feature.dynatrace.com/"

	PublicRegistryKey = FFPrefix + "public-registry"
	NoProxyKey        = FFPrefix + "no-proxy"

	HostAvailabilityDetection = FFPrefix + "host-availability-detection"

	// Deprecated: Dedicated field since v1beta2.
	ApiRequestThresholdKey = FFPrefix + "dynatrace-api-request-threshold"

	silentPhrase = "silent"
	failPhrase   = "fail"

	DefaultMinRequestThresholdMinutes = 15
)

type FeatureFlags struct {
	annotations map[string]string
}

func NewFlags(annotations map[string]string) *FeatureFlags {
	return &FeatureFlags{annotations: annotations}
}

// Deprecated: Dedicated field since v1beta2.
func (ff *FeatureFlags) GetApiRequestThreshold() time.Duration {
	interval := ff.getIntWithDefault(ApiRequestThresholdKey, DefaultMinRequestThresholdMinutes)
	if interval < 0 {
		interval = DefaultMinRequestThresholdMinutes
	}

	return time.Duration(interval) * time.Minute
}

// GetNoProxy is a feature flag to set the NO_PROXY value to be used by the dtClient.
func (ff *FeatureFlags) GetNoProxy() string {
	return ff.getRaw(NoProxyKey)
}

func (ff *FeatureFlags) IsPublicRegistry() bool {
	return ff.getBoolWithDefault(PublicRegistryKey, false)
}

func (ff *FeatureFlags) IsHostAvailabilityDetectionEnabled() bool {
	return ff.getBoolWithDefault(HostAvailabilityDetection, false)
}

// Deprecated: Do not use "disable" feature flags.
func (ff *FeatureFlags) getDisableFlagWithDeprecatedAnnotation(annotation string, deprecatedAnnotation string) bool {
	if ff.getRaw(annotation) != "" {
		return !ff.getBoolWithDefault(annotation, true)
	} else {
		return ff.getBoolWithDefault(deprecatedAnnotation, false)
	}
}

func (ff *FeatureFlags) getRaw(annotation string) string {
	if ff.annotations == nil {
		return ""
	}

	if raw, ok := ff.annotations[annotation]; ok {
		return raw
	}

	return ""
}

func (ff *FeatureFlags) getBoolWithDefault(annotation string, defaultVal bool) bool {
	raw := ff.getRaw(annotation)
	if raw == "" {
		return defaultVal
	}

	val, err := strconv.ParseBool(raw)
	if err != nil {
		return defaultVal
	}

	return val
}

func (ff *FeatureFlags) getIntWithDefault(annotation string, defaultVal int) int {
	raw := ff.getRaw(annotation)
	if raw == "" {
		return defaultVal
	}

	val, err := strconv.Atoi(raw)
	if err != nil {
		return defaultVal
	}

	return val
}
