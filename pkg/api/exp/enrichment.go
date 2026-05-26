package exp

const (
	EnrichmentEnableAttributesDTKubernetes = FFPrefix + "enable-attributes-dt.kubernetes"
)

func (ff *FeatureFlags) EnableAttributesDTKubernetes() bool {
	defaultVal := !ff.hasPlatformToken

	return ff.getBoolWithDefault(EnrichmentEnableAttributesDTKubernetes, defaultVal)
}
