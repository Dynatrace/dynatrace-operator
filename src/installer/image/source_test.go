package image

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/src/dockerconfig"
	"github.com/containers/image/v5/docker/reference"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testDir           = "test"
	testImageRegistry = "quay.io"
	testImageName = "busybox:1.36.0@sha256:b5d6fe0712636ceb7430189de28819e195e8966372edfc2d9409d79402a0dc16"
	testImageUri  = testImageRegistry + "/repo/" + testImageName
	testPassword  = "pass"
	testUsername  = "user"
)

func TestGetSourceInfo(t *testing.T) {
	t.Run(`not nil`, func(t *testing.T) {
		sourceCtx, sourceRef, imageRef, err := getSourceInfo(testDir, Properties{
			ImageUri:     testImageUri,
			DockerConfig: createTestDockerConfig(),
		})
		require.NoError(t, err)
		assert.NotNil(t, sourceCtx)
		assert.NotNil(t, sourceRef)
		assert.NotNil(t, imageRef)
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
	return imageRef
}
