package attributes

import (
	containerattr "github.com/Dynatrace/dynatrace-bootstrapper/cmd/configure/attributes/container"
	"strings"
)

func createImageInfo(imageURI string) containerattr.ImageInfo { // TODO: move to bootstrapper repo
	// can't use the name.ParseReference() as that will fill in some defaults if certain things are defined, but we want to preserve the original string value, without any modification. Tried it with a regexp, was worse.
	imageInfo := containerattr.ImageInfo{}

	registry, repo, found := strings.Cut(imageURI, "/")
	if found {
		imageInfo.Registry = registry
	} else {
		repo = registry
	}

	repo, digest, found := strings.Cut(repo, "@")
	if found {
		imageInfo.ImageDigest = digest
	}

	var tag string

	imageInfo.Repository, tag, found = strings.Cut(repo, ":")
	if found {
		imageInfo.Tag = tag
	}

	return imageInfo
}
