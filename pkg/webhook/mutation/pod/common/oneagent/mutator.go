package oneagent

import (
	maputils "github.com/Dynatrace/dynatrace-operator/pkg/util/map"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

const (
	CSIVolumeType       = "csi"
	EphemeralVolumeType = "ephemeral"
)

func IsEnabled(request *dtwebhook.BaseRequest) bool {
	enabledOnPod := maputils.GetFieldBool(request.Pod.Annotations, AnnotationInject, request.DynaKube.FF().IsAutomaticInjection())
	enabledOnDynakube := request.DynaKube.OneAgent().GetNamespaceSelector() != nil

	matchesNamespaceSelector := true // if no namespace selector is configured, we just pass set this to true

	if request.DynaKube.OneAgent().GetNamespaceSelector().Size() > 0 {
		selector, _ := metav1.LabelSelectorAsSelector(request.DynaKube.OneAgent().GetNamespaceSelector())

		matchesNamespaceSelector = selector.Matches(labels.Set(request.Namespace.Labels))
	}

	return matchesNamespaceSelector && enabledOnPod && enabledOnDynakube
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

func SetNotInjectedAnnotations(pod *corev1.Pod, reason string) {
	if pod.Annotations == nil {
		pod.Annotations = make(map[string]string)
	}

	pod.Annotations[AnnotationInjected] = "false"
	pod.Annotations[AnnotationReason] = reason
}
