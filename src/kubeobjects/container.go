package kubeobjects

import (
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
)

func FindContainerInPod(pod corev1.Pod, name string) (*corev1.Container, error) {
	container := FindContainerInPodSpec(&pod.Spec, name)
	if container != nil {
		return container, nil
	}
	podName := pod.Name
	if podName == "" {
		podName = pod.GenerateName
	}
	return nil, errors.Errorf("no container %s found for pod %s", name, podName)
}

func FindContainerInPodSpec(spec *corev1.PodSpec, name string) *corev1.Container {
	for i := range spec.Containers {
		container := &spec.Containers[i]
		if container.Name == name {
			return container
		}
	}
	return nil
}
