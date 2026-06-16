package attributes

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/metadataenrichment"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/objects/k8spod"
	corev1 "k8s.io/api/core/v1"
)

func (attrs *Pod) ApplyJSONAnnotationToPod(pod *corev1.Pod) error {
	json, err := attrs.combineForJSONAnnotation()
	if err != nil {
		return err
	}

	k8spod.SetAnnotationIfNotExists(pod, metadataenrichment.Annotation, json)

	return nil
}
