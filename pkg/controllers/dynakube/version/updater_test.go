package version

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/oci/registry"
	registryMock "github.com/Dynatrace/dynatrace-operator/pkg/oci/registry/mocks"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/timeprovider"
	mocks "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/controllers/dynakube/version"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestRun(t *testing.T) {
	ctx := context.TODO()
	testImage := dtclient.LatestImageInfo{
		Source: "some.registry.com",
		Tag:    "1.2.3.4-5",
	}
	timeProvider := timeprovider.New().Freeze()

	t.Run("set source and probe at the end, if no error", func(t *testing.T) {
		fakeRegistry := newFakeRegistry(map[string]registry.ImageVersion{
			testImage.String(): {
				Version: testImage.Tag,
			},
		})

		mockImageGetter := registryMock.NewMockImageGetter(t)

		target := &status.VersionStatus{}
		versionReconciler := reconciler{
			timeProvider:   timeProvider,
			registryClient: mockImageGetter,
		}
		updater := newCustomImageUpdater(target, testImage.String())
		err := versionReconciler.run(ctx, updater)
		require.NoError(t, err)
		assert.Equal(t, timeProvider.Now(), target.LastProbeTimestamp)
		assert.Equal(t, status.CustomImageVersionSource, target.Source)
		assertVersionStatusEquals(t, fakeRegistry, getTaggedReference(t, testImage.String()), *target)
	})

	t.Run("set source and probe at the end, if invalid custom image", func(t *testing.T) {
		target := &status.VersionStatus{}
		mockImageGetter := registryMock.NewMockImageGetter(t)
		versionReconciler := reconciler{
			timeProvider:   timeProvider,
			registryClient: mockImageGetter,
		}
		updater := newCustomImageUpdater(target, "incorrect-uri")
		err := versionReconciler.run(ctx, updater)
		require.NoError(t, err)
		assert.Equal(t, timeProvider.Now(), target.LastProbeTimestamp)
		assert.Equal(t, status.CustomImageVersionSource, target.Source)
		assert.Equal(t, string(status.CustomImageVersionSource), target.Version)
	})
	t.Run("autoUpdate disabled, runs if status is empty or source changes", func(t *testing.T) {
		target := &status.VersionStatus{}
		versionReconciler := reconciler{
			timeProvider:   timeProvider,
			registryClient: &registryMock.MockImageGetter{},
		}
		updater := newDefaultUpdater(target, false)

		// 1. call => status empty => should run
		err := versionReconciler.run(ctx, updater)
		require.NoError(t, err)
		updater.AssertNumberOfCalls(t, "UseTenantRegistry", 1)
		assert.Equal(t, timeProvider.Now(), target.LastProbeTimestamp)
		assert.Equal(t, status.TenantRegistryVersionSource, target.Source)

		// 2. call => status NOT empty => should NOT run
		err = versionReconciler.run(ctx, updater)
		require.NoError(t, err)
		updater.AssertNumberOfCalls(t, "UseTenantRegistry", 1)

		// 3. call => source is different => should run
		target.Source = status.CustomImageVersionSource
		err = versionReconciler.run(ctx, updater)
		require.NoError(t, err)
		updater.AssertNumberOfCalls(t, "UseTenantRegistry", 2)

		// 4. call => source is NOT different => should NOT run
		err = versionReconciler.run(ctx, updater)
		require.NoError(t, err)
		updater.AssertNumberOfCalls(t, "UseTenantRegistry", 2)
	})
	t.Run("public registry", func(t *testing.T) {
		fakeRegistry := newFakeRegistry(map[string]registry.ImageVersion{
			testImage.String(): {
				Version: testImage.Tag,
			},
		})
		target := &status.VersionStatus{
			Source: status.TenantRegistryVersionSource,
		}
		mockImageGetter := registryMock.MockImageGetter{}
		mockImageGetter.On("GetImageVersion", mock.Anything, mock.Anything).Return(registry.ImageVersion{Version: testImage.Tag}, nil)

		versionReconciler := reconciler{
			timeProvider:   timeProvider,
			registryClient: &mockImageGetter,
		}
		updater := newPublicRegistryUpdater(target, &testImage, false)
		updater.On("IsClassicFullStackEnabled").Return(false)
		updater.On("CheckForDowngrade", mock.AnythingOfType("string")).Return(false, nil)

		err := versionReconciler.run(ctx, updater)
		require.NoError(t, err)
		updater.AssertNumberOfCalls(t, "LatestImageInfo", 1)
		assert.Equal(t, timeProvider.Now(), target.LastProbeTimestamp)
		assert.Equal(t, status.PublicRegistryVersionSource, target.Source)
		assertVersionStatusEquals(t, fakeRegistry, getTaggedReference(t, testImage.String()), *target)
		assert.NotEmpty(t, target.Version)
	})

	t.Run("public registry, no downgrade allowed", func(t *testing.T) {
		mockImageGetter := registryMock.MockImageGetter{}
		mockImageGetter.On("GetImageVersion", mock.Anything, mock.Anything).Return(registry.ImageVersion{Version: testImage.Tag}, nil)

		target := &status.VersionStatus{
			Source: status.TenantRegistryVersionSource,
		}
		versionReconciler := reconciler{
			timeProvider:   timeProvider,
			registryClient: &mockImageGetter,
		}
		updater := newPublicRegistryUpdater(target, &testImage, false)
		updater.On("IsClassicFullStackEnabled").Return(false)
		updater.On("CheckForDowngrade", mock.AnythingOfType("string")).Return(true, nil)

		err := versionReconciler.run(ctx, updater)
		require.NoError(t, err)
		updater.AssertNumberOfCalls(t, "LatestImageInfo", 1)
		assert.Equal(t, timeProvider.Now(), target.LastProbeTimestamp)
		assert.Equal(t, status.PublicRegistryVersionSource, target.Source)
		assert.Empty(t, target.Version)
		assert.Empty(t, target.ImageID)
	})
	t.Run("classicfullstack enabled, public registry is ignored", func(t *testing.T) {
		mockImageGetter := registryMock.MockImageGetter{}
		mockImageGetter.On("GetImageVersion", mock.Anything, mock.Anything).Return(registry.ImageVersion{Version: testImage.Tag}, nil)

		target := &status.VersionStatus{
			Source: status.TenantRegistryVersionSource,
		}
		versionReconciler := reconciler{
			timeProvider:   timeProvider,
			registryClient: &mockImageGetter,
		}
		updater := newClassicFullStackUpdater(target, false)
		updater.On("CustomImage").Return("")
		updater.On("CustomVersion").Return("")
		updater.On("UseTenantRegistry", mock.Anything).Return(nil)

		err := versionReconciler.run(ctx, updater)
		require.NoError(t, err)
		updater.AssertNumberOfCalls(t, "LatestImageInfo", 0)
		assert.Equal(t, timeProvider.Now(), target.LastProbeTimestamp)
		assert.Equal(t, status.TenantRegistryVersionSource, target.Source)
		assert.Equal(t, target.Version, target.Version)
	})
	t.Run("classicfullstack enabled, public registry is ignored, custom image is set", func(t *testing.T) {
		mockImageGetter := registryMock.MockImageGetter{}
		mockImageGetter.On("GetImageVersion", mock.Anything, mock.Anything).Return(registry.ImageVersion{Version: testImage.Tag}, nil)

		target := &status.VersionStatus{
			Source: status.TenantRegistryVersionSource,
		}
		versionReconciler := reconciler{
			timeProvider:   timeProvider,
			registryClient: &mockImageGetter,
		}
		updater := newClassicFullStackUpdater(target, false)
		updater.On("CustomImage").Return(testImage.String())
		updater.On("CustomVersion").Return(testImage.Tag)
		updater.On("UseTenantRegistry", mock.Anything).Return(nil)

		err := versionReconciler.run(ctx, updater)
		require.NoError(t, err)
		updater.AssertNumberOfCalls(t, "LatestImageInfo", 0)
		assert.Equal(t, timeProvider.Now(), target.LastProbeTimestamp)
		assert.Equal(t, status.CustomImageVersionSource, target.Source)
	})
}

