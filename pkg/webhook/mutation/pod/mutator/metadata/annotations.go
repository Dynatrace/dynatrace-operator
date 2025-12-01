package metadata

import (
	"encoding/json"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/metadataenrichment"
	"golang.org/x/exp/maps"
	corev1 "k8s.io/api/core/v1"
)

func CopyMetadataFromNamespace(pod *corev1.Pod, namespace corev1.Namespace, dk dynakube.DynaKube) map[string]string {
	copiedCustomRuleAnnotations := copyAccordingToCustomRules(pod, namespace, dk)
	copiedPrefixAnnotations := copyAccordingToPrefix(pod, namespace)

	maps.Copy(copiedCustomRuleAnnotations, copiedPrefixAnnotations)

	copiedCustomRuleAnnotations = removeMetadataPrefix(copiedCustomRuleAnnotations)
	setPodMetadataJSONAnnotation(pod, copiedCustomRuleAnnotations)

	return copiedCustomRuleAnnotations
}

func copyAccordingToPrefix(pod *corev1.Pod, namespace corev1.Namespace) map[string]string {
	metadataAnnotations := make(map[string]string)

	for key, value := range namespace.Annotations {
		if strings.HasPrefix(key, metadataenrichment.Prefix) {
			_ = setPodAnnotationIfNotExists(pod, key, value)
		}
	}

	for key, value := range pod.Annotations {
		if strings.HasPrefix(key, metadataenrichment.Prefix) {
			metadataAnnotations[key] = value
		}
	}

	return metadataAnnotations
}

func copyAccordingToCustomRules(pod *corev1.Pod, namespace corev1.Namespace, dk dynakube.DynaKube) map[string]string {
	copiedAnnotations := make(map[string]string)

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
				added := setPodAnnotationIfNotExists(pod, rule.ToAnnotationKey(), valueFromNamespace)
				if added {
					copiedAnnotations[rule.ToAnnotationKey()] = valueFromNamespace
				}
			} else {
				copiedAnnotations[metadataenrichment.GetEmptyTargetEnrichmentKey(string(rule.Type), rule.Source)] = valueFromNamespace
			}
		}
	}

	return copiedAnnotations
}

func setPodMetadataJSONAnnotation(pod *corev1.Pod, annotations map[string]string) {
	marshaledAnnotations, err := json.Marshal(annotations)
	if err != nil {
		log.Error(err, "failed to marshal annotations to map", "annotations", annotations)
	}

	_ = setPodAnnotationIfNotExists(pod, metadataenrichment.Annotation, string(marshaledAnnotations))
}

func setPodAnnotationIfNotExists(pod *corev1.Pod, key, value string) bool {
	if pod.Annotations == nil {
		pod.Annotations = make(map[string]string)
	}

	if _, ok := pod.Annotations[key]; !ok {
		pod.Annotations[key] = value

		return true
	}

	return false
}

func removeMetadataPrefix(annotations map[string]string) map[string]string {
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
