package attributes

import (
	"encoding/json"
	"errors"
	"maps"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/metadataenrichment"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator"
	corev1 "k8s.io/api/core/v1"
)

const (
	// AnnotationWorkloadKind is added to any injected pods when the metadata-enrichment feature is enabled
	AnnotationWorkloadKind = "metadata.dynatrace.com/k8s.workload.kind"
	// AnnotationWorkloadName is added to any injected pods when the metadata-enrichment feature is enabled
	AnnotationWorkloadName = "metadata.dynatrace.com/k8s.workload.name"
)

//TODO: why are there three functions when this one would suffice
func (attrs *PodAttributes) SetMetadataAnnotations(request *dtwebhook.MutationRequest) {
	attrs.SetWorkloadMetadataAnnotations(request.Pod)

	err := attrs.PropagateMetadataAnnotations(request.Pod)
	if err != nil {
		logd.FromContext(request.Context).Error(err, "failed to propagate metadata annotations", "pod", request.Pod, "namespace", request.Namespace)
	}
}

func (attrs *PodAttributes) PropagateMetadataAnnotations(pod *corev1.Pod) error {
	annotations := make(map[string]string)
	maps.Copy(annotations, attrs.namespaceAnnotations)
	maps.Copy(annotations, attrs.rulesPropagate)
	maps.Copy(annotations, attrs.workloadInfo)

	for key, value := range annotations {
		setPodAnnotationIfNotExists(pod, metadataenrichment.Prefix+key, value)
	}

	return setPodMetadataJSONAnnotation(pod, annotations)
}

func (attrs *PodAttributes) SetWorkloadMetadataAnnotations(pod *corev1.Pod) {
	if pod.Annotations == nil {
		pod.Annotations = make(map[string]string)
	}

	for key, value := range attrs.workloadInfo {
		pod.Annotations[metadataenrichment.Prefix+key] = value
	}
}

func setPodMetadataJSONAnnotation(pod *corev1.Pod, annotations map[string]string) error {
	marshaledAnnotations, err := json.Marshal(annotations)
	if err != nil {
		return errors.Join(errors.New("could not marshal metadata annoations to JSON"), err)
	}

	setPodAnnotationIfNotExists(pod, metadataenrichment.Annotation, string(marshaledAnnotations))
	return nil
}

func setPodAnnotationIfNotExists(pod *corev1.Pod, key, value string) {
	if pod.Annotations == nil {
		pod.Annotations = make(map[string]string)
	}

	if _, ok := pod.Annotations[key]; !ok {
		pod.Annotations[key] = value
	}
}
