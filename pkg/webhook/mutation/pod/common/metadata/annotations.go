package metadata

import (
	"encoding/json"
	"github.com/vladimirvivien/gexe/str"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta4/dynakube"
	corev1 "k8s.io/api/core/v1"
)

func CopyMetadataFromNamespace(pod *corev1.Pod, namespace corev1.Namespace, dk dynakube.DynaKube) {
	copyAccordingToCustomRules(pod, namespace, dk)
	copyAccordingToPrefix(pod, namespace)
}

func copyAccordingToPrefix(pod *corev1.Pod, namespace corev1.Namespace) {
	for key, value := range namespace.Annotations {
		if strings.HasPrefix(key, dynakube.MetadataPrefix) {
			setPodAnnotationIfNotExists(pod, key, value)
		}
	}
}

func copyAccordingToCustomRules(pod *corev1.Pod, namespace corev1.Namespace, dk dynakube.DynaKube) {
	emptyTargetValues := make(map[string]string)
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
			if str.IsEmpty(rule.Target) {
				emptyTargetValues[getEmptyTargetEnrichmentKey(string(rule.Type), rule.Source)] = valueFromNamespace
			} else {
				setPodAnnotationIfNotExists(pod, rule.ToAnnotationKey(), valueFromNamespace)
			}
		}
	}

	if len(emptyTargetValues) > 0 {
		setEmptyTargetValuesToPodAnnotations(pod, emptyTargetValues)
	}
}

func setEmptyTargetValuesToPodAnnotations(pod *corev1.Pod, emptyTargetValues map[string]string) {
	marshaledEmptyTargetValues, err := json.Marshal(emptyTargetValues)
	if err != nil {
		log.Error(err, "failed to marshal annotations to map", "annotations", emptyTargetValues)
	}

	setPodAnnotationIfNotExists(pod, dynakube.MetadataAnnotation, string(marshaledEmptyTargetValues))

}

func setPodAnnotationIfNotExists(pod *corev1.Pod, key, value string) {
	if pod.Annotations == nil {
		pod.Annotations = make(map[string]string)
	}

	if _, ok := pod.Annotations[key]; !ok {
		pod.Annotations[key] = value
	}
}

func getEmptyTargetEnrichmentKey(metadataType string, key string) string {
	return "k8s.namespace." + strings.ToLower(metadataType) + "." + key
}
