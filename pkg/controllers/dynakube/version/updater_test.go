package version

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/timeprovider"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestRun(t *testing.T) {
	testImage := "some.registry.com:1.2.3.4-5"
	timeProvider := timeprovider.New().Freeze()

	t.Run("set source and probe at the end, if no error", func(t *testing.T) {
		target := &status.VersionStatus{}
		versionReconciler := reconciler{
			timeProvider: timeProvider,
		}
		updater := newCustomImageUpdater(t, target, testImage)
		err := versionReconciler.run(t.Context(), updater)
		require.NoError(t, err)
		assert.Equal(t, timeProvider.Now(), target.LastProbeTimestamp)
		assert.Equal(t, status.CustomImageVersionSource, target.Source)
		assert.Equal(t, testImage, target.ImageID)
		assert.Equal(t, string(status.CustomImageVersionSource), target.Version)
	})

	t.Run("set source and probe at the end, if invalid custom image", func(t *testing.T) {
		target := &status.VersionStatus{}
		versionReconciler := reconciler{
			timeProvider: timeProvider,
		}
		updater := newCustomImageUpdater(t, target, "incorrect-uri")
		err := versionReconciler.run(t.Context(), updater)
		require.NoError(t, err)
		assert.Equal(t, timeProvider.Now(), target.LastProbeTimestamp)
		assert.Equal(t, status.CustomImageVersionSource, target.Source)
		assert.Equal(t, string(status.CustomImageVersionSource), target.Version)
	})
	t.Run("autoUpdate disabled, runs if status is empty or source changes", func(t *testing.T) {
		target := &status.VersionStatus{}
		versionReconciler := reconciler{
			timeProvider: timeProvider,
		}
		updater := newDefaultUpdater(t, target, false)

		// 1. call => status empty => should run
		err := versionReconciler.run(t.Context(), updater)
		require.NoError(t, err)
		updater.AssertNumberOfCalls(t, "UseTenantRegistry", 1)
		assert.Equal(t, timeProvider.Now(), target.LastProbeTimestamp)
		assert.Equal(t, status.TenantRegistryVersionSource, target.Source)

		// 2. call => status NOT empty => should NOT run
		err = versionReconciler.run(t.Context(), updater)
		require.NoError(t, err)
		updater.AssertNumberOfCalls(t, "UseTenantRegistry", 1)

		// 3. call => source is different => should run
		target.Source = status.CustomImageVersionSource
		err = versionReconciler.run(t.Context(), updater)
		require.NoError(t, err)
		updater.AssertNumberOfCalls(t, "UseTenantRegistry", 2)

		// 4. call => source is NOT different => should NOT run
		err = versionReconciler.run(t.Context(), updater)
		require.NoError(t, err)
		updater.AssertNumberOfCalls(t, "UseTenantRegistry", 2)
	})

	t.Run("autoUpdate disabled, runs if status custom-version is set", func(t *testing.T) {
		target := &status.VersionStatus{}
		versionReconciler := reconciler{
			timeProvider: timeProvider,
		}
		updater := newCustomVersionUpdater(t, target, "123", false)

		// 1. call => status empty => should run
		err := versionReconciler.run(t.Context(), updater)
		require.NoError(t, err)
		updater.AssertNumberOfCalls(t, "UseTenantRegistry", 1)
		assert.Equal(t, timeProvider.Now(), target.LastProbeTimestamp)
		assert.Equal(t, status.CustomVersionVersionSource, target.Source)

		// 2. call => it is custom version => should run
		err = versionReconciler.run(t.Context(), updater)
		require.NoError(t, err)
		updater.AssertNumberOfCalls(t, "UseTenantRegistry", 2)
		assert.Equal(t, status.CustomVersionVersionSource, target.Source)
	})
	t.Run("classicfullstack enabled, public registry is ignored", func(t *testing.T) {
		target := &status.VersionStatus{
			Source: status.TenantRegistryVersionSource,
		}
		versionReconciler := reconciler{
			timeProvider: timeProvider,
		}
		updater := newClassicFullStackUpdater(t, target, false)
		updater.On("CustomImage").Return("")
		updater.On("CustomVersion").Return("")

		err := versionReconciler.run(t.Context(), updater)
		require.NoError(t, err)
		assert.Equal(t, timeProvider.Now(), target.LastProbeTimestamp)
		assert.Equal(t, status.TenantRegistryVersionSource, target.Source)
		assert.Empty(t, target.Version)
	})
	t.Run("classicfullstack enabled, public registry is ignored, custom image is set", func(t *testing.T) {
		target := &status.VersionStatus{
			Source: status.TenantRegistryVersionSource,
		}
		versionReconciler := reconciler{
			timeProvider: timeProvider,
		}
		updater := newClassicFullStackUpdater(t, target, false)
		updater.On("CustomImage").Return(testImage)

		err := versionReconciler.run(t.Context(), updater)
		require.NoError(t, err)
		assert.Equal(t, timeProvider.Now(), target.LastProbeTimestamp)
		assert.Equal(t, status.CustomImageVersionSource, target.Source)
	})
}

