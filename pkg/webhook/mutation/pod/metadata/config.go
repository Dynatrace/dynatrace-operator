package metadata

import "github.com/Dynatrace/dynatrace-operator/pkg/logd"

const (
	AnnotationPrefix = "metadata-enrichment"
	// AnnotationMetadataEnrichmentInject can be set at pod level to enable/disable metadata-enrichment injection.
	AnnotationInject   = AnnotationPrefix + ".dynatrace.com/inject"
	AnnotationInjected = AnnotationPrefix + ".dynatrace.com/injected"

	// AnnotationWorkloadKind is added to any injected pods when the metadata-enrichment feature is enabled
	AnnotationWorkloadKind = "metadata.dynatrace.com/k8s.workload.kind"
	// AnnotationWorkloadName is added to any injected pods when the metadata-enrichment feature is enabled
	AnnotationWorkloadName = "metadata.dynatrace.com/k8s.workload.name"
)

var (
	log = logd.Get().WithName("metadata-enrichment-pod-common")
)
