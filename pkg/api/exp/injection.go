package exp

import (
	"encoding/json"
	"fmt"
)

const (
	// Deprecated: Dedicated field since v1beta3.
	InjectionDisableMetadataEnrichmentKey = FFPrefix + "disable-metadata-enrichment"
	// Deprecated: Dedicated field since v1beta3.
	InjectionMetadataEnrichmentKey = FFPrefix + "metadata-enrichment"

	InjectionIgnoredNamespacesKey     = FFPrefix + "ignored-namespaces"
	InjectionAutomaticKey             = FFPrefix + "automatic-injection"
	InjectionLabelVersionDetectionKey = FFPrefix + "label-version-detection"
	InjectionFailurePolicyKey         = FFPrefix + "injection-failure-policy"

	// Deprecated: This field will be removed in a future release.
	InjectionSeccompKey = FFPrefix + "init-container-seccomp-profile"
)

// Deprecated: Dedicated field since v1beta3.
func (ff *FeatureFlags) DisableMetadataEnrichment() bool {
	return ff.getDisableFlagWithDeprecatedAnnotation(InjectionMetadataEnrichmentKey, InjectionDisableMetadataEnrichmentKey)
}

// IsAutomaticInjection controls OneAgent is injected to pods in selected namespaces automatically ("automatic-injection=true" or flag not set)
// or if pods need to be opted-in one by one ("automatic-injection=false").
func (ff *FeatureFlags) IsAutomaticInjection() bool {
	return ff.getBoolWithDefault(InjectionAutomaticKey, true)
}

// GetIgnoredNamespaces is a feature flag for ignoring certain namespaces.
// defaults to "[ \"^dynatrace$\", \"^kube-.*\", \"openshift(-.*)?\" ]".
func (ff *FeatureFlags) GetIgnoredNamespaces(ns string) []string {
	raw := ff.getRaw(InjectionIgnoredNamespacesKey)
	if raw == "" {
		return getDefaultIgnoredNamespaces(ns)
	}

	ignoredNamespaces := &[]string{}

	err := json.Unmarshal([]byte(raw), ignoredNamespaces)
	if err != nil {
		return getDefaultIgnoredNamespaces(ns)
	}

	return *ignoredNamespaces
}

func getDefaultIgnoredNamespaces(ns string) []string {
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
	return ff.getBoolWithDefault(InjectionLabelVersionDetectionKey, false)
}

func (ff *FeatureFlags) GetInjectionFailurePolicy() string {
	if ff.getRaw(InjectionFailurePolicyKey) == failPhrase {
		return failPhrase
	}

	return silentPhrase
}

func (ff *FeatureFlags) HasInitSeccomp() bool {
	return ff.getBoolWithDefault(InjectionSeccompKey, true)
}
