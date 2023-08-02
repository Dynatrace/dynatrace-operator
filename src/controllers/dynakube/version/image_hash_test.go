package version

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/src/api/status"
	"github.com/Dynatrace/dynatrace-operator/src/api/v1beta1/dynakube"
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

func assertPublicRegistryVersionStatusEquals(t *testing.T, registry *fakeRegistry, imageRef reference.NamedTagged, versionStatus status.VersionStatus) { //nolint:revive // argument-limit
	assertVersionStatusEquals(t, registry, imageRef, versionStatus)
	assert.Empty(t, versionStatus.Version)
}

func assertVersionStatusEquals(t *testing.T, registry *fakeRegistry, imageRef reference.NamedTagged, versionStatus status.VersionStatus) { //nolint:revive // argument-limit
	expectedDigest, err := registry.ImageVersion(imageRef.String())

	assert.NoError(t, err, "Image version is unexpectedly unknown for '%s'", imageRef.String())
	assert.Contains(t, versionStatus.ImageID, expectedDigest.Digest)
}

func Test_prepareProxyURL(t *testing.T) {
	type args struct {
		dynakube *dynakube.DynaKube
	}
	tests := []struct {
		name string
		args args
		want *url.URL
	}{
		{
			name: "return URL when dynakube has proxy",
			args: args{
				dynakube: &dynakube.DynaKube{
					Spec: dynakube.DynaKubeSpec{
						Proxy: &dynakube.DynaKubeProxy{
							Value: "http://dummy-proxy",
						},
					},
				},
			},
			want: &url.URL{
				Scheme: "http",
				Host:   "dummy-proxy",
			},
		},
		{
			name: "return nil when dynakube does not have proxy",
			args: args{
				dynakube: &dynakube.DynaKube{},
			},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _ := prepareProxyURL(tt.args.dynakube)
			if !assert.Equal(t, got, tt.want) {
				t.Errorf("prepareProxyURL() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_prepareTransport(t *testing.T) {
	type args struct {
		proxyURL *url.URL
	}
	tests := []struct {
		name string
		args args
		want *http.Transport
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := prepareTransport(tt.args.proxyURL); !assert.Equal(t, got, tt.want) {
				t.Errorf("prepareTransport() = %v, want %v", got, tt.want)
			}
		})
	}
}
