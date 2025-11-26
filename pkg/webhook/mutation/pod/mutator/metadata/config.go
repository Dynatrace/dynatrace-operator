package metadata

import "github.com/Dynatrace/dynatrace-operator/pkg/logd"

const (
	AnnotationPrefix = "metadata-enrichment"
	// AnnotationMetadataEnrichmentInject can be set at pod level to enable/disable metadata-enrichment injection.
	AnnotationInject   = AnnotationPrefix + ".dynatrace.com/inject"
	AnnotationInjected = AnnotationPrefix + ".dynatrace.com/injected"
	AnnotationReason   = AnnotationPrefix + ".dynatrace.com/reason"

	OwnerLookupFailedReason = "OwnerLookupFailed"
)

var (
	log = logd.Get().WithName("metadata-enrichment-pod-common")
)
