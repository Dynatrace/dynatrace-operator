package exp

const (
	OtelcDebugExporterVerbosity          = FFPrefix + "otelc-debug-exporter-verbosity"
	OtelcDebugExporterSamplingInitial    = FFPrefix + "otelc-debug-exporter-sampling-initial"
	OtelcDebugExporterSamplingThereafter = FFPrefix + "otelc-debug-exporter-sampling-thereafter"
)

func (ff *FeatureFlags) IsOtelDebugExporterEnabled() bool {
	return ff.getRaw(OtelcDebugExporterVerbosity) != ""
}

func (ff *FeatureFlags) GetOtelDebugExporterVerbosity() string {
	return ff.getRaw(OtelcDebugExporterVerbosity)
}

func (ff *FeatureFlags) GetOtelDebugExporterSamplingInitial() int {
	return ff.getIntWithDefault(OtelcDebugExporterSamplingInitial, 5)
}

func (ff *FeatureFlags) GetOtelDebugExporterSamplingThereafter() int {
	return ff.getIntWithDefault(OtelcDebugExporterSamplingThereafter, 200)
}
