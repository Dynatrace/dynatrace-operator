package attributes

import (
	"encoding/json"
	"strings"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
)

type ContainerInfos struct {
	// used for container.conf for code modules
	Registry    string `json:"container_image.registry,omitempty"`
	Repository  string `json:"container_image.repository,omitempty"`
	Tag         string `json:"container_image.tags,omitempty"`
	ImageDigest string `json:"container_image.digest,omitempty"`

	// used for metadata enrichment and OTLP exporter auto-config
	ContainerAttributes `json:",omitempty"`
}

func NewContainerInfos(c corev1.Container) *ContainerInfos {
	infos := &ContainerInfos{
		ContainerAttributes: *NewContainerAttributes(c),
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
func (attrs *ContainerInfos) ToJSON() (string, error) {
	jsonAttr, err := json.Marshal(attrs)
	if err != nil {
		return "", errors.WithStack(err)
	}

	return string(jsonAttr), nil
}
