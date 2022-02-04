package kubeobjects

import corev1 "k8s.io/api/core/v1"

type ContainerBuilder interface {
	BuildContainer() corev1.Container
	BuildVolumes() []corev1.Volume
}
