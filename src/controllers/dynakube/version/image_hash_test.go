package version

import (
	"context"
	"fmt"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/src/api/status"
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1/dynakube"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/version/testdata"
	"github.com/Dynatrace/dynatrace-operator/src/registry"
	"github.com/Dynatrace/dynatrace-operator/src/registry/mocks"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

func (fakeRegistry *fakeRegistry) ImageVersionExt(_ context.Context, _ client.Reader, _ registry.ImageGetter, _ *dynatracev1beta1.DynaKube, imagePath string) (registry.ImageVersion, error) { //nolint:revive // argument-limit
	return fakeRegistry.ImageVersion(imagePath)
}

func assertPublicRegistryVersionStatusEquals(t *testing.T, registry *fakeRegistry, imageRef name.Tag, versionStatus status.VersionStatus) { //nolint:revive // argument-limit
	assertVersionStatusEquals(t, registry, imageRef, versionStatus)
	assert.Empty(t, versionStatus.Version)
}

func assertVersionStatusEquals(t *testing.T, registry *fakeRegistry, imageRef name.Tag, versionStatus status.VersionStatus) { //nolint:revive // argument-limit
	expectedDigest, err := registry.ImageVersion(imageRef.String())

	assert.NoError(t, err, "Image version is unexpectedly unknown for '%s'", imageRef.String())
	assert.Contains(t, versionStatus.ImageID, expectedDigest.Digest)
}

func TestGetImageVersion(t *testing.T) {
	dynakube := dynatracev1beta1.DynaKube{}
	imageName := "dynatrace-operator:1.0.0"
	apiReader := fake.NewClientBuilder().Build()

	pullSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: dynakube.PullSecretName(),
		},
		Data: map[string][]byte{
			".dockerconfigjson": []byte(""),
		},
	}
	apiReader.Create(context.Background(), pullSecret)

	t.Run("without proxy or trustedCAs", func(t *testing.T) {
		mockImageGetter := &mocks.MockImageGetter{}
		mockImageGetter.On("GetImageVersion", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(registry.ImageVersion{}, nil)

		got, err := GetImageVersion(context.TODO(), apiReader, mockImageGetter, &dynakube, imageName)
		assert.NotNil(t, got)
		assert.Nil(t, err)
	})
	t.Run("with proxy", func(t *testing.T) {
		mockImageGetter := &mocks.MockImageGetter{}
		mockImageGetter.On("GetImageVersion", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(registry.ImageVersion{}, nil)
		dynakube.Spec = dynatracev1beta1.DynaKubeSpec{Proxy: &dynatracev1beta1.DynaKubeProxy{Value: "dummy-proxy"}}

		got, err := GetImageVersion(context.Background(), apiReader, mockImageGetter, &dynakube, imageName)
		assert.NotNil(t, got)
		assert.Nil(t, err)
	})
	t.Run("with trustedCAs", func(t *testing.T) {
		mockImageGetter := &mocks.MockImageGetter{}
		mockImageGetter.On("GetImageVersion", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(registry.ImageVersion{}, nil)
		configMapName := "dummy-certs-configmap"

		dynakube := dynatracev1beta1.DynaKube{Spec: dynatracev1beta1.DynaKubeSpec{TrustedCAs: configMapName}}
		imageName := "dynatrace-operator:1.0.0"

		apiReader.Create(context.Background(),
			&corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{Name: configMapName},
				Data: map[string]string{
					"certs": testdata.CertsContent,
				},
			})

		got, err := GetImageVersion(context.Background(), apiReader, mockImageGetter, &dynakube, imageName)
		assert.NotNil(t, got)
		assert.Nil(t, err)
	})
}
