package v2

import (
	"testing"

	containerattr "github.com/Dynatrace/dynatrace-bootstrapper/pkg/configure/attributes/container"
	"github.com/stretchr/testify/require"
)

func TestAddPodAttributes(t *testing.T) {

}

func TestAddContainerAttributes(t *testing.T) {

}

func TestCreateImageInfo(t *testing.T) {
	t.Run("with tag", func(t *testing.T) {
		imageURI := "registry.example.com/repository/image:tag"

		imageInfo := createImageInfo(imageURI)

		require.Equal(t, containerattr.ImageInfo{
			Registry:    "registry.example.com",
			Repository:  "repository/image",
			Tag:         "tag",
			ImageDigest: "",
		}, imageInfo)
	})
	t.Run("with digest", func(t *testing.T) {
		imageURI := "registry.example.com/repository/image@sha256:7173b809ca12ec5dee4506cd86be934c4596dd234ee82c0662eac04a8c2c71dc"

		imageInfo := createImageInfo(imageURI)

		require.Equal(t, containerattr.ImageInfo{
			Registry:    "registry.example.com",
			Repository:  "repository/image",
			Tag:         "",
			ImageDigest: "sha256:7173b809ca12ec5dee4506cd86be934c4596dd234ee82c0662eac04a8c2c71dc",
		}, imageInfo)
	})
	t.Run("with digest and tag", func(t *testing.T) {
		imageURI := "registry.example.com/repository/image:tag@sha256:7173b809ca12ec5dee4506cd86be934c4596dd234ee82c0662eac04a8c2c71dc"

		imageInfo := createImageInfo(imageURI)

		require.Equal(t, containerattr.ImageInfo{
			Registry:    "registry.example.com",
			Repository:  "repository/image",
			Tag:         "tag",
			ImageDigest: "sha256:7173b809ca12ec5dee4506cd86be934c4596dd234ee82c0662eac04a8c2c71dc",
		}, imageInfo)
	})
	t.Run("with missing tag", func(t *testing.T) {
		imageURI := "registry.example.com/repository/image"

		imageInfo := createImageInfo(imageURI)

		require.Equal(t, containerattr.ImageInfo{
			Registry:    "registry.example.com",
			Repository:  "repository/image",
			Tag:         "",
			ImageDigest: "",
		}, imageInfo)
	})

	t.Run("actual example", func(t *testing.T) {
		imageURI := "docker.io/php:fpm-stretch"

		imageInfo := createImageInfo(imageURI)

		require.Equal(t, containerattr.ImageInfo{
			Registry:    "docker.io",
			Repository:  "php",
			Tag:         "fpm-stretch",
			ImageDigest: "",
		}, imageInfo)
	})
}
