// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package attributes

import (
	"encoding/json"
	"strings"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
)

type Container struct {
	ContainerName string `json:"k8s.container.name,omitempty"`
}

func NewContainerAttributes(c corev1.Container) *Container {
	return &Container{
		ContainerName: c.Name,
	}
}

func (attrs *Container) ToMap() map[string]string {
	combined := make(map[string]string)
	combined[K8sContainerNameAttr] = attrs.ContainerName

	return combined
}

type ContainerInfo struct {
	// used for container.conf for code modules
	Registry    string `json:"container_image.registry,omitempty"`
	Repository  string `json:"container_image.repository,omitempty"`
	Tag         string `json:"container_image.tags,omitempty"`
	ImageDigest string `json:"container_image.digest,omitempty"`

	// used for metadata enrichment and OTLP exporter auto-config
	Container `json:",omitempty"`
}

func NewContainerInfo(c corev1.Container) *ContainerInfo {
	infos := &ContainerInfo{
		Container: *NewContainerAttributes(c),
	}

	registry, repo, found := strings.Cut(c.Image, "/")
	if found {
		infos.Registry = registry
	} else {
		repo = registry
	}

	repo, digest, found := strings.Cut(repo, "@")
	if found {
		infos.ImageDigest = digest
	}

	var tag string

	infos.Repository, tag, found = strings.Cut(repo, ":")
	if found {
		infos.Tag = tag
	}

	return infos
}

// Converts the whole object to a single JSON string used by the bootstrapper
func (c *ContainerInfo) ToJSON() (string, error) {
	jsonAttr, err := json.Marshal(c)
	if err != nil {
		return "", errors.WithStack(err)
	}

	return string(jsonAttr), nil
}
