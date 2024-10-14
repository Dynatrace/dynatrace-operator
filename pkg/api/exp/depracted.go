package exp

import "time"

const (
	// Deprecated: DisableActiveGateUpdatesAnnotation use AnnotationFeatureActiveGateUpdates instead.
	DisableActiveGateUpdatesAnnotation = AnnotationPrefix + "disable-activegate-updates"

	// Deprecated: ActiveGateIgnoreProxyAnnotation use AnnotationFeatureNoProxy instead.
	ActiveGateIgnoreProxyAnnotation = AnnotationPrefix + "activegate-ignore-proxy"

	// Deprecated
	ApiRequestThresholdAnnotation = AnnotationPrefix + "dynatrace-api-request-threshold"

	// Deprecated
	OneAgentSecCompProfileAnnotation = AnnotationPrefix + "oneagent-seccomp-profile"

	// Deprecated
	DisableMetadataEnrichmentAnnotation = AnnotationPrefix + "disable-metadata-enrichment"

	// Deprecated
	MetadataEnrichmentAnnotation = AnnotationPrefix + "metadata-enrichment"

	// Deprecated: OneAgentIgnoreProxyAnnotation use AnnotationFeatureNoProxy instead.
	OneAgentIgnoreProxyAnnotation = AnnotationPrefix + "oneagent-ignore-proxy"
)

var (
	DeprecatedFeatureFlags = []string{
		DisableActiveGateUpdatesAnnotation,
		ActiveGateIgnoreProxyAnnotation,
		ApiRequestThresholdAnnotation,
		OneAgentSecCompProfileAnnotation,
		DisableMetadataEnrichmentAnnotation,
		MetadataEnrichmentAnnotation,
		OneAgentIgnoreProxyAnnotation,
	}
)

// Deprecated
func (f *FeatureFlags) ApiRequestThreshold() time.Duration {
	interval := f.getIntValue(ApiRequestThresholdAnnotation, DefaultMinRequestThresholdMinutes)
	if interval < 0 {
		interval = DefaultMinRequestThresholdMinutes
	}

	return time.Duration(interval) * time.Minute
}

// Deprecated
func (f *FeatureFlags) DisableMetadataEnrichment() bool {
	return f.getDisableFlagWithDeprecatedAnnotation(MetadataEnrichmentAnnotation, DisableMetadataEnrichmentAnnotation)
}

// Deprecated
func (f *FeatureFlags) OneAgentSecCompProfile() string {
	return f.getRawValue(OneAgentSecCompProfileAnnotation)
}

// Deprecated
func (f *FeatureFlags) IsOneAgentProxyIgnored() bool {
	return f.getRawValue(OneAgentIgnoreProxyAnnotation) == truePhrase
}

// Deprecated
func (f *FeatureFlags) IsActiveGateProxyIgnored() bool {
	return f.getRawValue(ActiveGateIgnoreProxyAnnotation) == truePhrase
}
