package attributes

import (
	"encoding/json"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/metadataenrichment"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/objects/k8spod"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
)

const (
	// AnnotationWorkloadKind is added to any injected pods when the metadata-enrichment feature is enabled
	AnnotationWorkloadKind = metadataenrichment.Prefix + K8sWorkloadKindAttr
	// AnnotationWorkloadName is added to any injected pods when the metadata-enrichment feature is enabled
	AnnotationWorkloadName = metadataenrichment.Prefix + K8sWorkloadNameAttr
)

func (attrs *PodAttributes) ApplyAnnotationsToPod(pod *corev1.Pod) error {
	annotations := attrs.combineForMetadataAnnotations()

	for key, value := range annotations {
		k8spod.SetPodAnnotationIfNotExists(pod, metadataenrichment.Prefix+key, value)
	}

	// set workload annotations no matter what
	attrs.setWorkloadAnnotations(pod)

	return attrs.setPodMetadataJSONAnnotation(pod)
}

func (attrs *PodAttributes) setPodMetadataJSONAnnotation(pod *corev1.Pod) error {
	annotations := attrs.combineForJSONAnnotation()

	marshaledAnnotations, err := json.Marshal(annotations)
	if err != nil {
		return errors.WithMessage(errors.WithStack(err), "could not marshal metadata annotations to JSON")
	}

	k8spod.SetPodAnnotationIfNotExists(pod, metadataenrichment.Annotation, string(marshaledAnnotations))

	return nil
}

func (attrs *PodAttributes) setWorkloadAnnotations(pod *corev1.Pod) {
	if pod.Annotations == nil {
		pod.Annotations = make(map[string]string)
	}

	pod.Annotations[metadataenrichment.Prefix+K8sWorkloadNameAttr] = attrs.workloadInfo[K8sWorkloadNameAttr]
	pod.Annotations[metadataenrichment.Prefix+K8sWorkloadKindAttr] = attrs.workloadInfo[K8sWorkloadKindAttr]
}
