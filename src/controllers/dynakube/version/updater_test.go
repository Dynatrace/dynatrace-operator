package version

import (
	"context"
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/dockerconfig"
	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
	"github.com/Dynatrace/dynatrace-operator/src/timeprovider"
	"github.com/containers/image/v5/docker/reference"
	"github.com/opencontainers/go-digest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type mockUpdater struct {
	mock.Mock
}

func (m *mockUpdater) Name() string {
	args := m.Called()
	return args.Get(0).(string)
}
func (m *mockUpdater) IsEnabled() bool {
	args := m.Called()
	return args.Get(0).(bool)
}
func (m *mockUpdater) Target() *dynatracev1beta1.VersionStatus {
	args := m.Called()
	return args.Get(0).(*dynatracev1beta1.VersionStatus)
}
func (m *mockUpdater) CustomImage() string {
	args := m.Called()
	return args.Get(0).(string)
}
func (m *mockUpdater) CustomVersion() string {
	args := m.Called()
	return args.Get(0).(string)
}
func (m *mockUpdater) IsAutoUpdateEnabled() bool {
	args := m.Called()
	return args.Get(0).(bool)
}
func (m *mockUpdater) IsPublicRegistryEnabled() bool {
	args := m.Called()
	return args.Get(0).(bool)
}
func (m *mockUpdater) LatestImageInfo() (*dtclient.LatestImageInfo, error) {
	args := m.Called()
	return args.Get(0).(*dtclient.LatestImageInfo), args.Error(1)
}
func (m *mockUpdater) UseTenantRegistry(_ context.Context, _ *dockerconfig.DockerConfig) error {
	args := m.Called()
	return args.Error(0)
}

func (m *mockUpdater) CheckForDowngrade(latestVersion string) (bool, error) {
	args := m.Called()
	return args.Bool(0), args.Error(1)
}

