package installer

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/src/dockerconfig"
	"github.com/containers/image/v5/docker/reference"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testImageRegistry = "quay.io"
	testImageName     = "image:tag"
	testImageUri      = testImageRegistry + "/repo/" + testImageName
	testPassword      = "pass"
	testUsername      = "user"
)

func TestGetSourceInfo(t *testing.T) {
	t.Run(`not nil`, func(t *testing.T) {
		sourceCtx, sourceRef, err := getSourceInfo(testDir, ImageInfo{
			Image:        testImageUri,
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
		assert.Equal(t, testUsername, sourceContext.DockerAuthConfig.Username)
		assert.Equal(t, testPassword, sourceContext.DockerAuthConfig.Password)
	})
}

func createTestDockerConfig() dockerconfig.DockerConfig {
	testDockerAuth := dockerconfig.DockerAuth{
		Username: testUsername,
		Password: testPassword,
	}
	dockerConfig := dockerconfig.DockerConfig{
		Auths: map[string]dockerconfig.DockerAuth{
			testImageRegistry: testDockerAuth,
		},
	}
	return dockerConfig
}

func getTestImageReference(t *testing.T) reference.Named {
	imageRef, err := parseImageReference(testImageUri)
	require.NoError(t, err)
	require.NotNil(t, imageRef)
	assert.Equal(t, testImageUri, imageRef.String())
	return imageRef
}
