// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package image

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRef_String(t *testing.T) {
	tests := []struct {
		name     string
		ref      Ref
		expected string
	}{
		{
			"repository and tag",
			Ref{Repository: "docker.io/dynatrace/image", Tag: "1.2.3"},
			"docker.io/dynatrace/image:1.2.3",
		},
		{
			"digest wins over tag",
			Ref{Repository: "docker.io/dynatrace/image", Tag: "1.2.3", Digest: "sha256:abc"},
			"docker.io/dynatrace/image@sha256:abc",
		},
		{
			"digest without tag",
			Ref{Repository: "docker.io/dynatrace/image", Digest: "sha256:abc"},
			"docker.io/dynatrace/image@sha256:abc",
		},
		{"empty", Ref{}, ""},
		{"only repository", Ref{Repository: "test"}, ""},
		{"only tag", Ref{Tag: "test"}, ""},
		{"only digest", Ref{Digest: "test"}, ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.ref.String())
		})
	}
}

func TestRef_StringWithDefaults(t *testing.T) {
	t.Run("falls back to default repo and tag", func(t *testing.T) {
		ref := Ref{}
		assert.Equal(t, "default-repo:default-tag", ref.StringWithDefaults("default-repo", "default-tag"))
	})

	t.Run("digest on ref ignores default tag", func(t *testing.T) {
		ref := Ref{Digest: "sha256:abc"}
		assert.Equal(t, "default-repo@sha256:abc", ref.StringWithDefaults("default-repo", "default-tag"))
	})
}
