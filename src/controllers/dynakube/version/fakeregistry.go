package version

import (
	"context"
	"fmt"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/src/api/status"
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1/dynakube"
	"github.com/Dynatrace/dynatrace-operator/src/registry"
	"github.com/containers/image/v5/docker/reference"
	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type fakeRegistry struct {
	imageVersions map[string]registry.ImageVersion
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

func (fakeRegistry *fakeRegistry) ImageVersionExt(_ context.Context, _ client.Reader, _ registry.ImageGetter, _ *dynatracev1beta1.DynaKube, imagePath string) (registry.ImageVersion, error) { //nolint:revive // argument-limit
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
