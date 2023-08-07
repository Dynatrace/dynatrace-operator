package version

import (
	"context"
	"fmt"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/src/api/status"
	"github.com/Dynatrace/dynatrace-operator/src/api/v1beta1/dynakube"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/version/testdata"
	"github.com/Dynatrace/dynatrace-operator/src/registry"
	"github.com/Dynatrace/dynatrace-operator/src/registry/mocks"
	"github.com/Dynatrace/dynatrace-operator/src/scheme"
	"github.com/containers/image/v5/docker/reference"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
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

func (fakeRegistry *fakeRegistry) ImageVersionExt(_ context.Context, _ registry.ImageGetter, imagePath string, _ string, _ *dynakube.DynaKube, _ client.Reader) (registry.ImageVersion, error) { //nolint:revive // argument-limit
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

func TestGetImageVersion(t *testing.T) {
	t.Run("without proxy or trustedCAs", func(t *testing.T) {
		mockImageGetter := &mocks.MockImageGetter{}
		mockImageGetter.On("GetImageVersion", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(registry.ImageVersion{}, nil)

		dynakube := dynakube.DynaKube{}
		imageName := "dynatrace-operator:1.0.0"
		registryAuthPath := "dummy-registry-auth-path"
		apiReader := fake.NewClientBuilder().Build()

		got, err := GetImageVersion(context.TODO(), mockImageGetter, imageName, registryAuthPath, &dynakube, apiReader)
		assert.NotNil(t, got)
		assert.Nil(t, err)
	})
	t.Run("with proxy", func(t *testing.T) {
		mockImageGetter := &mocks.MockImageGetter{}
		mockImageGetter.On("GetImageVersion", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(registry.ImageVersion{}, nil)

		dynakube := dynakube.DynaKube{Spec: dynakube.DynaKubeSpec{Proxy: &dynakube.DynaKubeProxy{Value: "dummy-proxy"}}}
		imageName := "dynatrace-operator:1.0.0"
		registryAuthPath := "dummy-registry-auth-path"
		apiReader := fake.NewClientBuilder().Build()

		got, err := GetImageVersion(context.TODO(), mockImageGetter, imageName, registryAuthPath, &dynakube, apiReader)
		assert.NotNil(t, got)
		assert.Nil(t, err)
	})
	t.Run("with trustedCAs", func(t *testing.T) {
		mockImageGetter := &mocks.MockImageGetter{}
		mockImageGetter.On("GetImageVersion", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(registry.ImageVersion{}, nil)
		configMapName := "dummy-certs-configmap"

		dynakube := dynakube.DynaKube{Spec: dynakube.DynaKubeSpec{TrustedCAs: configMapName}}
		imageName := "dynatrace-operator:1.0.0"
		registryAuthPath := "dummy-registry-auth-path"
		apiReader := fake.NewClientBuilder().
			WithScheme(scheme.Scheme).
			WithObjects(
				&corev1.ConfigMap{
					ObjectMeta: v1.ObjectMeta{Name: configMapName},
					Data: map[string]string{
						"certs": testdata.CertsContent,
					},
				},
			).
			Build()

		got, err := GetImageVersion(context.TODO(), mockImageGetter, imageName, registryAuthPath, &dynakube, apiReader)
		assert.NotNil(t, got)
		assert.Nil(t, err)
	})
}
