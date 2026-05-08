package attributes

import (
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/metadataenrichment"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator"
	corev1 "k8s.io/api/core/v1"
)

func (attrs *PodAttributes) readMetadataAnnotations(request mutator.BaseRequest) {
	attrs.readFromEnrichmentRules(request.Namespace, request.DynaKube)
	attrs.readNamespaceAnnotationAttributes(request.Namespace)
	attrs.readPodAnnotationAttributes(*request.Pod)
}

// collect attributes from pod and namespace "metadata.dynatrace.com/" annotations
func (attrs *PodAttributes) readNamespaceAnnotationAttributes(namespace corev1.Namespace) {
	for key, value := range namespace.Annotations {
		if after, ok := strings.CutPrefix(key, metadataenrichment.Prefix); ok {
			attrs.namespaceAnnotations[after] = value
		}
	}
}

// collect attributes from pod and namespace "metadata.dynatrace.com/" annotations
func (attrs *PodAttributes) readPodAnnotationAttributes(pod corev1.Pod) {
	// pod annotations take precedence over namespace annotations
	for key, value := range pod.Annotations {
		if after, ok := strings.CutPrefix(key, metadataenrichment.Prefix); ok {
			attrs.podAnnotations[after] = value
		}
	}
}

func (attrs *PodAttributes) readFromEnrichmentRules(namespace corev1.Namespace, dk dynakube.DynaKube) {
	for _, rule := range dk.Status.MetadataEnrichment.Rules {
		var (
			valueFromNamespace string
			exists             bool
		)

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
