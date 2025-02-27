package sharedoneagent

import (
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook"
	corev1 "k8s.io/api/core/v1"
)

func SetInjectedAnnotation(pod *corev1.Pod) {
	if pod.Annotations == nil {
		pod.Annotations = make(map[string]string)
	}

	pod.Annotations[dtwebhook.AnnotationOneAgentInjected] = "true"
}

func SetNotInjectedAnnotations(pod *corev1.Pod, reason string) {
	if pod.Annotations == nil {
		pod.Annotations = make(map[string]string)
	}

	pod.Annotations[dtwebhook.AnnotationOneAgentInjected] = "false"
	pod.Annotations[dtwebhook.AnnotationOneAgentReason] = reason
}
