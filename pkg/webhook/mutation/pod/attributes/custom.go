package attributes

import (
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/metadataenrichment"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator"
	corev1 "k8s.io/api/core/v1"
)

func (attrs *PodAttributes) GetMetadataAnnotations(request mutator.BaseRequest) {
	attrs.getFromEnrichmentRules(request.Namespace, request.DynaKube)
	attrs.getNamespaceAnnotationAttributes(request.Namespace)
	attrs.getPodAnnotationAttributes(*request.Pod)
}

// collect attributes from pod and namespace "metadata.dynatrace.com/" annotations
func (attrs *PodAttributes) getNamespaceAnnotationAttributes(namespace corev1.Namespace) {
	for key, value := range namespace.Annotations {
		if strings.HasPrefix(key, metadataenrichment.Prefix) {
			attrs.namespaceAnnotations[strings.TrimPrefix(key, metadataenrichment.Prefix)] = value
		}
	}
}

// collect attributes from pod and namespace "metadata.dynatrace.com/" annotations
func  (attrs *PodAttributes) getPodAnnotationAttributes(pod corev1.Pod) {
	// pod annotations take precedence over namespace annotations
	for key, value := range pod.Annotations {
		if strings.HasPrefix(key, metadataenrichment.Prefix) {
			attrs.podAnnotations[strings.TrimPrefix(key, metadataenrichment.Prefix)] = value
		}
	}
}

func  (attrs *PodAttributes) getFromEnrichmentRules(namespace corev1.Namespace, dk dynakube.DynaKube) {
	for _, rule := range dk.Status.MetadataEnrichment.Rules {
		var valueFromNamespace string
		var exists bool

		switch rule.Type {
		case metadataenrichment.LabelRule:
			valueFromNamespace, exists = namespace.Labels[rule.Source]
		case metadataenrichment.AnnotationRule:
			valueFromNamespace, exists = namespace.Annotations[rule.Source]
		}

		if exists {
			if len(rule.Target) > 0 {
				attrs.rulesPropagate[rule.Target] = valueFromNamespace
			} else {

				attrs.rules[metadataenrichment.GetEmptyTargetEnrichmentKey(string(rule.Type), rule.Source)] = valueFromNamespace
			}
		}
	}
}

func RemoveMetadataPrefix(annotations map[string]string) map[string]string {
	annotationsWithoutPrefix := make(map[string]string)

	for key, value := range annotations {
		if strings.HasPrefix(key, metadataenrichment.Prefix) {
			split := strings.Split(key, metadataenrichment.Prefix)
			annotationsWithoutPrefix[split[1]] = value
		} else {
			annotationsWithoutPrefix[key] = value
		}
	}

	return annotationsWithoutPrefix
}
