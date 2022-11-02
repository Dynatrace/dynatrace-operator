package image

import (
	"github.com/containers/image/v5/docker"
	"github.com/containers/image/v5/docker/reference"
	"github.com/containers/image/v5/transports"
	"github.com/pkg/errors"
)

const (
	defaultRegistry = "registry-1.docker.io"
	defaultVersion  = "latest"
)

type Components struct {
	Registry  string
	Image     string
	Version   string
	reference reference.Named
}

func ComponentsFromUri(imageUri string) (Components, error) {
	transport := transports.Get(docker.Transport.Name())
	transportReference, err := transport.ParseReference("//" + imageUri)

	if err != nil {
		return Components{}, errors.WithStack(err)
	}

	dockerReference := transportReference.DockerReference()
	registry := reference.Domain(dockerReference)
	image := reference.Path(dockerReference)
	version := defaultVersion
	taggedReference, isTaggedReference := dockerReference.(reference.Tagged)
	digestReference, isDigestReference := dockerReference.(reference.Canonical)

	if isTaggedReference {
		version = taggedReference.Tag()
	} else if isDigestReference {
		version = digestReference.Digest().String()
	}
	if registry == "" {
		registry = defaultRegistry
	}
	if image == "" {
		return Components{}, errors.New("image name is missing from image uri")
	}

	return Components{
		Registry:  registry,
		Image:     image,
		Version:   version,
		reference: dockerReference,
	}, nil
}

func (components Components) VersionUrlPostfix() string {
	version := ""
	taggedReference, isTaggedReference := components.reference.(reference.Tagged)
	digestReference, isDigestReference := components.reference.(reference.Canonical)

	if isTaggedReference {
		version = ":" + taggedReference.Tag()
	} else if isDigestReference {
		version = "@" + digestReference.Digest().String()
	}

	return version
}
