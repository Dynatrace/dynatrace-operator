package exp

import (
	"strconv"
	"time"
)

const (
	FFPrefix = "feature.dynatrace.com/"

	PublicRegistryKey = FFPrefix + "public-registry"
	NoProxyKey        = FFPrefix + "no-proxy"

	// Deprecated: Dedicated field since v1beta2.
	APIRequestThresholdKey = FFPrefix + "dynatrace-api-request-threshold"

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

// Deprecated: Dedicated field since v1beta2.
func (ff *FeatureFlags) GetAPIRequestThreshold() time.Duration {
	interval := ff.getIntWithDefault(APIRequestThresholdKey, DefaultMinRequestThresholdMinutes)
	if interval < 0 {
		interval = DefaultMinRequestThresholdMinutes
	}

	return time.Duration(interval) * time.Minute
}

// GetNoProxy is a feature flag to set the NO_PROXY value to be used by the dtClient.
func (ff *FeatureFlags) GetNoProxy() string {
	return ff.getRaw(NoProxyKey)
}

// GetAGConnectionInfo is a feature flag to
func (ff *FeatureFlags) GetAGConnectionInfo() string {
	return ff.getRaw(FFPrefix + "ag-connection-info")
}

// GetInClusterAGDNSEntryPoint is a feature flag to
func (ff *FeatureFlags) GetInClusterAGDNSEntryPoint() string {
	return ff.getRaw(FFPrefix + "incluster-ag-dns-entry-point")
}

// GetComponentNoProxy is a feature flag to
func (ff *FeatureFlags) GetComponentNoProxy() string {
	return ff.getRaw(FFPrefix + "component-no-proxy")
}

// GetOtelcDtEndpoint is a feature flag to
func (ff *FeatureFlags) GetOtelcDtEndpoint() string {
	return ff.getRaw(FFPrefix + "otelc-dt-endpoint")
}

// GetOtelcAGTLSSecretName is a feature flag to
func (ff *FeatureFlags) GetOtelcAGTLSSecretName() string {
	return ff.getRaw(FFPrefix + "otelc-ag-tls-secret-name")
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
