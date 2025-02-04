package metadata

import (
	"encoding/json"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook"
	corev1 "k8s.io/api/core/v1"
)

func propagateMetadataAnnotations(request *dtwebhook.MutationRequest) {
	copyMetadataFromNamespace(request.Pod, request.Namespace, request.DynaKube)
	addMetadataToInitEnv(request.Pod, request.InstallContainer)
}

func copyMetadataFromNamespace(pod *corev1.Pod, namespace corev1.Namespace, dk dynakube.DynaKube) {
	copyMetadataAccordingToCustomRules(pod, namespace, dk)
	copyMetadataAccordingToPrefix(pod, namespace)
}

func copyMetadataAccordingToPrefix(pod *corev1.Pod, namespace corev1.Namespace) {
	for key, value := range namespace.Annotations {
		if strings.HasPrefix(key, dynakube.MetadataPrefix) {
			setPodAnnotationIfNotExists(pod, key, value)
		}
	}
}

func copyMetadataAccordingToCustomRules(pod *corev1.Pod, namespace corev1.Namespace, dk dynakube.DynaKube) {
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

func addMetadataToInitEnv(pod *corev1.Pod, installContainer *corev1.Container) {
	metadataAnnotations := map[string]string{}

	for key, value := range pod.Annotations {
		if !strings.HasPrefix(key, dynakube.MetadataPrefix) {
			continue
		}

		split := strings.Split(key, dynakube.MetadataPrefix)
		metadataAnnotations[split[1]] = value
	}

	workloadAnnotationsJson, _ := json.Marshal(metadataAnnotations)
	installContainer.Env = append(installContainer.Env,
		corev1.EnvVar{
			Name: consts.EnrichmentWorkloadAnnotationsEnv, Value: string(workloadAnnotationsJson)},
	)
}
