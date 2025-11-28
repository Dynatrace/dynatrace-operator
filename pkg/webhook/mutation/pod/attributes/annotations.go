package attributes

import (
	"encoding/json"
	podattr "github.com/Dynatrace/dynatrace-bootstrapper/cmd/configure/attributes/pod"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/workload"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/metadataenrichment"
	"golang.org/x/exp/maps"
	corev1 "k8s.io/api/core/v1"
)

func GetNamespaceAttributes(attrs podattr.Attributes, request *mutator.BaseRequest) podattr.Attributes {
	copiedMetadataAnnotations := copyMetadataFromNamespace(request)
	if copiedMetadataAnnotations == nil {
		log.Info("copied metadata annotations from namespace is empty, propagation is not necessary")
	} else {
		if attrs.UserDefined == nil {
			attrs.UserDefined = make(map[string]string)
		}
		maps.Copy(attrs.UserDefined, copiedMetadataAnnotations)
	}

	return attrs
}

func copyMetadataFromNamespace(request *mutator.BaseRequest) map[string]string {
	copiedCustomRuleAnnotations := copyAccordingToCustomRules(request.Pod, request.Namespace, request.DynaKube)
	copiedPrefixAnnotations := copyAccordingToPrefix(request.Pod, request.Namespace)

	maps.Copy(copiedCustomRuleAnnotations, copiedPrefixAnnotations)

	copiedCustomRuleAnnotations = removeMetadataPrefix(copiedCustomRuleAnnotations)
	setPodMetadataJSONAnnotation(request.Pod, copiedCustomRuleAnnotations)

	return copiedCustomRuleAnnotations
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

func copyAccordingToPrefix(pod *corev1.Pod, namespace corev1.Namespace) map[string]string {
	metadataAnnotations := make(map[string]string)

	// first propagate metadata annotations from namespace to pod
	for key, value := range namespace.Annotations {
		if strings.HasPrefix(key, metadataenrichment.Prefix) {
			_ = setPodAnnotationIfNotExists(pod, key, value)
		}
	}

	// then collect all metadata annotations from pod in one go
	for key, value := range pod.Annotations {
		if strings.HasPrefix(key, metadataenrichment.Prefix) {
			metadataAnnotations[key] = value
		}
	}

	return metadataAnnotations
}

func setPodMetadataJSONAnnotation(pod *corev1.Pod, annotations map[string]string) {
	marshaledAnnotations, err := json.Marshal(annotations)
	if err != nil {
		log.Error(err, "failed to marshal annotations to map", "annotations", annotations)
	}

	_ = setPodAnnotationIfNotExists(pod, metadataenrichment.Annotation, string(marshaledAnnotations))
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