func TestRun(t *testing.T) {
	ctx := context.TODO()
	testImage := dtclient.LatestImageInfo{
		Source: "some.registry.com",
		Tag:    "1.2.3",
	}
	testDockerCfg := &dockerconfig.DockerConfig{}
	timeProvider := timeprovider.New()

	t.Run("set source and probe at the end, if no error", func(t *testing.T) {
		registry := newFakeRegistry(map[string]ImageVersion{
			testImage.String(): {
				Version: testImage.Tag,
			},
		})
		target := &dynatracev1beta1.VersionStatus{}
		versionReconciler := Reconciler{
			dynakube:     &dynatracev1beta1.DynaKube{},
			timeProvider: timeProvider,
			versionFunc:  registry.ImageVersionExt,
		}
		updater := newCustomImageUpdater(target, testImage.String())
		err := versionReconciler.run(ctx, updater, testDockerCfg)
		require.NoError(t, err)
		assert.Equal(t, timeProvider.Now(), target.LastProbeTimestamp)
		assert.Equal(t, dynatracev1beta1.CustomImageVersionSource, target.Source)
		assertVersionStatusEquals(t, registry, getTaggedReference(t, testImage.String()), *target)
	})
	t.Run("DON'T set source and probe at the end, if error", func(t *testing.T) {
		registry := newEmptyFakeRegistry()
		target := &dynatracev1beta1.VersionStatus{}
		versionReconciler := Reconciler{
			dynakube:     &dynatracev1beta1.DynaKube{},
			timeProvider: timeProvider,
			versionFunc:  registry.ImageVersionExt,
		}
		updater := newCustomImageUpdater(target, "incorrect-uri")
		err := versionReconciler.run(ctx, updater, testDockerCfg)
		require.Error(t, err)
		assert.Nil(t, target.LastProbeTimestamp)
		assert.Empty(t, target.Source)
	})
	t.Run("autoUpdate disabled, runs if status is empty or source changes", func(t *testing.T) {
		registry := newEmptyFakeRegistry()
		target := &dynatracev1beta1.VersionStatus{}
		versionReconciler := Reconciler{
			dynakube:     &dynatracev1beta1.DynaKube{},
			timeProvider: timeProvider,
			versionFunc:  registry.ImageVersionExt,
		}
		updater := newDefaultUpdater(target, false)

		// 1. call => status empty => should run
		err := versionReconciler.run(ctx, updater, testDockerCfg)
		require.NoError(t, err)
		updater.AssertNumberOfCalls(t, "UseTenantRegistry", 1)
		assert.Equal(t, timeProvider.Now(), target.LastProbeTimestamp)
		assert.Equal(t, dynatracev1beta1.TenantRegistryVersionSource, target.Source)

		// 2. call => status NOT empty => should NOT run
		err = versionReconciler.run(ctx, updater, testDockerCfg)
		require.NoError(t, err)
		updater.AssertNumberOfCalls(t, "UseTenantRegistry", 1)

		// 3. call => source is different => should run
		target.Source = dynatracev1beta1.CustomImageVersionSource
		err = versionReconciler.run(ctx, updater, testDockerCfg)
		require.NoError(t, err)
		updater.AssertNumberOfCalls(t, "UseTenantRegistry", 2)

		// 4. call => source is NOT different => should NOT run
		err = versionReconciler.run(ctx, updater, testDockerCfg)
		require.NoError(t, err)
		updater.AssertNumberOfCalls(t, "UseTenantRegistry", 2)
	})
	t.Run("public registry", func(t *testing.T) {
		registry := newFakeRegistry(map[string]ImageVersion{
			testImage.String(): {
				Version: testImage.Tag,
			},
		})
		target := &dynatracev1beta1.VersionStatus{
			Source: dynatracev1beta1.TenantRegistryVersionSource,
		}
		versionReconciler := Reconciler{
			dynakube:     enablePublicRegistry(&dynatracev1beta1.DynaKube{}),
			timeProvider: timeProvider,
			versionFunc:  registry.ImageVersionExt,
		}
		updater := newPublicRegistryUpdater(target, &testImage, false)
		updater.On("IsClassicFullStackEnabled").Return(false)
		updater.On("CheckForDowngrade").Return(false, nil)

		err := versionReconciler.run(ctx, updater, testDockerCfg)
		require.NoError(t, err)
		updater.AssertNumberOfCalls(t, "LatestImageInfo", 1)
		assert.Equal(t, timeProvider.Now(), target.LastProbeTimestamp)
		assert.Equal(t, dynatracev1beta1.PublicRegistryVersionSource, target.Source)
		assertVersionStatusEquals(t, registry, getTaggedReference(t, testImage.String()), *target)
		assert.Empty(t, target.Version)
	})

	t.Run("public registry, no downgrade allowed", func(t *testing.T) {
		registry := newFakeRegistry(map[string]ImageVersion{
			testImage.String(): {
				Version: testImage.Tag,
			},
		})
		target := &dynatracev1beta1.VersionStatus{
			Source: dynatracev1beta1.TenantRegistryVersionSource,
		}
		versionReconciler := Reconciler{
			dynakube:     enablePublicRegistry(&dynatracev1beta1.DynaKube{}),
			timeProvider: timeProvider,
			versionFunc:  registry.ImageVersionExt,
		}
		updater := newPublicRegistryUpdater(target, &testImage, false)
		updater.On("IsClassicFullStackEnabled").Return(false)
		updater.On("CheckForDowngrade").Return(true, nil)

		err := versionReconciler.run(ctx, updater, testDockerCfg)
		require.NoError(t, err)
		updater.AssertNumberOfCalls(t, "LatestImageInfo", 1)
		assert.Equal(t, timeProvider.Now(), target.LastProbeTimestamp)
		assert.Equal(t, dynatracev1beta1.PublicRegistryVersionSource, target.Source)
		assert.Empty(t, target.Version)
		assert.Empty(t, target.ImageID)
	})
	t.Run("classicfullstack enabled, public registry is ignored", func(t *testing.T) {
		registry := newFakeRegistry(map[string]ImageVersion{
			testImage.String(): {
				Version: testImage.Tag,
			},
		})
		target := &dynatracev1beta1.VersionStatus{
			Source: dynatracev1beta1.TenantRegistryVersionSource,
		}
		versionReconciler := Reconciler{
			dynakube:     enablePublicRegistry(newClassicFullStackDynakube()),
			timeProvider: timeProvider,
			versionFunc:  registry.ImageVersionExt,
		}
		updater := newClassicFullStackUpdater(target, false)
		updater.On("CustomImage").Return("")
		updater.On("CustomVersion").Return("")
		updater.On("UseTenantRegistry").Return(nil)

		err := versionReconciler.run(ctx, updater, testDockerCfg)
		require.NoError(t, err)
		updater.AssertNumberOfCalls(t, "LatestImageInfo", 0)
		assert.Equal(t, timeProvider.Now(), target.LastProbeTimestamp)
		assert.Equal(t, dynatracev1beta1.TenantRegistryVersionSource, target.Source)
		assert.Equal(t, target.Version, target.Version)
	})
	t.Run("classicfullstack enabled, public registry is ignored, custom image is set", func(t *testing.T) {
		registry := newFakeRegistry(map[string]ImageVersion{
			testImage.String(): {
				Version: testImage.Tag,
			},
		})
		target := &dynatracev1beta1.VersionStatus{
			Source: dynatracev1beta1.TenantRegistryVersionSource,
		}
		versionReconciler := Reconciler{
			dynakube:     enablePublicRegistry(newClassicFullStackDynakube()),
			timeProvider: timeProvider,
			versionFunc:  registry.ImageVersionExt,
		}
		updater := newClassicFullStackUpdater(target, false)
		updater.On("CustomImage").Return(testImage.String())
		updater.On("CustomVersion").Return(testImage.Tag)
		updater.On("UseTenantRegistry").Return(nil)

		err := versionReconciler.run(ctx, updater, testDockerCfg)
		require.NoError(t, err)
		updater.AssertNumberOfCalls(t, "LatestImageInfo", 0)
		assert.Equal(t, timeProvider.Now(), target.LastProbeTimestamp)
		assert.Equal(t, dynatracev1beta1.CustomImageVersionSource, target.Source)
	})
}

