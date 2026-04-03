package exp

const (
	EnrichmentEnableAttributesDTKubernetes = FFPrefix + "enable-attributes-dt.kubernetes"
)

func (ff *FeatureFlags) EnableAttributesDTKubernetes() bool {
	return ff.getBoolWithDefault(EnrichmentEnableAttributesDTKubernetes, true)
}
