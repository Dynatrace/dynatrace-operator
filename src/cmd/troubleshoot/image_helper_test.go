package troubleshoot

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestImagePullableSplitImage(t *testing.T) {
	t.Run("valid image name with version", func(t *testing.T) {
		imgInfo, err := splitImageName(testValidImageNameWithVersion)
		require.NoError(t, err)
		assert.Equal(t, testRegistry, imgInfo.registry, "invalid registry")
		assert.Equal(t, testImage, imgInfo.image, "invalid image")
		assert.Equal(t, testVersion, imgInfo.version, "invalid version")
	})
	t.Run("valid image name without version", func(t *testing.T) {
		imgInfo, err := splitImageName(testValidImageNameWithoutVersion)
		require.NoError(t, err)
		assert.Equal(t, testRegistry, imgInfo.registry, "invalid registry")
		assert.Equal(t, testImage, imgInfo.image, "invalid image")
		assert.Equal(t, "latest", imgInfo.version, "invalid version")
	})
	t.Run("invalid image name", func(t *testing.T) {
		imgInfo, err := splitImageName(testInvalidImageName)
		assert.Error(t, err)
		assert.NotEqual(t, testRegistry, imgInfo.registry, "valid registry")
		assert.NotEqual(t, testInvalidImage, imgInfo.image, "valid image")
		assert.NotEqual(t, testVersion, imgInfo.version, "valid version")
	})
}

func TestSplitCustomImage(t *testing.T) {
	// myhiddenserver.com/myrepo/mymissingcodemodules:1.253.0.20220929-235140
	// quay.io/dynatrace/custom-codemodules:1.253.0.20220929-235140
	t.Run("valid image name", func(t *testing.T) {
		imgInfo, err := splitCustomImageName("quay.io/dynatrace/custom/custom-codemodules:1.253.0.20220929-235140")
		require.NoError(t, err)
		assert.Equal(t, "quay.io", imgInfo.registry)
		assert.Equal(t, "dynatrace/custom/custom-codemodules:1.253.0.20220929-235140", imgInfo.image)

	})
	t.Run("valid image name without version", func(t *testing.T) {
		imgInfo, err := splitCustomImageName("quay.io/dynatrace/custom/custom-codemodules")
		require.NoError(t, err)
		assert.Equal(t, "quay.io", imgInfo.registry)
		assert.Equal(t, "dynatrace/custom/custom-codemodules", imgInfo.image)

	})
	t.Run("valid image name without repository", func(t *testing.T) {
		imgInfo, err := splitCustomImageName("quay.io/custom-codemodules:1.253.0.20220929-235140")
		require.NoError(t, err)
		assert.Equal(t, "quay.io", imgInfo.registry)
		assert.Equal(t, "custom-codemodules:1.253.0.20220929-235140", imgInfo.image)
	})
	t.Run("valid image name without repository and version", func(t *testing.T) {
		imgInfo, err := splitCustomImageName("quay.io/custom-codemodules")
		require.NoError(t, err)
		assert.Equal(t, "quay.io", imgInfo.registry)
		assert.Equal(t, "custom-codemodules", imgInfo.image)
	})
}
