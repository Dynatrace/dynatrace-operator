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
			name:     "repository and tag",
			ref:      Ref{Repository: "docker.io/dynatrace/image", Tag: "1.2.3"},
			expected: "docker.io/dynatrace/image:1.2.3",
		},
		{
			name:     "digest wins over tag",
			ref:      Ref{Repository: "docker.io/dynatrace/image", Tag: "1.2.3", Digest: "sha256:abc"},
			expected: "docker.io/dynatrace/image@sha256:abc",
		},
		{
			name:     "digest without tag",
			ref:      Ref{Repository: "docker.io/dynatrace/image", Digest: "sha256:abc"},
			expected: "docker.io/dynatrace/image@sha256:abc",
		},
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

func TestRef_HasImage(t *testing.T) {
	tests := []struct {
		name     string
		ref      Ref
		expected bool
	}{
		{name: "empty", ref: Ref{}, expected: false},
		{name: "repo only", ref: Ref{Repository: "repo"}, expected: false},
		{name: "tag only", ref: Ref{Tag: "1.0"}, expected: false},
		{name: "digest only", ref: Ref{Digest: "sha256:abc"}, expected: false},
		{name: "repo + tag", ref: Ref{Repository: "repo", Tag: "1.0"}, expected: true},
		{name: "repo + digest", ref: Ref{Repository: "repo", Digest: "sha256:abc"}, expected: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.ref.HasImage())
		})
	}
}
