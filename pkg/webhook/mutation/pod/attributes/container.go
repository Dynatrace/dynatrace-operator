package attributes

import (
	v1 "k8s.io/api/core/v1"
)

const K8sContainerNameAttr = "k8s.container.name"

type ContainerAttributes struct {
	ContainerName string `json:"k8s.container.name,omitempty"`
}

func NewContainerAttributes(c v1.Container) *ContainerAttributes {
	return &ContainerAttributes{
		ContainerName: c.Name,
	}
}

func (attrs *ContainerAttributes) ToMap() map[string]string {
	combined := make(map[string]string)
	combined[K8sContainerNameAttr] = attrs.ContainerName

	return combined
}
