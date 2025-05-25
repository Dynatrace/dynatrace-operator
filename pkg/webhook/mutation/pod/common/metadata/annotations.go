package metadata

import (
	"encoding/json"
	"github.com/vladimirvivien/gexe/str"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	corev1 "k8s.io/api/core/v1"
)

func CopyMetadataFromNamespace(pod *corev1.Pod, namespace corev1.Namespace, dk dynakube.DynaKube) map[string]string {
	copiedCustomRuleAnnotations := copyAccordingToCustomRules(pod, namespace, dk)
	copiedPrefixAnnotations := copyAccordingToPrefix(pod, namespace)

	for k, v := range copiedPrefixAnnotations {
		if _, ok := copiedCustomRuleAnnotations[k]; !ok {
			copiedCustomRuleAnnotations[k] = v
		}
	}

	setMetadataAnnotationValue(pod, copiedCustomRuleAnnotations) //json

	return copiedCustomRuleAnnotations
}

func copyAccordingToPrefix(pod *corev1.Pod, namespace corev1.Namespace) map[string]string {
	addedAnnotations := make(map[string]string)
	for key, value := range namespace.Annotations {
		if strings.HasPrefix(key, dynakube.MetadataPrefix) {
			added := setPodAnnotationIfNotExists(pod, key, value)

			if added {
				addedAnnotations[key] = value
			}
		}
	}
	return addedAnnotations
}

func copyAccordingToCustomRules(pod *corev1.Pod, namespace corev1.Namespace, dk dynakube.DynaKube) map[string]string {
	copiedAnnotations := make(map[string]string)
	for _, rule := range dk.Status.MetadataEnrichment.Rules {
		var valueFromNamespace string
		var exists bool

		switch rule.Type {
		case dynakube.EnrichmentLabelRule:
			valueFromNamespace, exists = namespace.Labels[rule.Source]
		case dynakube.EnrichmentAnnotationRule:
			valueFromNamespace, exists = namespace.Annotations[rule.Source]
		}

		if exists {
			if str.IsEmpty(rule.Target) { // Empty target rules are not copied as a single annotation but bulk into the json annotation
				copiedAnnotations[getEmptyTargetEnrichmentKey(string(rule.Type), rule.Source)] = valueFromNamespace
			} else {
				added := setPodAnnotationIfNotExists(pod, rule.ToAnnotationKey(), valueFromNamespace)
				if added {
					copiedAnnotations[rule.ToAnnotationKey()] = valueFromNamespace
				}
			}
		}
	}
	return copiedAnnotations
}

func setMetadataAnnotationValue(pod *corev1.Pod, annotations map[string]string) {
	metadataAnnotations := make(map[string]string)
	for key, value := range annotations {
		// Annotations added to the json must not have metadata.dynatrace.com/ prefix
		if strings.HasPrefix(key, dynakube.MetadataPrefix) {
			split := strings.Split(key, dynakube.MetadataPrefix)
			metadataAnnotations[split[1]] = value
		} else {
			metadataAnnotations[key] = value
		}
	}

	marshaledAnnotations, err := json.Marshal(metadataAnnotations)
	if err != nil {
		log.Error(err, "failed to marshal annotations to map", "annotations", annotations)
	}

	setPodAnnotationIfNotExists(pod, dynakube.MetadataAnnotation, string(marshaledAnnotations))
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

func getEmptyTargetEnrichmentKey(metadataType, key string) string {
	return dynakube.EnrichmentNamespaceKey + strings.ToLower(metadataType) + "." + key
}
