// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package oci

import (
	"strings"

	"github.com/google/go-containerregistry/pkg/name"
)

type ImageReference struct {
	Registry   string
	Repository string
	Tag        string
	Digest     string
}

func ParseImageReference(imageURI string) ImageReference {
	var imageReference ImageReference

	ref, err := name.ParseReference(imageURI, name.WithDefaultTag(""))
	if err != nil {
		return imageReference
	}

	imageReference.Registry = ref.Context().RegistryStr()
	imageReference.Repository = ref.Context().RepositoryStr()

	if digestRef, ok := ref.(name.Digest); ok {
		imageReference.Digest = digestRef.DigestStr()
	}

	imagePart, _, _ := strings.Cut(imageURI, "@")

	taggedRef, err := name.NewTag(imagePart, name.WithDefaultTag(""))
	if err != nil {
		return imageReference
	}

	imageReference.Tag = taggedRef.TagStr()

	return imageReference
}
