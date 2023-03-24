package image

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/src/dockerconfig"
	"github.com/containers/image/v5/docker/reference"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testDir              = "test"
	testImageUri         = "some.registry.com/image:tag@sha256:7ece13a07a20c77a31cc36906a10ebc90bd47970905ee61e8ed491b7f4c5d62f"
	expectedRef          = "some.registry.com/image@sha256:7ece13a07a20c77a31cc36906a10ebc90bd47970905ee61e8ed491b7f4c5d62f"
	testRegistryAuthPath = "testAuthPath"
	testCAPath           = "testCAPath"
)

func TestGetSourceInfo(t *testing.T) {
	t.Run(`not nil`, func(t *testing.T) {
		sourceCtx, sourceRef, err := getSourceInfo(testDir, Properties{
			ImageUri:     testImageUri,
			DockerConfig: createTestDockerConfig(),
		})
		require.NoError(t, err)
		assert.NotNil(t, sourceCtx)
		assert.NotNil(t, sourceRef)
	})
}

func TestParseImageReference(t *testing.T) {
	t.Run(`not nil`, func(t *testing.T) {
		_ = getTestImageReference(t)
	})
}

func TestGetSourceReference(t *testing.T) {
	t.Run(`not nil`, func(t *testing.T) {
		imageRef := getTestImageReference(t)
		sourceRef, err := getSourceReference(imageRef)
		require.NoError(t, err)
		require.NotNil(t, sourceRef)
	})
}

func TestBuildSourceContext(t *testing.T) {
	t.Run(`not nil`, func(t *testing.T) {
		imageRef := getTestImageReference(t)
		dockerConfig := createTestDockerConfig()
		sourceContext := buildSourceContext(imageRef, testDir, dockerConfig)
		require.NotNil(t, sourceContext)
		assert.Equal(t, testCAPath, sourceContext.DockerCertPath)
		assert.Equal(t, testRegistryAuthPath, sourceContext.AuthFilePath)
	})
}

func createTestDockerConfig() dockerconfig.DockerConfig {
	return dockerconfig.DockerConfig{
		RegistryAuthPath: testRegistryAuthPath,
		TrustedCertsPath: testCAPath,
	}
}

func getTestImageReference(t *testing.T) reference.Named {
	imageRef, err := parseImageReference(testImageUri)
	require.NoError(t, err)
	require.NotNil(t, imageRef)
	assert.Equal(t, expectedRef, imageRef.String())
	return imageRef
}
