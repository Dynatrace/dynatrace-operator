package version

import (
	"context"
	"fmt"
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/dockerconfig"
	"github.com/containers/image/v5/docker/reference"
	"github.com/stretchr/testify/assert"
)

type fakeRegistry struct {
	imageVersions map[string]ImageVersion
}

func newEmptyFakeRegistry() *fakeRegistry {
	return newFakeRegistry(make(map[string]ImageVersion))
}

func newFakeRegistry(src map[string]ImageVersion) *fakeRegistry {
	reg := fakeRegistry{
		imageVersions: src,
	}
	return &reg
}

func (registry *fakeRegistry) ImageVersion(imagePath string) (ImageVersion, error) {
	if version, exists := registry.imageVersions[imagePath]; !exists {
		return ImageVersion{}, fmt.Errorf(`cannot provide version for image: "%s"`, imagePath)
	} else {
		return version, nil
	}
}

func (registry *fakeRegistry) ImageVersionExt(_ context.Context, imagePath string, _ *dockerconfig.DockerConfig) (ImageVersion, error) {
	return registry.ImageVersion(imagePath)
}

func assertPublicRegistryVersionStatusEquals(t *testing.T, registry *fakeRegistry, imageRef reference.NamedTagged, versionStatus dynatracev1beta1.VersionStatus) { //nolint:revive // argument-limit
	assertVersionStatusEquals(t, registry, imageRef, versionStatus)
	assert.Empty(t, versionStatus.Version)
}

func assertVersionStatusEquals(t *testing.T, registry *fakeRegistry, imageRef reference.NamedTagged, versionStatus dynatracev1beta1.VersionStatus) { //nolint:revive // argument-limit
	expectedDigest, err := registry.ImageVersion(imageRef.String())

	assert.NoError(t, err, "Image version is unexpectedly unknown for '%s'", imageRef.String())
	assert.Contains(t, versionStatus.ImageID, expectedDigest.Hash)
}
