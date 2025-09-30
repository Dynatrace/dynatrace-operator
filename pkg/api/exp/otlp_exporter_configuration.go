package exp

const (
	OTLPExporterConfigurationKey = FFPrefix + "otlp-exporter-configuration"
)

// IsOTLPExporterConfiguration controls whether the automatic configuration of the OTLP exporter via environment variables
// for user containers is enabled.
func (ff *FeatureFlags) IsOTLPExporterConfiguration() bool {
	return ff.getBoolWithDefault(OTLPExporterConfigurationKey, false)
}
