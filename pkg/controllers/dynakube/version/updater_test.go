package version

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/image"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRun(t *testing.T) {
	testImage := "some.registry.com:1.2.3.4-5"

	t.Run("set source and probe at the end, if no error", func(t *testing.T) {
		target := &status.VersionStatus{}
		versionReconciler := Reconciler{}
		updater := NewMockStatusUpdater(t)
		updater.EXPECT().Name().Return("mock").Once()
		updater.EXPECT().Target().Return(target).Times(3)
		updater.EXPECT().CustomImage().Return(testImage).Times(2)
		err := versionReconciler.run(t.Context(), updater)
		require.NoError(t, err)
		require.NotNil(t, target.LastProbeTimestamp)
		assert.Equal(t, status.CustomImageVersionSource, target.Source)
		assert.Equal(t, testImage, target.ImageID)
		assert.Equal(t, string(status.CustomImageVersionSource), target.Version)
	})

	t.Run("set source and probe at the end, if invalid custom image", func(t *testing.T) {
		target := &status.VersionStatus{}
		versionReconciler := Reconciler{}
		updater := NewMockStatusUpdater(t)
		updater.EXPECT().Name().Return("mock").Once()
		updater.EXPECT().Target().Return(target).Times(3)
		updater.EXPECT().CustomImage().Return("incorrect-uri").Times(2)
		err := versionReconciler.run(t.Context(), updater)
		require.NoError(t, err)
		require.NotNil(t, target.LastProbeTimestamp)
		assert.Equal(t, status.CustomImageVersionSource, target.Source)
		assert.Equal(t, string(status.CustomImageVersionSource), target.Version)
	})
	t.Run("autoUpdate disabled, runs if status is empty or source changes", func(t *testing.T) {
		target := &status.VersionStatus{}
		versionReconciler := Reconciler{}
		updater := newDefaultUpdater(t, false)
		updater.EXPECT().Name().Return("mock").Times(5)
		updater.EXPECT().Target().Return(target).Times(16)
		updater.EXPECT().CustomImage().Return("").Times(4)
		updater.EXPECT().CustomVersion().Return("").Times(4)
		updater.EXPECT().IsPublicRegistryEnabled().Return(false).Times(6)

		// 1. call => status empty => should run
		err := versionReconciler.run(t.Context(), updater)
		require.NoError(t, err)
		updater.AssertNumberOfCalls(t, "UseTenantRegistry", 1)
		require.NotNil(t, target.LastProbeTimestamp)
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
		versionReconciler := Reconciler{}
		updater := newCustomVersionUpdater(t, false)
		updater.EXPECT().Name().Return("mock").Times(2)
		updater.EXPECT().Target().Return(target).Times(4)
		updater.EXPECT().CustomImage().Return("").Times(2)
		updater.EXPECT().CustomVersion().Return("123").Times(2)
		updater.EXPECT().IsPublicRegistryEnabled().Return(false).Times(4)

		// 1. call => status empty => should run
		err := versionReconciler.run(t.Context(), updater)
		require.NoError(t, err)
		updater.AssertNumberOfCalls(t, "UseTenantRegistry", 1)
		require.NotNil(t, target.LastProbeTimestamp)
		assert.Equal(t, status.CustomVersionVersionSource, target.Source)

		// 2. call => it is custom version => should run
		err = versionReconciler.run(t.Context(), updater)
		require.NoError(t, err)
		updater.AssertNumberOfCalls(t, "UseTenantRegistry", 2)
		assert.Equal(t, status.CustomVersionVersionSource, target.Source)
	})
	t.Run("classicFullStack enabled, public registry is ignored", func(t *testing.T) {
		target := &status.VersionStatus{
			Source: status.TenantRegistryVersionSource,
		}
		versionReconciler := Reconciler{}
		updater := NewMockStatusUpdater(t)
		updater.EXPECT().Name().Return("mock").Once()
		updater.EXPECT().Target().Return(target).Times(4)
		updater.EXPECT().CustomImage().Return("").Once()
		updater.EXPECT().CustomVersion().Return("").Once()
		updater.EXPECT().IsPublicRegistryEnabled().Return(false).Once()
		updater.EXPECT().IsAutoUpdateEnabled().Return(false).Once()

		err := versionReconciler.run(t.Context(), updater)
		require.NoError(t, err)
		require.NotNil(t, target.LastProbeTimestamp)
		assert.Equal(t, status.TenantRegistryVersionSource, target.Source)
		assert.Empty(t, target.Version)
	})
	t.Run("classicFullStack enabled, public registry is ignored, custom image is set", func(t *testing.T) {
		target := &status.VersionStatus{
			Source: status.TenantRegistryVersionSource,
		}
		versionReconciler := Reconciler{}
		updater := NewMockStatusUpdater(t)
		updater.EXPECT().Name().Return("mock").Once()
		updater.EXPECT().Target().Return(target).Times(3)
		updater.EXPECT().CustomImage().Return(testImage).Twice()

		err := versionReconciler.run(t.Context(), updater)
		require.NoError(t, err)
		require.NotNil(t, target.LastProbeTimestamp)
		assert.Equal(t, status.CustomImageVersionSource, target.Source)
	})
	t.Run("public registry: happy path, status updated", func(t *testing.T) {
		target := &status.VersionStatus{}
		versionReconciler := Reconciler{}
		imageInfo := &image.Info{URI: "registry.io/dynatrace/oneagent:1.2.3", Tag: "1.2.3"}
		updater := newPublicRegistryUpdater(t, true)
		updater.EXPECT().Name().Return("mock").Once()
		updater.EXPECT().Target().Return(target).Times(3)
		updater.EXPECT().LatestImageInfo(anyCtx).Return(imageInfo, nil).Once()
		updater.EXPECT().CheckForDowngrade(anyCtx, "1.2.3").Return(false, nil).Once()
		updater.EXPECT().ValidateStatus(anyCtx).Return(nil).Once()

		err := versionReconciler.run(t.Context(), updater)
		require.NoError(t, err)
		assert.Equal(t, "registry.io/dynatrace/oneagent:1.2.3", target.ImageID)
		assert.Equal(t, "1.2.3", target.Version)
		assert.Equal(t, status.PublicRegistryVersionSource, target.Source)
		require.NotNil(t, target.LastProbeTimestamp)
	})

	t.Run("public registry: API error propagated, status not updated", func(t *testing.T) {
		target := &status.VersionStatus{}
		versionReconciler := Reconciler{}
		updater := newPublicRegistryUpdater(t, true)
		updater.EXPECT().Name().Return("mock").Times(2)
		updater.EXPECT().LatestImageInfo(anyCtx).Return(nil, errors.New("API error")).Once()

		err := versionReconciler.run(t.Context(), updater)
		require.Error(t, err)
		assert.Empty(t, target.ImageID)
		assert.Nil(t, target.LastProbeTimestamp)
	})

	t.Run("public registry: downgrade detected, image not updated", func(t *testing.T) {
		target := &status.VersionStatus{ImageID: "registry.io/oneagent:1.3.0", Version: "1.3.0"}
		versionReconciler := Reconciler{}
		imageInfo := &image.Info{URI: "registry.io/dynatrace/oneagent:1.2.0", Tag: "1.2.0"}
		updater := newPublicRegistryUpdater(t, true)
		updater.EXPECT().Name().Return("mock").Once()
		updater.EXPECT().Target().Return(target).Times(2)
		updater.EXPECT().LatestImageInfo(anyCtx).Return(imageInfo, nil).Once()
		updater.EXPECT().CheckForDowngrade(anyCtx, "1.2.0").Return(true, nil).Once()
		updater.EXPECT().ValidateStatus(anyCtx).Return(nil).Once()

		err := versionReconciler.run(t.Context(), updater)
		require.NoError(t, err)
		assert.Equal(t, "registry.io/oneagent:1.3.0", target.ImageID)
		assert.Equal(t, "1.3.0", target.Version)
	})

	t.Run("public registry: empty tag, ValidateStatus fails", func(t *testing.T) {
		target := &status.VersionStatus{}
		versionReconciler := Reconciler{}
		imageInfo := &image.Info{URI: "registry.io/dynatrace/oneagent@sha256:abc123", Tag: ""}

		updater := NewMockStatusUpdater(t)
		updater.EXPECT().Name().Return("mock")
		updater.EXPECT().Target().Return(target).Times(3)
		updater.EXPECT().IsAutoUpdateEnabled().Return(true)
		updater.EXPECT().CustomImage().Return("").Once()
		updater.EXPECT().IsPublicRegistryEnabled().Return(true)
		updater.EXPECT().LatestImageInfo(anyCtx).Return(imageInfo, nil).Once()
		updater.EXPECT().CheckForDowngrade(anyCtx, "").Return(false, nil).Once()
		updater.EXPECT().ValidateStatus(anyCtx).Return(errors.New("build version not set")).Once()

		err := versionReconciler.run(t.Context(), updater)
		require.Error(t, err)
		require.NotNil(t, target.LastProbeTimestamp)
		assert.Equal(t, "registry.io/dynatrace/oneagent@sha256:abc123", target.ImageID)
		assert.Empty(t, target.Version)
	})
}

