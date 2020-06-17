// +build integration

package docker

import (
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func TestDockerHub(t *testing.T) {
	username := os.Getenv(DockerUsername)
	password := os.Getenv(DockerPassword)

	registry := Registry{
		Server:   "",
		Image:    "alpine",
		Username: username,
		Password: password,
	}

	manifest, err := registry.GetLatestManifest()
	assert.NotNil(t, manifest)
	assert.NotEmpty(t, manifest.Config.Digest)
	assert.Nil(t, err)
}

const (
	DockerUsername = "DOCKER_USERNAME"
	DockerPassword = "DOCKER_PASSWORD"
)
