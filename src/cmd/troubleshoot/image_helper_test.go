package troubleshoot

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestImagePullableSplitImage(t *testing.T) {
	t.Run("valid image name with version", func(t *testing.T) {
		registry, image, version, err := splitImageName(testValidImageNameWithVersion)
		require.NoError(t, err)
		assert.Equal(t, testRegistry, registry, "invalid registry")
		assert.Equal(t, testImage, image, "invalid image")
		assert.Equal(t, testVersion, version, "invalid version")
	})
	t.Run("valid image name without version", func(t *testing.T) {
		registry, image, version, err := splitImageName(testValidImageNameWithoutVersion)
		require.NoError(t, err)
		assert.Equal(t, testRegistry, registry, "invalid registry")
		assert.Equal(t, testImage, image, "invalid image")
		assert.Equal(t, "latest", version, "invalid version")
	})
	t.Run("invalid image name", func(t *testing.T) {
		registry, image, version, err := splitImageName(testInvalidImageName)
		assert.Error(t, err)
		assert.NotEqual(t, testRegistry, registry, "valid registry")
		assert.NotEqual(t, testInvalidImage, image, "valid image")
		assert.NotEqual(t, testVersion, version, "valid version")
	})
}

func TestSplitCustomImage(t *testing.T) {
	// myhiddenserver.com/myrepo/mymissingcodemodules:1.253.0.20220929-235140
	// quay.io/dynatrace/custom-codemodules:1.253.0.20220929-235140
	t.Run("valid image name", func(t *testing.T) {
		registry, imagePath, err := splitCustomImageName("quay.io/dynatrace/custom/custom-codemodules:1.253.0.20220929-235140")
		require.NoError(t, err)
		assert.Equal(t, "quay.io", registry)
		assert.Equal(t, "dynatrace/custom/custom-codemodules:1.253.0.20220929-235140", imagePath)

	})
	t.Run("valid image name without version", func(t *testing.T) {
		registry, imagePath, err := splitCustomImageName("quay.io/dynatrace/custom/custom-codemodules")
		require.NoError(t, err)
		assert.Equal(t, "quay.io", registry)
		assert.Equal(t, "dynatrace/custom/custom-codemodules", imagePath)

	})
	t.Run("valid image name without repository", func(t *testing.T) {
		registry, imagePath, err := splitCustomImageName("quay.io/custom-codemodules:1.253.0.20220929-235140")
		require.NoError(t, err)
		assert.Equal(t, "quay.io", registry)
		assert.Equal(t, "custom-codemodules:1.253.0.20220929-235140", imagePath)
	})
	t.Run("valid image name without repository and version", func(t *testing.T) {
		registry, imagePath, err := splitCustomImageName("quay.io/custom-codemodules")
		require.NoError(t, err)
		assert.Equal(t, "quay.io", registry)
		assert.Equal(t, "custom-codemodules", imagePath)
	})
}