func TestDetermineSource(t *testing.T) {
	customImage := "my.special.image"
	customVersion := "3.2.1.4-5"
	t.Run("custom-image", func(t *testing.T) {
		updater := newCustomImageUpdater(nil, customImage)
		source := determineSource(updater)
		assert.Equal(t, status.CustomImageVersionSource, source)
	})
	t.Run("custom-version", func(t *testing.T) {
		updater := newCustomVersionUpdater(nil, customVersion, false)
		source := determineSource(updater)
		assert.Equal(t, status.CustomVersionVersionSource, source)
	})

	t.Run("public-registry", func(t *testing.T) {
		updater := newPublicRegistryUpdater(nil, nil, false)
		source := determineSource(updater)
		assert.Equal(t, status.PublicRegistryVersionSource, source)
	})

	t.Run("default", func(t *testing.T) {
		updater := newDefaultUpdater(nil, true)
		source := determineSource(updater)
		assert.Equal(t, status.TenantRegistryVersionSource, source)
	})

	t.Run("classicfullstack ignores public registry feature flag", func(t *testing.T) {
		updater := newClassicFullStackUpdater(nil, false)
		updater.On("CustomImage").Return("")
		updater.On("CustomVersion").Return("")
		source := determineSource(updater)
		assert.Equal(t, status.TenantRegistryVersionSource, source)
	})

	t.Run("classicfullstack ignores public registry feature flag and sets custom image if set", func(t *testing.T) {
		testImage := dtclient.LatestImageInfo{
			Source: "some.registry.com",
			Tag:    "1.2.3.4-5",
		}
		updater := newClassicFullStackUpdater(nil, false)
		updater.On("CustomImage").Return("")
		updater.On("CustomVersion").Return(testImage.Tag)
		source := determineSource(updater)
		assert.Equal(t, status.CustomVersionVersionSource, source)
	})
}

