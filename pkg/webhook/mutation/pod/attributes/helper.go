package attributes

import corev1 "k8s.io/api/core/v1"

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
