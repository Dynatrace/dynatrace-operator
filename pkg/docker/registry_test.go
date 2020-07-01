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

	registry := RegistryFromImage("alpine")
	registry.Username = username
	registry.Password = password

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

func TestRhcc(t *testing.T) {
	username := os.Getenv(RhccUsername)
	password := os.Getenv(RhccPassword)

	registry := Registry{
		Server:   "registry.connect.redhat.com",
		Image:    "dynatrace/oneagent",
		Username: username,
		Password: password,
	}

	manifest, err := registry.GetLatestManifest()
	assert.NotNil(t, manifest)
	assert.NotEmpty(t, manifest.Config.Digest)
	assert.Nil(t, err)
}

func TestQuay(t *testing.T) {
	username := os.Getenv(QuayUsername)
	password := os.Getenv(QuayPassword)
	image := os.Getenv(QuayRepo)

	registry := Registry{
		Server:   "quay.io",
		Image:    image,
		Username: username,
		Password: password,
	}

	manifest, err := registry.GetLatestManifest()
	assert.NotNil(t, manifest)
	assert.NotEmpty(t, manifest.Config.Digest)
	assert.Nil(t, err)
}

func TestRegistryFromImage(t *testing.T) {
	awsurl := "123456.dkr.ecr.region." + AmazonAws
	image := "image"
	registry := RegistryFromImage(image + ":tag")

	assert.NotNil(t, registry)
	assert.Equal(t, image, registry.Image)
	assert.Equal(t, DockerHubApiServer, registry.Server)

	registry = RegistryFromImage(GcrApiServer + "/" + image + ":tag")

	assert.NotNil(t, registry)
	assert.Equal(t, image, registry.Image)
	assert.Equal(t, GcrApiServer, registry.Server)

	registry = RegistryFromImage(RhccApiServer + "/" + image + ":tag")

	assert.NotNil(t, registry)
	assert.Equal(t, image, registry.Image)
	assert.Equal(t, RhccApiServer, registry.Server)

	registry = RegistryFromImage(QuayApiServer + "/" + image)

	assert.NotNil(t, registry)
	assert.Equal(t, image, registry.Image)
	assert.Equal(t, QuayApiServer, registry.Server)

	registry = RegistryFromImage(awsurl + "/" + image)

	assert.NotNil(t, registry)
	assert.Equal(t, image, registry.Image)
	assert.Equal(t, awsurl, registry.Server)

	image = "user/image"
	registry = RegistryFromImage(image)

	assert.NotNil(t, registry)
	assert.Equal(t, image, registry.Image)
	assert.Equal(t, DockerHubApiServer, registry.Server)

	registry = RegistryFromImage(GcrApiServer + "/" + image)

	assert.NotNil(t, registry)
	assert.Equal(t, image, registry.Image)
	assert.Equal(t, GcrApiServer, registry.Server)

	registry = RegistryFromImage(RhccApiServer + "/" + image)

	assert.NotNil(t, registry)
	assert.Equal(t, image, registry.Image)
	assert.Equal(t, RhccApiServer, registry.Server)

	registry = RegistryFromImage(QuayApiServer + "/" + image + ":tag")

	assert.NotNil(t, registry)
	assert.Equal(t, image, registry.Image)
	assert.Equal(t, QuayApiServer, registry.Server)

	registry = RegistryFromImage(awsurl + "/" + image + ":tag")

	assert.NotNil(t, registry)
	assert.Equal(t, image, registry.Image)
	assert.Equal(t, awsurl, registry.Server)
}

const (
	DockerUsername = "DOCKER_USERNAME"
	DockerPassword = "DOCKER_PASSWORD"

	GcrUsername = "GCR_USERNAME"
	GcrPassword = "GCR_PASSWORD"

	RhccUsername = "RHCC_USERNAME"
	RhccPassword = "RHCC_PASSWORD"

	QuayUsername = "QUAY_USERNAME"
	QuayPassword = "QUAY_PASSWORD"
	QuayRepo     = "QUAY_REPO"
)
