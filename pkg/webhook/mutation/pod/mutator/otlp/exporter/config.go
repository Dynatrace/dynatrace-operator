package exporter

const (
	AnnotationPrefix = "otlp-exporter-configuration"
	// AnnotationInject can be set at pod level to enable/disable OTLP exporter configuration injection.
	AnnotationInject   = AnnotationPrefix + ".dynatrace.com/inject"
	AnnotationInjected = AnnotationPrefix + ".dynatrace.com/injected"
	AnnotationReason   = AnnotationPrefix + ".dynatrace.com/reason"

	CouldNotGetIngestEndpointReason = "IngestEndpointUnavailable"
)