func TestDetermineSource(t *testing.T) {
	customImage := "my.special.image"
	customVersion := "3.2.1"
	t.Run("custom-image", func(t *testing.T) {
		updater := newCustomImageUpdater(nil, customImage)
		source := determineSource(updater)
		assert.Equal(t, dynatracev1beta1.CustomImageVersionSource, source)
	})
	t.Run("custom-version", func(t *testing.T) {
		updater := newCustomVersionUpdater(nil, customVersion, false)
		source := determineSource(updater)
		assert.Equal(t, dynatracev1beta1.CustomVersionVersionSource, source)
	})

	t.Run("public-registry", func(t *testing.T) {
		updater := newPublicRegistryUpdater(nil, nil, false)
		source := determineSource(updater)
		assert.Equal(t, dynatracev1beta1.PublicRegistryVersionSource, source)
	})

	t.Run("default", func(t *testing.T) {
		updater := newDefaultUpdater(nil, true)
		source := determineSource(updater)
		assert.Equal(t, dynatracev1beta1.TenantRegistryVersionSource, source)
	})

	t.Run("classicfullstack ignores public registry feature flag", func(t *testing.T) {
		updater := newClassicFullStackUpdater(nil, false)
		updater.On("CustomImage").Return("")
		updater.On("CustomVersion").Return("")
		source := determineSource(updater)
		assert.Equal(t, dynatracev1beta1.TenantRegistryVersionSource, source)
	})

	t.Run("classicfullstack ignores public registry feature flag and sets custom image if set", func(t *testing.T) {
		testImage := dtclient.LatestImageInfo{
			Source: "some.registry.com",
			Tag:    "1.2.3",
		}
		updater := newClassicFullStackUpdater(nil, false)
		updater.On("CustomImage").Return("")
		updater.On("CustomVersion").Return(testImage.Tag)
		source := determineSource(updater)
		assert.Equal(t, dynatracev1beta1.CustomVersionVersionSource, source)
	})
}

func TestUpdateVersionStatus(t *testing.T) {
	ctx := context.TODO()
	testImage := dtclient.LatestImageInfo{
		Source: "some.registry.com",
		Tag:    "1.2.3",
	}
	testDockerCfg := &dockerconfig.DockerConfig{}

	t.Run("failing to get digest should not cause error, should fall back to using the tag", func(t *testing.T) {
		registry := newEmptyFakeRegistry()
		target := dynatracev1beta1.VersionStatus{}
		err := setImageIDWithDigest(ctx, &target, testImage.String(), registry.ImageVersionExt, testDockerCfg)
		assert.NoError(t, err)
		assert.Equal(t, testImage.String(), target.ImageID)
	})

	t.Run("set status", func(t *testing.T) {
		registry := newFakeRegistry(map[string]ImageVersion{
			testImage.String(): {
				Version: testImage.Tag,
			},
		})
		target := dynatracev1beta1.VersionStatus{}
		err := setImageIDWithDigest(ctx, &target, testImage.String(), registry.ImageVersionExt, testDockerCfg)
		require.NoError(t, err)
		assertVersionStatusEquals(t, registry, getTaggedReference(t, testImage.String()), target)
	})

	t.Run("set status, not call digest func", func(t *testing.T) {
		expectedRepo := "some.registry.com/image"
		expectedDigest := "sha256:7ece13a07a20c77a31cc36906a10ebc90bd47970905ee61e8ed491b7f4c5d62f"
		expectedID := expectedRepo + "@" + expectedDigest
		target := dynatracev1beta1.VersionStatus{}
		boomFunc := func(_ context.Context, imagePath string, _ *dockerconfig.DockerConfig) (ImageVersion, error) {
			t.Error("digest function was called unexpectedly")
			return ImageVersion{}, nil
		}
		err := setImageIDWithDigest(ctx, &target, expectedID, boomFunc, testDockerCfg)
		require.NoError(t, err)
		assert.Equal(t, expectedID, target.ImageID)
	})
	t.Run("accept tagged + digest image reference", func(t *testing.T) {
		expectedRepo := "some.registry.com/image"
		expectedDigest := "sha256:7ece13a07a20c77a31cc36906a10ebc90bd47970905ee61e8ed491b7f4c5d62f"
		expectedID := expectedRepo + ":tag@" + expectedDigest
		target := dynatracev1beta1.VersionStatus{}
		boomFunc := func(_ context.Context, imagePath string, _ *dockerconfig.DockerConfig) (ImageVersion, error) {
			t.Error("digest function was called unexpectedly")
			return ImageVersion{}, nil
		}
		err := setImageIDWithDigest(ctx, &target, expectedID, boomFunc, testDockerCfg)
		require.NoError(t, err)
		assert.Equal(t, expectedID, target.ImageID)
	})
}

