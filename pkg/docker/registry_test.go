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

func TestGcr(t *testing.T) {
	username := os.Getenv(GcrUsername)
	password := os.Getenv(GcrPassword)

	registry := Registry{
		Server:   "gcr.io",
		Image:    "dynatrace-marketplace-dev/dynatrace-oneagent-operator",
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

	GcrUsername = "GCR_USERNAME"
	GcrPassword = "GCR_PASSWORD"
)
