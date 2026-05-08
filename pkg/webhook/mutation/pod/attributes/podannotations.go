package attributes

import (
	"encoding/json"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/metadataenrichment"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
)

func (attrs *PodAttributes) ApplyAnnotationsToPod(pod *corev1.Pod) error {
	annotations := attrs.combineForMetadataAnnotations()

	for key, value := range annotations {
		setPodAnnotationIfNotExists(pod, metadataenrichment.Prefix+key, value)
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

	setPodAnnotationIfNotExists(pod, metadataenrichment.Annotation, string(marshaledAnnotations))

	return nil
}

func (attrs *PodAttributes) setWorkloadAnnotations(pod *corev1.Pod) {
	if pod.Annotations == nil {
		pod.Annotations = make(map[string]string)
	}

	pod.Annotations[metadataenrichment.Prefix+K8sWorkloadNameAttr] = attrs.workloadInfo[K8sWorkloadNameAttr]
	pod.Annotations[metadataenrichment.Prefix+K8sWorkloadKindAttr] = attrs.workloadInfo[K8sWorkloadKindAttr]
}

func setPodAnnotationIfNotExists(pod *corev1.Pod, key, value string) {
	if pod.Annotations == nil {
		pod.Annotations = make(map[string]string)
	}

	if _, ok := pod.Annotations[key]; !ok {
		pod.Annotations[key] = value
	}
}