func TestUpdateVersionStatus(t *testing.T) {
	ctx := context.TODO()
	testImage := dtclient.LatestImageInfo{
		Source: "some.registry.com",
		Tag:    "1.2.3.4-5",
	}

	t.Run("failing to get digest should cause error", func(t *testing.T) {
		target := status.VersionStatus{}
		mockImageGetter := registryMock.MockImageGetter{}
		mockImageGetter.On("GetImageVersion", mock.Anything, mock.Anything).Return(registry.ImageVersion{}, errors.New("something went wrong"))
		err := setImageIDWithDigest(ctx, &target, &mockImageGetter, testImage.String())
		assert.Error(t, err)
	})

	t.Run("failing to get digest should cause error if proxy is set", func(t *testing.T) {
		target := status.VersionStatus{}
		dynakube := newClassicFullStackDynakube()
		dynakube.Spec.Proxy = &dynatracev1beta1.DynaKubeProxy{Value: "http://username:password@host:port"}
		mockImageGetter := registryMock.MockImageGetter{}
		mockImageGetter.On("GetImageVersion", mock.Anything, mock.Anything).Return(registry.ImageVersion{}, errors.New("something went wrong"))
		err := setImageIDWithDigest(ctx, &target, &mockImageGetter, testImage.String())
		assert.Error(t, err)
	})

	t.Run("set status", func(t *testing.T) {
		fakeRegistry := newFakeRegistry(map[string]registry.ImageVersion{
			testImage.String(): {
				Version: testImage.Tag,
			},
		})
		target := status.VersionStatus{}
		mockImageGetter := registryMock.MockImageGetter{}
		mockImageGetter.On("GetImageVersion", mock.Anything, mock.Anything).Return(registry.ImageVersion{Version: testImage.Tag}, nil)
		err := setImageIDWithDigest(ctx, &target, &mockImageGetter, testImage.String())
		require.NoError(t, err)
		assertVersionStatusEquals(t, fakeRegistry, getTaggedReference(t, testImage.String()), target)
	})

	t.Run("set status, not call digest func", func(t *testing.T) {
		expectedRepo := "some.registry.com/image"
		expectedDigest := "sha256:7ece13a07a20c77a31cc36906a10ebc90bd47970905ee61e8ed491b7f4c5d62f"
		expectedID := expectedRepo + "@" + expectedDigest
		target := status.VersionStatus{}

		mockImageGetter := registryMock.MockImageGetter{}
		mockImageGetter.On("GetImageVersion", mock.Anything, mock.Anything).Return(registry.ImageVersion{}, nil)
		err := setImageIDWithDigest(ctx, &target, &mockImageGetter, expectedID)
		require.NoError(t, err)
		assert.Equal(t, expectedID, target.ImageID)
	})
	t.Run("accept tagged + digest image reference", func(t *testing.T) {
		expectedRepo := "some.registry.com/image"
		expectedDigest := "sha256:7ece13a07a20c77a31cc36906a10ebc90bd47970905ee61e8ed491b7f4c5d62f"
		expectedID := expectedRepo + ":tag@" + expectedDigest
		target := status.VersionStatus{}
		mockImageGetter := registryMock.MockImageGetter{}
		mockImageGetter.On("GetImageVersion", mock.Anything, mock.Anything).Return(registry.ImageVersion{}, nil)
		err := setImageIDWithDigest(ctx, &target, &mockImageGetter, expectedID)
		require.NoError(t, err)
		assert.Equal(t, expectedID, target.ImageID)
	})
	t.Run("providing it with digest still requires registry access", func(t *testing.T) {
		expectedRepo := "some.registry.com/image"
		expectedDigest := "sha256:7ece13a07a20c77a31cc36906a10ebc90bd47970905ee61e8ed491b7f4c5d62f"
		expectedID := expectedRepo + "@" + expectedDigest
		target := status.VersionStatus{}
		faultyRegistry := registryMock.MockImageGetter{}
		faultyRegistry.On("GetImageVersion", mock.Anything, mock.Anything).Return(registry.ImageVersion{}, fmt.Errorf("WANT TO ACCESS REGISTRY"))
		err := setImageIDWithDigest(ctx, &target, &faultyRegistry, expectedID)
		require.Error(t, err)
	})
}

