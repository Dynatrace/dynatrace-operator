package container

import (
	k8spod "github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/pod"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
)

func FindContainerInPod(pod corev1.Pod, name string) (*corev1.Container, error) {
	container := FindContainerInPodSpec(&pod.Spec, name)
	if container != nil {
		return container, nil
	}

	podName := k8spod.GetName(pod)

	return nil, errors.Errorf("no container %s found for pod %s", name, podName)
}

func FindContainerInPodSpec(podSpec *corev1.PodSpec, containerName string) *corev1.Container {
	for i := range podSpec.Containers {
		container := &podSpec.Containers[i]
		if container.Name == containerName {
			return container
		}
	}

	return nil
}

func FindInitContainerInPodSpec(podSpec *corev1.PodSpec, containerName string) *corev1.Container {
	for i := range podSpec.InitContainers {
		container := &podSpec.InitContainers[i]
		if container.Name == containerName {
			return container
		}
	}

	return nil
}
