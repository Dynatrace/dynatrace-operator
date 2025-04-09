package exp

import (
	"encoding/json"
	"fmt"
)

const (
	// Deprecated: Dedicated field since v1beta3
	InjectionDisableMetadataEnrichmentKey = FFPrefix + "disable-metadata-enrichment"
	// Deprecated: Dedicated field since v1beta3
	InjectionMetadataEnrichmentKey = FFPrefix + "metadata-enrichment"

	InjectionIgnoreUnknownStateKey    = FFPrefix + "ignore-unknown-state"
	InjectionIgnoredNamespacesKey     = FFPrefix + "ignored-namespaces"
	InjectionAutomaticKey             = FFPrefix + "automatic-injection"
	InjectionLabelVersionDetectionKey = FFPrefix + "label-version-detection"
	InjectionFailurePolicyKey         = FFPrefix + "injection-failure-policy"
	InjectionSeccompKey               = FFPrefix + "init-container-seccomp-profile"
	InjectionEnforcementModeKey       = FFPrefix + "enforcement-mode"
)

// Deprecated: Dedicated field since v1beta3
func (ff *FeatureFlags) DisableMetadataEnrichment() bool {
	return ff.getDisableFlagWithDeprecatedAnnotation(InjectionMetadataEnrichmentKey, InjectionDisableMetadataEnrichmentKey)
}

// IgnoreUnknownState is a feature flag that makes the operator inject into applications even when the dynakube is in an UNKNOWN state,
// this may cause extra host to appear in the tenant for each process.
func (ff *FeatureFlags) IgnoreUnknownState() bool {
	return ff.getFeatureFlagRaw(InjectionIgnoreUnknownStateKey) == truePhrase
}

// IsInjectionAutomatic controls OneAgent is injected to pods in selected namespaces automatically ("automatic-injection=true" or flag not set)
// or if pods need to be opted-in one by one ("automatic-injection=false").
func (ff *FeatureFlags) IsInjectionAutomatic() bool {
	return ff.getFeatureFlagRaw(InjectionAutomaticKey) != falsePhrase
}

// GetIgnoredNamespaces is a feature flag for ignoring certain namespaces.
// defaults to "[ \"^dynatrace$\", \"^kube-.*\", \"openshift(-.*)?\" ]".
func (ff *FeatureFlags) GetIgnoredNamespaces(ns string) []string {
	raw := ff.getFeatureFlagRaw(InjectionIgnoredNamespacesKey)
	if raw == "" {
		return ff.getDefaultIgnoredNamespaces(ns)
	}

	ignoredNamespaces := &[]string{}

	err := json.Unmarshal([]byte(raw), ignoredNamespaces)
	if err != nil {

		return ff.getDefaultIgnoredNamespaces(ns)
	}

	return *ignoredNamespaces
}

func (ff *FeatureFlags) getDefaultIgnoredNamespaces(ns string) []string {
	defaultIgnoredNamespaces := []string{
		fmt.Sprintf("^%s$", ns),
		"^kube-.*",
		"^openshift(-.*)?",
		"^gke-.*",
		"^gmp-.*",
	}

	return defaultIgnoredNamespaces
}

// IsLabelVersionDetection is a feature flag to enable injecting additional environment variables based on user labels.
func (ff *FeatureFlags) IsLabelVersionDetection() bool {
	return ff.getFeatureFlagRaw(InjectionLabelVersionDetectionKey) == truePhrase
}

func (ff *FeatureFlags) GetInjectionFailurePolicy() string {
	if ff.getFeatureFlagRaw(InjectionFailurePolicyKey) == failPhrase {
		return failPhrase
	}

	return silentPhrase
}

func (ff *FeatureFlags) HasInitSeccomp() bool {
	return ff.getFeatureFlagRaw(InjectionSeccompKey) == truePhrase
}

// IsEnforcementMode is a feature flag to control how the initContainer
// sets the tenantUUID to the container.conf file (always vs if oneAgent is present).
func (ff *FeatureFlags) IsEnforcementMode() bool {
	return ff.getFeatureFlagRaw(InjectionEnforcementModeKey) != falsePhrase
}