func TestGetTagFromImageID(t *testing.T) {
	t.Run("get tag from imageID", func(t *testing.T) {
		imageID := "some.registry.com:1.2.3@sha256:7ece13a07a20c77a31cc36906a10ebc90bd47970905ee61e8ed491b7f4c5d62f"

		tag, err := getTagFromImageID(imageID)

		require.NoError(t, err)
		assert.Equal(t, "1.2.3", tag)
	})
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

func newCustomImageUpdater(target *dynatracev1beta1.VersionStatus, image string) *mockUpdater {
	updater := newBaseUpdater(target, true)
	updater.On("CustomImage").Return(image)
	return updater
}

func newCustomVersionUpdater(target *dynatracev1beta1.VersionStatus, version string, autoUpdate bool) *mockUpdater {
	updater := newBaseUpdater(target, autoUpdate)
	updater.On("CustomImage").Return("")
	updater.On("IsPublicRegistryEnabled").Return(false)
	updater.On("CustomVersion").Return(version)
	return updater
}

func newDefaultUpdater(target *dynatracev1beta1.VersionStatus, autoUpdate bool) *mockUpdater {
	updater := newBaseUpdater(target, autoUpdate)
	updater.On("CustomImage").Return("")
	updater.On("IsPublicRegistryEnabled").Return(false)
	updater.On("CustomVersion").Return("")
	updater.On("UseTenantRegistry").Return(nil)
	return updater
}

func newPublicRegistryUpdater(target *dynatracev1beta1.VersionStatus, imageInfo *dtclient.LatestImageInfo, autoUpdate bool) *mockUpdater {
	updater := newBaseUpdater(target, autoUpdate)
	updater.On("CustomImage").Return("")
	updater.On("IsPublicRegistryEnabled").Return(true)
	updater.On("LatestImageInfo").Return(imageInfo, nil)
	return updater
}

func newClassicFullStackUpdater(target *dynatracev1beta1.VersionStatus, autoUpdate bool) *mockUpdater {
	updater := newBaseUpdater(target, autoUpdate)
	updater.On("IsPublicRegistryEnabled").Return(false)
	return updater
}

func newBaseUpdater(target *dynatracev1beta1.VersionStatus, autoUpdate bool) *mockUpdater {
	updater := mockUpdater{}
	updater.On("Name").Return("mock")
	updater.On("Target").Return(target)
	updater.On("IsEnabled").Return(true)
	updater.On("IsAutoUpdateEnabled").Return(autoUpdate)
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

func getTaggedReference(t *testing.T, image string) reference.NamedTagged {
	ref, err := reference.Parse(image)
	require.NoError(t, err)
	taggedRef, ok := ref.(reference.NamedTagged)
	require.True(t, ok)
	return taggedRef
}

func assertStatusBasedOnTenantRegistry(t *testing.T, expectedImage, expectedVersion string, versionStatus dynatracev1beta1.VersionStatus) { //nolint:revive // argument-limit
	assert.Equal(t, expectedImage, versionStatus.ImageID)
	assert.Equal(t, expectedVersion, versionStatus.Version)
}

func getTestDigest() digest.Digest {
	return digest.FromString("sha256:7ece13a07a20c77a31cc36906a10ebc90bd47970905ee61e8ed491b7f4c5d62f")
}
