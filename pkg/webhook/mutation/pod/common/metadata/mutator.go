package metadata

import (
	maputils "github.com/Dynatrace/dynatrace-operator/pkg/util/map"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

func IsEnabled(request *dtwebhook.BaseRequest) bool {
	enabledOnPod := maputils.GetFieldBool(request.Pod.Annotations, AnnotationInject,
		request.DynaKube.FF().IsAutomaticInjection())
	enabledOnDynakube := request.DynaKube.MetadataEnrichmentEnabled()

	matchesNamespace := true // if no namespace selector is configured, we just pass set this to true

	if request.DynaKube.MetadataEnrichmentNamespaceSelector().Size() > 0 {
		selector, _ := metav1.LabelSelectorAsSelector(request.DynaKube.MetadataEnrichmentNamespaceSelector())

		matchesNamespace = selector.Matches(labels.Set(request.Namespace.Labels))
	}

	return matchesNamespace && enabledOnPod && enabledOnDynakube
}

func IsInjected(request *dtwebhook.BaseRequest) bool {
	return maputils.GetFieldBool(request.Pod.Annotations, AnnotationInjected, false)
}

func SetInjectedAnnotation(pod *corev1.Pod) {
	if pod.Annotations == nil {
		pod.Annotations = make(map[string]string)
	}

	pod.Annotations[AnnotationInjected] = "true"
}

func SetWorkloadAnnotations(pod *corev1.Pod, workload *WorkloadInfo) {
	if pod.Annotations == nil {
		pod.Annotations = make(map[string]string)
	}

	pod.Annotations[AnnotationWorkloadKind] = workload.Kind
	pod.Annotations[AnnotationWorkloadName] = workload.Name
}
