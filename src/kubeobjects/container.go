package kubeobjects

import (
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
)

func GetContainer(pod corev1.Pod, name string) (*corev1.Container, error) {
	for i := range pod.Spec.Containers {
		container := &pod.Spec.Containers[i]
		if container.Name == name {
			return container, nil
		}
	}
	return nil, errors.Errorf("no container found for pod %s with name %s", pod.Name, name)
}