func TestDetermineSource(t *testing.T) {
	customImage := "my.special.image"
	customVersion := "3.2.1.4-5"

	t.Run("custom-image", func(t *testing.T) {
		updater := NewMockStatusUpdater(t)
		updater.EXPECT().CustomImage().Return(customImage).Once()
		source := determineSource(updater)
		assert.Equal(t, status.CustomImageVersionSource, source)
	})
	t.Run("custom-version", func(t *testing.T) {
		updater := NewMockStatusUpdater(t)
		updater.EXPECT().CustomImage().Return("").Once()
		updater.EXPECT().IsPublicRegistryEnabled().Return(false).Once()
		updater.EXPECT().CustomVersion().Return(customVersion).Once()
		source := determineSource(updater)
		assert.Equal(t, status.CustomVersionVersionSource, source)
	})

	t.Run("default", func(t *testing.T) {
		updater := NewMockStatusUpdater(t)
		updater.EXPECT().CustomImage().Return("").Once()
		updater.EXPECT().IsPublicRegistryEnabled().Return(false).Once()
		updater.EXPECT().CustomVersion().Return("").Once()
		source := determineSource(updater)
		assert.Equal(t, status.TenantRegistryVersionSource, source)
	})

	t.Run("public registry enabled", func(t *testing.T) {
		updater := NewMockStatusUpdater(t)
		updater.EXPECT().CustomImage().Return("").Once()
		updater.EXPECT().IsPublicRegistryEnabled().Return(true).Once()
		source := determineSource(updater)
		assert.Equal(t, status.PublicRegistryVersionSource, source)
	})

	t.Run("classicFullStack ignores public registry feature flag", func(t *testing.T) {
		updater := NewMockStatusUpdater(t)
		updater.EXPECT().CustomImage().Return("").Once()
		updater.EXPECT().IsPublicRegistryEnabled().Return(false).Once()
		updater.EXPECT().CustomVersion().Return("").Once()
		source := determineSource(updater)
		assert.Equal(t, status.TenantRegistryVersionSource, source)
	})

	t.Run("classicFullStack ignores public registry feature flag and sets custom image if set", func(t *testing.T) {
		customVersion := "1.2.3.4-5"
		updater := NewMockStatusUpdater(t)
		updater.EXPECT().CustomImage().Return("").Once()
		updater.EXPECT().IsPublicRegistryEnabled().Return(false).Once()
		updater.EXPECT().CustomVersion().Return(customVersion).Once()
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

func newCustomVersionUpdater(t *testing.T, autoUpdate bool) *MockStatusUpdater {
	updater := NewMockStatusUpdater(t)
	updater.EXPECT().IsAutoUpdateEnabled().Return(autoUpdate)
	updater.EXPECT().UseTenantRegistry(anyCtx).Return(nil)
	updater.EXPECT().ValidateStatus(anyCtx).Return(nil)

	return updater
}

func newFailingUpdater(t *testing.T) *MockStatusUpdater {
	updater := NewMockStatusUpdater(t)
	updater.EXPECT().Name().Return("mock").Times(3)
	updater.EXPECT().CustomImage().Return("")
	updater.EXPECT().IsPublicRegistryEnabled().Return(false)
	updater.EXPECT().IsAutoUpdateEnabled().Return(true)
	updater.EXPECT().CustomVersion().Return("")
	updater.EXPECT().UseTenantRegistry(anyCtx).Return(errors.New("BOOM"))

	return updater
}

func newDefaultUpdater(t *testing.T, autoUpdate bool) *MockStatusUpdater {
	updater := NewMockStatusUpdater(t)
	updater.EXPECT().IsAutoUpdateEnabled().Return(autoUpdate)
	updater.EXPECT().UseTenantRegistry(anyCtx).Return(nil)
	updater.EXPECT().ValidateStatus(anyCtx).Return(nil)

	return updater
}

func newPublicRegistryUpdater(t *testing.T, autoUpdate bool) *MockStatusUpdater {
	updater := NewMockStatusUpdater(t)
	updater.EXPECT().IsAutoUpdateEnabled().Return(autoUpdate)
	updater.EXPECT().CustomImage().Return("")
	updater.EXPECT().IsPublicRegistryEnabled().Return(true)

	return updater
}

func assertStatusBasedOnTenantRegistry(t *testing.T, expectedImage, expectedVersion string, versionStatus status.VersionStatus) {
	assert.Equal(t, expectedImage, versionStatus.ImageID)
	assert.Equal(t, expectedVersion, versionStatus.Version)
}
