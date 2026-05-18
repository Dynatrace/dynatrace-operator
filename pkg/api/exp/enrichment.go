package exp

const (
	EnrichmentEnableAttributesDTKubernetes = FFPrefix + "enable-attributes-dt.kubernetes"
)

// EnableAttributesDTKubernetes returns whether dt.kubernetes.* attributes should be added.
// When using a classic token (apiToken) this defaults to true (opt-out behavior).
// When using a platform token this defaults to false (opt-in behavior).
func (ff *FeatureFlags) EnableAttributesDTKubernetes() bool {
	defaultVal := !ff.HasPlatformToken()

	return ff.getBoolWithDefault(EnrichmentEnableAttributesDTKubernetes, defaultVal)
}
