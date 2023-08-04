package version

import (
	"context"
	"fmt"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/src/api/status"
	"github.com/Dynatrace/dynatrace-operator/src/dockerconfig"
	"github.com/Dynatrace/dynatrace-operator/src/registry"
	"github.com/containers/image/v5/docker/reference"
	"github.com/stretchr/testify/assert"
)

type fakeRegistry struct {
	imageVersions map[string]registry.ImageVersion
}

func newEmptyFakeRegistry() *fakeRegistry {
	return newFakeRegistry(make(map[string]registry.ImageVersion))
}

func newFakeRegistry(src map[string]registry.ImageVersion) *fakeRegistry {
	reg := fakeRegistry{
		imageVersions: src,
	}
	return &reg
}

func (fakeRegistry *fakeRegistry) ImageVersion(imagePath string) (registry.ImageVersion, error) {
	if version, exists := fakeRegistry.imageVersions[imagePath]; !exists {
		return registry.ImageVersion{}, fmt.Errorf(`cannot provide version for image: "%s"`, imagePath)
	} else {
		return version, nil
	}
}

func (fakeRegistry *fakeRegistry) ImageVersionExt(_ context.Context, _ registry.Client, imagePath string, _ *dockerconfig.DockerConfig) (registry.ImageVersion, error) {
	return fakeRegistry.ImageVersion(imagePath)
}

func assertPublicRegistryVersionStatusEquals(t *testing.T, registry *fakeRegistry, imageRef reference.NamedTagged, versionStatus status.VersionStatus) { //nolint:revive // argument-limit
	assertVersionStatusEquals(t, registry, imageRef, versionStatus)
	assert.Empty(t, versionStatus.Version)
}

func assertVersionStatusEquals(t *testing.T, registry *fakeRegistry, imageRef reference.NamedTagged, versionStatus status.VersionStatus) { //nolint:revive // argument-limit
	expectedDigest, err := registry.ImageVersion(imageRef.String())

	assert.NoError(t, err, "Image version is unexpectedly unknown for '%s'", imageRef.String())
	assert.Contains(t, versionStatus.ImageID, expectedDigest.Digest)
}

// func TestGetImageVersion(t *testing.T) {
// 	fakeImageIndex := fakeregistry.FakeImageIndex{}
// 	digest, _ := fakeImageIndex.Digest()
// 	descriptor := remote.Descriptor{
// 		Descriptor: v1.Descriptor{
// 			Digest: digest,
// 		},
// 	}

// 	t.Run("", func(t *testing.T) {
// 		registryMockClient := &mocks.MockClient{}
// 		registryMockClient.On("Get", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&descriptor, nil)
// 		imageName := "dynatrace-operator:1.0.0"
// 		dockerConfig := dockerconfig.NewDockerConfig(fake.NewClient(), dynakube.DynaKube{})

// 		got, err := GetImageVersion(context.TODO(), mocks.MockClient{}, imageName, dockerConfig)
// 		assert.NotNil(t, got)
// 		assert.NotNil(t, err)
// 	})
// }