func TestNewImageLib(t *testing.T) {
	mockImageGetter := &registryMock.MockImageGetter{}
	const fakeDigest = "sha256:7ece13a07a20c77a31cc36906a10ebc90bd47970905ee61e8ed491b7f4c5d62f"
	fakeImageVersion := registry.ImageVersion{Digest: fakeDigest}
	mockImageGetter.On("GetImageVersion", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(fakeImageVersion, nil)

	tests := []struct {
		input    string
		expected string
		wantErr  require.ErrorAssertionFunc
	}{
		{
			input:    "some.registry.com/image",
			expected: "",
			wantErr:  require.Error,
		},
		{
			input:    "some.registry.com/image:latest",
			expected: "some.registry.com/image:latest@sha256:7ece13a07a20c77a31cc36906a10ebc90bd47970905ee61e8ed491b7f4c5d62f",
			wantErr:  require.NoError,
		},
		{
			input:    "some.registry.com/image@sha256:7ece13a07a20c77a31cc36906a10ebc90bd47970905ee61e8ed491b7f4c5d62f",
			expected: "some.registry.com/image@sha256:7ece13a07a20c77a31cc36906a10ebc90bd47970905ee61e8ed491b7f4c5d62f",
			wantErr:  require.NoError,
		},
		{
			input:    "some.registry.com/image:0.1.2.3",
			expected: "some.registry.com/image:0.1.2.3@sha256:7ece13a07a20c77a31cc36906a10ebc90bd47970905ee61e8ed491b7f4c5d62f",
			wantErr:  require.NoError,
		},
		{
			input:    "some.registry.com/image:0.1.2.3@sha256:7ece13a07a20c77a31cc36906a10ebc90bd47970905ee61e8ed491b7f4c5d62f",
			expected: "some.registry.com/image:0.1.2.3@sha256:7ece13a07a20c77a31cc36906a10ebc90bd47970905ee61e8ed491b7f4c5d62f",
			wantErr:  require.NoError,
		},
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			targetNew := status.VersionStatus{}
			err := setImageIDWithDigest(context.TODO(), &targetNew, mockImageGetter, test.input)
			test.wantErr(t, err)
			assert.Equal(t, test.expected, targetNew.ImageID)
		})
	}
}

