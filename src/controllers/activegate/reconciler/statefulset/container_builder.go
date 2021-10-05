package statefulset

import corev1 "k8s.io/api/core/v1"

type ContainerBuilder interface {
	BuildContainer() corev1.Container
	BuildVolumes() []corev1.Volume
}

type GenericContainer struct {
	StsProperties *statefulSetProperties
}

func NewGenericContainer(stsProperties *statefulSetProperties) *GenericContainer {
	return &GenericContainer{StsProperties: stsProperties}
}
