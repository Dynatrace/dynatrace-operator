package exp

const (
	WebhookEnableAttributesDtKubernetes = FFPrefix + "enable-attributes-dt.kubernetes"
)

func (ff *FeatureFlags) EnableAttributesDtKubernetes() bool {
	return ff.getBoolWithDefault(WebhookEnableAttributesDtKubernetes, false)
}
