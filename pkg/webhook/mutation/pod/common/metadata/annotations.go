package metadata

import (
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
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
	for _, rule := range dk.Status.MetadataEnrichment.Rules {
		if rule.Target == "" {
			log.Info("rule without target set found, ignoring", "source", rule.Source, "type", rule.Type)

			continue
		}

		var valueFromNamespace string

		var exists bool

		switch rule.Type {
		case dynakube.EnrichmentLabelRule:
			valueFromNamespace, exists = namespace.Labels[rule.Source]
		case dynakube.EnrichmentAnnotationRule:
			valueFromNamespace, exists = namespace.Annotations[rule.Source]
		}

		if exists {
			setPodAnnotationIfNotExists(pod, rule.ToAnnotationKey(), valueFromNamespace)
		}
	}
}

func setPodAnnotationIfNotExists(pod *corev1.Pod, key, value string) {
	if pod.Annotations == nil {
		pod.Annotations = make(map[string]string)
	}

	if _, ok := pod.Annotations[key]; !ok {
		pod.Annotations[key] = value
	}
}
