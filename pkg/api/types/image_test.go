// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestImageRef_String(t *testing.T) {
	tests := []struct {
		name     string
		ref      ImageRef
		expected string
	}{
		{
			"repository and tag",
			ImageRef{Repository: "docker.io/dynatrace/image", Tag: "1.2.3"},
			"docker.io/dynatrace/image:1.2.3",
		},
		{
			"digest wins over tag",
			ImageRef{Repository: "docker.io/dynatrace/image", Tag: "1.2.3", Digest: "sha256:abc"},
			"docker.io/dynatrace/image@sha256:abc",
		},
		{
			"digest without tag",
			ImageRef{Repository: "docker.io/dynatrace/image", Digest: "sha256:abc"},
			"docker.io/dynatrace/image@sha256:abc",
		},
		{"empty", ImageRef{}, ""},
		{"only repository", ImageRef{Repository: "test"}, ""},
		{"only tag", ImageRef{Tag: "test"}, ""},
		{"only digest", ImageRef{Digest: "test"}, ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.ref.String())
		})
	}
}

func TestImageRef_StringWithDefaults(t *testing.T) {
	t.Run("falls back to default repo and tag", func(t *testing.T) {
		ref := ImageRef{}
		assert.Equal(t, "default-repo:default-tag", ref.StringWithDefaults("default-repo", "default-tag"))
	})

	t.Run("digest on ref ignores default tag", func(t *testing.T) {
		ref := ImageRef{Digest: "sha256:abc"}
		assert.Equal(t, "default-repo@sha256:abc", ref.StringWithDefaults("default-repo", "default-tag"))
	})
}
