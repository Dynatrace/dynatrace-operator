package metadata

const (
	AnnotationPrefix = "metadata-enrichment"
	// AnnotationMetadataEnrichmentInject can be set at pod level to enable/disable metadata-enrichment injection.
	AnnotationInject   = AnnotationPrefix + ".dynatrace.com/inject"
	AnnotationInjected = AnnotationPrefix + ".dynatrace.com/injected"
	AnnotationReason   = AnnotationPrefix + ".dynatrace.com/reason"

	OwnerLookupFailedReason = "OwnerLookupFailed"
)