func TestDetermineSource(t *testing.T) {
	customImage := "my.special.image"
	customVersion := "3.2.1.4-5"

	t.Run("custom-image", func(t *testing.T) {
		updater := newCustomImageUpdater(t, nil, customImage)
		source := determineSource(updater)
		assert.Equal(t, status.CustomImageVersionSource, source)
	})
	t.Run("custom-version", func(t *testing.T) {
		updater := newCustomVersionUpdater(t, nil, customVersion, false)
		source := determineSource(updater)
		assert.Equal(t, status.CustomVersionVersionSource, source)
	})

	t.Run("default", func(t *testing.T) {
		updater := newDefaultUpdater(t, nil, true)
		source := determineSource(updater)
		assert.Equal(t, status.TenantRegistryVersionSource, source)
	})

	t.Run("classicfullstack ignores public registry feature flag", func(t *testing.T) {
		updater := newClassicFullStackUpdater(t, nil, false)
		updater.On("CustomImage").Return("")
		updater.On("CustomVersion").Return("")
		source := determineSource(updater)
		assert.Equal(t, status.TenantRegistryVersionSource, source)
	})

	t.Run("classicfullstack ignores public registry feature flag and sets custom image if set", func(t *testing.T) {
		customVersion := "1.2.3.4-5"
		updater := newClassicFullStackUpdater(t, nil, false)
		updater.On("CustomImage").Return("")
		updater.On("CustomVersion").Return(customVersion)
		source := determineSource(updater)
		assert.Equal(t, status.CustomVersionVersionSource, source)
	})
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

func newCustomImageUpdater(t *testing.T, target *status.VersionStatus, image string) *MockStatusUpdater {
	updater := newBaseUpdater(t, target, true)
	updater.On("CustomImage").Maybe().Return(image)

	return updater
}

func newCustomVersionUpdater(t *testing.T, target *status.VersionStatus, version string, autoUpdate bool) *MockStatusUpdater {
	updater := newBaseUpdater(t, target, autoUpdate)
	updater.On("CustomImage").Maybe().Return("")
	updater.On("IsAutoRegistryEnabled").Maybe().Maybe().Return(false)
	updater.On("CustomVersion").Maybe().Return(version)
	updater.On("UseTenantRegistry", mock.Anything).Maybe().Return(nil)

	return updater
}

func newFailingUpdater(t *testing.T, target *status.VersionStatus) *MockStatusUpdater {
	updater := newBaseUpdater(t, target, true)
	updater.On("CustomImage").Maybe().Return("")
	updater.On("IsAutoRegistryEnabled").Maybe().Maybe().Return(false)
	updater.On("CustomVersion").Maybe().Return("")
	updater.On("UseTenantRegistry", mock.Anything).Maybe().Return(errors.New("BOOM"))

	return updater
}

func newDefaultUpdater(t *testing.T, target *status.VersionStatus, autoUpdate bool) *MockStatusUpdater {
	updater := newBaseUpdater(t, target, autoUpdate)
	updater.On("CustomImage").Maybe().Return("")
	updater.On("IsAutoRegistryEnabled").Maybe().Maybe().Return(false)
	updater.On("CustomVersion").Maybe().Return("")
	updater.On("UseTenantRegistry", mock.Anything).Maybe().Return(nil)

	return updater
}

func newClassicFullStackUpdater(t *testing.T, target *status.VersionStatus, autoUpdate bool) *MockStatusUpdater {
	updater := newBaseUpdater(t, target, autoUpdate)
	updater.On("IsAutoRegistryEnabled").Maybe().Return(false)

	return updater
}

func newBaseUpdater(t *testing.T, target *status.VersionStatus, autoUpdate bool) *MockStatusUpdater {
	updater := NewMockStatusUpdater(t)
	updater.On("Name").Maybe().Return("mock")
	updater.On("Target").Maybe().Return(target)
	updater.On("IsEnabled").Maybe().Return(true)
	updater.On("IsAutoUpdateEnabled").Maybe().Return(autoUpdate)
	updater.On("ValidateStatus").Maybe().Return(nil)

	return updater
}

func assertStatusBasedOnTenantRegistry(t *testing.T, expectedImage, expectedVersion string, versionStatus status.VersionStatus) {
	assert.Equal(t, expectedImage, versionStatus.ImageID)
	assert.Equal(t, expectedVersion, versionStatus.Version)
}
