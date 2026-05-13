package attributes

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/metadataenrichment"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/objects/k8spod"
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
	json, err := attrs.combineForJSONAnnotation()
	if err != nil {
		return err
	}

	k8spod.SetPodAnnotationIfNotExists(pod, metadataenrichment.Annotation, json)

	return nil
}

func (attrs *PodAttributes) setWorkloadAnnotations(pod *corev1.Pod) {
	if pod.Annotations == nil {
		pod.Annotations = make(map[string]string)
	}

	pod.Annotations[metadataenrichment.Prefix+K8sWorkloadNameAttr] = attrs.workloadInfo[K8sWorkloadNameAttr]
	pod.Annotations[metadataenrichment.Prefix+K8sWorkloadKindAttr] = attrs.workloadInfo[K8sWorkloadKindAttr]
}
