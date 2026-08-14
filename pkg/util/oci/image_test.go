// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package oci

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseImageReference(t *testing.T) {
	const (
		baseImage = "registry.example.com/dynatrace/image"
		digest    = "sha256:eb80829917c8bc4c531ac20a4b8ea3d9f7836a9e0ad9702da3cb06ab4205bf80"
	)

	tests := []struct {
		name               string
		imageURI           string
		expectedRegistry   string
		expectedRepository string
		expectedTag        string
		expectedDigest     string
	}{
		{
			name:               "repository only",
			imageURI:           baseImage,
			expectedRegistry:   "registry.example.com",
			expectedRepository: "dynatrace/image",
		},
		{
			name:               "tag",
			imageURI:           baseImage + ":1.2.3",
			expectedRegistry:   "registry.example.com",
			expectedRepository: "dynatrace/image",
			expectedTag:        "1.2.3",
		},
		{
			name:               "tag and digest",
			imageURI:           baseImage + ":1.2.3@" + digest,
			expectedRegistry:   "registry.example.com",
			expectedRepository: "dynatrace/image",
			expectedTag:        "1.2.3",
			expectedDigest:     digest,
		},
		{
			name:               "digest only",
			imageURI:           baseImage + "@" + digest,
			expectedRegistry:   "registry.example.com",
			expectedRepository: "dynatrace/image",
			expectedDigest:     digest,
		},
		{
			name:     "invalid reference",
			imageURI: "not a valid image reference",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			imageReference := ParseImageReference(test.imageURI)

			assert.Equal(t, test.expectedRegistry, imageReference.Registry)
			assert.Equal(t, test.expectedRepository, imageReference.Repository)
			assert.Equal(t, test.expectedTag, imageReference.Tag)
			assert.Equal(t, test.expectedDigest, imageReference.Digest)
		})
	}
}
