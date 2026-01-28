package exp

import (
	"strconv"
)

const (
	FFPrefix = "feature.dynatrace.com/"

	PublicRegistryKey = FFPrefix + "public-registry"
	NoProxyKey        = FFPrefix + "no-proxy"

	UseEECLegacyMountsKey = FFPrefix + "use-eec-legacy-mounts"

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

// GetNoProxy is a feature flag to set the NO_PROXY value to be used by the dtClient.
func (ff *FeatureFlags) GetNoProxy() string {
	return ff.getRaw(NoProxyKey)
}

func (ff *FeatureFlags) IsPublicRegistry() bool {
	return ff.getBoolWithDefault(PublicRegistryKey, false)
}

func (ff *FeatureFlags) UseEECLegacyMounts() bool {
	return ff.getBoolWithDefault(UseEECLegacyMountsKey, true)
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
	return ff.annotations[annotation]
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
