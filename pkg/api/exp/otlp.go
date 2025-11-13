package exp

const (
	OTLPInjectionSetNoProxy = FFPrefix + "otlp-exporter-configuration-set-no-proxy"
)

// IsOTLPInjectionSetNoProxy is a feature flag to enable injecting additional environment variables based on user labels.
func (ff *FeatureFlags) IsOTLPInjectionSetNoProxy() bool {
	return ff.getBoolWithDefault(OTLPInjectionSetNoProxy, true)
}
