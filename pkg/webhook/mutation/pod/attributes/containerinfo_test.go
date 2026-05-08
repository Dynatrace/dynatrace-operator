package attributes

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
)

func TestNewContainerInfos(t *testing.T) {
	tests := []struct {
		name            string
		image           string
		containerName   string
		wantRegistry    string
		wantRepository  string
		wantTag         string
		wantImageDigest string
	}{
		{
			name:           "full image with registry, path, and tag",
			image:          "registry.io/repo/image:tag",
			containerName:  "my-container",
			wantRegistry:   "registry.io",
			wantRepository: "repo/image",
			wantTag:        "tag",
		},
		{
			name:            "image with registry and digest",
			image:           "registry.io/repo@sha256:abc123",
			containerName:   "my-container",
			wantRegistry:    "registry.io",
			wantRepository:  "repo",
			wantTag:         "",
			wantImageDigest: "sha256:abc123",
		},
		{
			name:            "image with registry, tag, and digest",
			image:           "registry.io/repo/image:tag@sha256:abc123",
			containerName:   "my-container",
			wantRegistry:    "registry.io",
			wantRepository:  "repo/image",
			wantTag:         "tag",
			wantImageDigest: "sha256:abc123",
		},
		{
			name:           "image without registry (no slash)",
			image:          "image:tag",
			containerName:  "my-container",
			wantRegistry:   "",
			wantRepository: "image",
			wantTag:        "tag",
		},
		{
			name:           "bare image name, no registry and no tag",
			image:          "image",
			containerName:  "my-container",
			wantRegistry:   "",
			wantRepository: "image",
			wantTag:        "",
		},
		{
			name:           "image with registry but no tag",
			image:          "registry.io/repo/image",
			containerName:  "my-container",
			wantRegistry:   "registry.io",
			wantRepository: "repo/image",
			wantTag:        "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := corev1.Container{Name: tt.containerName, Image: tt.image}
			infos := NewContainerInfos(c)
			assert.Equal(t, tt.containerName, infos.ContainerName)
			assert.Equal(t, tt.wantRegistry, infos.Registry)
			assert.Equal(t, tt.wantRepository, infos.Repository)
			assert.Equal(t, tt.wantTag, infos.Tag)
			assert.Equal(t, tt.wantImageDigest, infos.ImageDigest)
		})
	}
}

func TestContainerInfos_ToJson(t *testing.T) {
	t.Run("produces valid JSON with all fields", func(t *testing.T) {
		c := corev1.Container{Name: "my-container", Image: "registry.io/repo/image:tag"}
		infos := NewContainerInfos(c)
		jsonStr, err := infos.ToJSON()
		require.NoError(t, err)

		var parsed map[string]string
		require.NoError(t, json.Unmarshal([]byte(jsonStr), &parsed))

		assert.Equal(t, "registry.io", parsed["container_image.registry"])
		assert.Equal(t, "repo/image", parsed["container_image.repository"])
		assert.Equal(t, "tag", parsed["container_image.tags"])
		assert.Equal(t, "my-container", parsed["k8s.container.name"])
	})

	t.Run("omits empty fields due to omitempty", func(t *testing.T) {
		c := corev1.Container{Name: "bare-container", Image: "image"}
		infos := NewContainerInfos(c)
		jsonStr, err := infos.ToJSON()
		require.NoError(t, err)

		var parsed map[string]string
		require.NoError(t, json.Unmarshal([]byte(jsonStr), &parsed))

		assert.NotContains(t, parsed, "container_image.registry")
		assert.NotContains(t, parsed, "container_image.tags")
		assert.NotContains(t, parsed, "container_image.digest")
		assert.Equal(t, "bare-container", parsed["k8s.container.name"])
	})
}
