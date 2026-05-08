package attributes

import (
	"encoding/json"
	"maps"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/metadataenrichment"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
)

const (
	// AnnotationWorkloadKind is added to any injected pods when the metadata-enrichment feature is enabled
	AnnotationWorkloadKind = "metadata.dynatrace.com/k8s.workload.kind"
	// AnnotationWorkloadName is added to any injected pods when the metadata-enrichment feature is enabled
	AnnotationWorkloadName = "metadata.dynatrace.com/k8s.workload.name"
)

func (attrs *PodAttributes) ApplyAnnotationsToPod(pod *corev1.Pod) error {
	annotations := make(map[string]string)

	// make sure we use the same precedence as in combine()
	maps.Copy(annotations, attrs.workloadInfo)
	maps.Copy(annotations, attrs.rulesPropagate)
	maps.Copy(annotations, attrs.namespaceAnnotations)

	// workload info uses the OTEL attribute name directly as the annotation key (no metadata prefix)
	for key, value := range annotations {
		setPodAnnotationIfNotExists(pod, metadataenrichment.Prefix+key, value)
	}

	return attrs.setPodMetadataJSONAnnotation(pod)
}

func setPodAnnotationIfNotExists(pod *corev1.Pod, key, value string) {
	if pod.Annotations == nil {
		pod.Annotations = make(map[string]string)
	}
	if _, ok := pod.Annotations[key]; !ok {
		pod.Annotations[key] = value
	}
}

func (attrs *PodAttributes) setPodMetadataJSONAnnotation(pod *corev1.Pod) error {
	annotations := make(map[string]string)

	// make sure we use the same precedence as in combine()
	maps.Copy(annotations, attrs.rules)
	maps.Copy(annotations, attrs.rulesPropagate)
	maps.Copy(annotations, attrs.namespaceAnnotations)
	maps.Copy(annotations, attrs.podAnnotations)

	marshaledAnnotations, err := json.Marshal(annotations)
	if err != nil {
		return errors.WithMessage(errors.WithStack(err), "could not marshal metadata annotations to JSON")
	}

	setPodAnnotationIfNotExists(pod, metadataenrichment.Annotation, string(marshaledAnnotations))
	return nil
}