func TestGetTagFromImageID(t *testing.T) {
	tests := []struct {
		name        string
		imageID     string
		expectedTag string
		wantErr     require.ErrorAssertionFunc
	}{
		{
			name:        "get tag from imageID",
			imageID:     "some.registry.com:1.2.3",
			expectedTag: "1.2.3",
			wantErr:     require.NoError,
		},
		{
			name:        "get tag from imageID with tag and digest",
			imageID:     "some.registry.com:1.2.3@sha256:7ece13a07a20c77a31cc36906a10ebc90bd47970905ee61e8ed491b7f4c5d62f",
			expectedTag: "1.2.3",
			wantErr:     require.NoError,
		},
		{
			name:        "get tag from imageID without tag",
			imageID:     "some.registry.com",
			expectedTag: "",
			wantErr:     require.Error,
		},
		{
			name:        "get tag from imageID with latest tag",
			imageID:     "some.registry.com:latest",
			expectedTag: "latest",
			wantErr:     require.NoError,
		},
		{
			name:        "get tag from imageID without tag but digest",
			imageID:     "some.registry.com@sha256:7ece13a07a20c77a31cc36906a10ebc90bd47970905ee61e8ed491b7f4c5d62f",
			expectedTag: "",
			wantErr:     require.Error,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tag, err := getTagFromImageID(test.imageID)
			test.wantErr(t, err)
			assert.Equal(t, test.expectedTag, tag)
		})
	}

	t.Run("error for malformed imageID", func(t *testing.T) {
		imageID := "some.registry.com@1.2.3"

		_, err := getTagFromImageID(imageID)

		require.Error(t, err)
	})
}

func enablePublicRegistry(dynakube *dynatracev1beta1.DynaKube) *dynatracev1beta1.DynaKube {
	if dynakube.Annotations == nil {
		dynakube.Annotations = make(map[string]string)
	}
	dynakube.Annotations[dynatracev1beta1.AnnotationFeaturePublicRegistry] = "true"
	return dynakube
}

func newCustomImageUpdater(target *status.VersionStatus, image string) *mocks.StatusUpdater {
	updater := newBaseUpdater(target, true)
	updater.On("CustomImage").Return(image)
	return updater
}

func newCustomVersionUpdater(target *status.VersionStatus, version string, autoUpdate bool) *mocks.StatusUpdater {
	updater := newBaseUpdater(target, autoUpdate)
	updater.On("CustomImage").Return("")
	updater.On("IsPublicRegistryEnabled").Return(false)
	updater.On("CustomVersion").Return(version)
	return updater
}

func newDefaultUpdater(target *status.VersionStatus, autoUpdate bool) *mocks.StatusUpdater {
	updater := newBaseUpdater(target, autoUpdate)
	updater.On("CustomImage").Return("")
	updater.On("IsPublicRegistryEnabled").Return(false)
	updater.On("CustomVersion").Return("")
	updater.On("UseTenantRegistry", mock.Anything).Return(nil)
	return updater
}

func newPublicRegistryUpdater(target *status.VersionStatus, imageInfo *dtclient.LatestImageInfo, autoUpdate bool) *mocks.StatusUpdater {
	updater := newBaseUpdater(target, autoUpdate)
	updater.On("CustomImage").Return("")
	updater.On("IsPublicRegistryEnabled").Return(true)
	updater.On("LatestImageInfo").Return(imageInfo, nil)
	return updater
}

func newClassicFullStackUpdater(target *status.VersionStatus, autoUpdate bool) *mocks.StatusUpdater {
	updater := newBaseUpdater(target, autoUpdate)
	updater.On("IsPublicRegistryEnabled").Return(false)
	return updater
}

func newBaseUpdater(target *status.VersionStatus, autoUpdate bool) *mocks.StatusUpdater {
	updater := mocks.StatusUpdater{}
	updater.On("Name").Return("mock")
	updater.On("Target").Return(target)
	updater.On("IsEnabled").Return(true)
	updater.On("IsAutoUpdateEnabled").Return(autoUpdate)
	updater.On("ValidateStatus").Return(nil)
	return &updater
}

func newClassicFullStackDynakube() *dynatracev1beta1.DynaKube {
	return &dynatracev1beta1.DynaKube{
		Spec: dynatracev1beta1.DynaKubeSpec{
			OneAgent: dynatracev1beta1.OneAgentSpec{
				ClassicFullStack: &dynatracev1beta1.HostInjectSpec{},
			},
		},
	}
}

func getTaggedReference(t *testing.T, image string) name.Tag {
	ref, err := name.ParseReference(image)
	require.NoError(t, err)
	taggedRef, ok := ref.(name.Tag)
	require.True(t, ok)
	return taggedRef
}

func assertStatusBasedOnTenantRegistry(t *testing.T, expectedImage, expectedVersion string, versionStatus status.VersionStatus) { //nolint:revive // argument-limit
	assert.Equal(t, expectedImage, versionStatus.ImageID)
	assert.Equal(t, expectedVersion, versionStatus.Version)
}
