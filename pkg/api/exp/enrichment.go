package exp

const (
	EnrichmentEnableAttributesDtKubernetes = FFPrefix + "enable-attributes-dt.kubernetes"
)

func (ff *FeatureFlags) EnableAttributesDtKubernetes() bool {
	return ff.getBoolWithDefault(EnrichmentEnableAttributesDtKubernetes, false)
}
