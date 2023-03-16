package version

import (
	"context"
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/dockerconfig"
	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
	"github.com/Dynatrace/dynatrace-operator/src/timeprovider"
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
func (m *mockUpdater) Enabled() bool {
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
func (m *mockUpdater) LatestImageInfo() (*dtclient.LatestImageInfo, error) {
	args := m.Called()
	return args.Get(0).(*dtclient.LatestImageInfo), args.Error(1)
}
func (m *mockUpdater) UseDefaults(_ context.Context, _ *dockerconfig.DockerConfig) error {
	args := m.Called()
	return args.Error(0)
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
		registry := newFakeRegistryForImages(testImage)
		target := &dynatracev1beta1.VersionStatus{}
		versionReconciler := Reconciler{
			dynakube:     &dynatracev1beta1.DynaKube{},
			timeProvider: timeProvider,
			hashFunc:     registry.ImageVersionExt,
		}
		updater := newCustomImageUpdater(target, testImage.Uri())
		err := versionReconciler.run(ctx, updater, testDockerCfg)
		require.NoError(t, err)
		assert.Equal(t, timeProvider.Now(), target.LastProbeTimestamp)
		assert.Equal(t, dynatracev1beta1.CustomImageVersionSource, target.Source)
		assertVersionStatusEquals(t, registry, testImage, *target)
	})
	t.Run("DON'T set source and probe at the end, if error", func(t *testing.T) {
		registry := newEmptyFakeRegistry()
		target := &dynatracev1beta1.VersionStatus{}
		versionReconciler := Reconciler{
			dynakube:     &dynatracev1beta1.DynaKube{},
			timeProvider: timeProvider,
			hashFunc:     registry.ImageVersionExt,
		}
		updater := newCustomImageUpdater(target, testImage.Uri())
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
			hashFunc:     registry.ImageVersionExt,
		}
		updater := newDefaultUpdater(target, false)

		// 1. call => status empty => should run
		err := versionReconciler.run(ctx, updater, testDockerCfg)
		require.NoError(t, err)
		updater.AssertNumberOfCalls(t, "UseDefaults", 1)
		assert.Equal(t, timeProvider.Now(), target.LastProbeTimestamp)
		assert.Equal(t, dynatracev1beta1.DefaultVersionSource, target.Source)

		// 2. call => status NOT empty => should NOT run
		err = versionReconciler.run(ctx, updater, testDockerCfg)
		require.NoError(t, err)
		updater.AssertNumberOfCalls(t, "UseDefaults", 1)

		// 3. call => source is different => should run
		target.Source = dynatracev1beta1.CustomImageVersionSource
		err = versionReconciler.run(ctx, updater, testDockerCfg)
		require.NoError(t, err)
		updater.AssertNumberOfCalls(t, "UseDefaults", 2)

		// 4. call => source is NOT different => should NOT run
		err = versionReconciler.run(ctx, updater, testDockerCfg)
		require.NoError(t, err)
		updater.AssertNumberOfCalls(t, "UseDefaults", 2)
	})
	t.Run("public registry, version set to imageTag", func(t *testing.T) {
		registry := newFakeRegistryForImages(testImage)
		target := &dynatracev1beta1.VersionStatus{
			Source: dynatracev1beta1.DefaultVersionSource,
		}
		versionReconciler := Reconciler{
			dynakube:     enablePublicRegistry(&dynatracev1beta1.DynaKube{}),
			timeProvider: timeProvider,
			hashFunc:     registry.ImageVersionExt,
		}
		updater := newPublicRegistryUpdater(target, &testImage, false)

		err := versionReconciler.run(ctx, updater, testDockerCfg)
		require.NoError(t, err)
		updater.AssertNumberOfCalls(t, "LatestImageInfo", 1)
		assert.Equal(t, timeProvider.Now(), target.LastProbeTimestamp)
		assert.Equal(t, dynatracev1beta1.PublicRegistryVersionSource, target.Source)
		assertVersionStatusEquals(t, registry, testImage, *target)
		assert.Equal(t, target.ImageTag, target.Version)
	})
}

func TestDetermineSource(t *testing.T) {
	customImage := "my.special.image"
	customVersion := "3.2.1"
	t.Run("custom-image", func(t *testing.T) {
		updater := newCustomImageUpdater(nil, customImage)

		versionReconciler := Reconciler{
			dynakube: &dynatracev1beta1.DynaKube{},
		}
		source := versionReconciler.determineSource(updater)
		assert.Equal(t, source, dynatracev1beta1.CustomImageVersionSource)
	})
	t.Run("custom-version", func(t *testing.T) {
		updater := newCustomVersionUpdater(nil, customVersion, false)

		versionReconciler := Reconciler{
			dynakube: &dynatracev1beta1.DynaKube{},
		}
		source := versionReconciler.determineSource(updater)
		assert.Equal(t, source, dynatracev1beta1.CustomVersionVersionSource)
	})

	t.Run("public-registry", func(t *testing.T) {
		updater := newPublicRegistryUpdater(nil, nil, false)

		versionReconciler := Reconciler{
			dynakube: enablePublicRegistry(&dynatracev1beta1.DynaKube{}),
		}
		source := versionReconciler.determineSource(updater)
		assert.Equal(t, source, dynatracev1beta1.PublicRegistryVersionSource)
	})

	t.Run("default", func(t *testing.T) {
		updater := newDefaultUpdater(nil, true)

		versionReconciler := Reconciler{
			dynakube: &dynatracev1beta1.DynaKube{},
		}
		source := versionReconciler.determineSource(updater)
		assert.Equal(t, source, dynatracev1beta1.DefaultVersionSource)
	})

	t.Run("custom-image overrules public-registry", func(t *testing.T) {
		updater := newCustomImageUpdater(nil, customImage)

		versionReconciler := Reconciler{
			dynakube: enablePublicRegistry(&dynatracev1beta1.DynaKube{}),
		}
		source := versionReconciler.determineSource(updater)
		assert.Equal(t, source, dynatracev1beta1.CustomImageVersionSource)
	})

	t.Run("custom-version overruled by public-registry", func(t *testing.T) {
		updater := newCustomVersionUpdater(nil, customVersion, true)

		versionReconciler := Reconciler{
			dynakube: enablePublicRegistry(&dynatracev1beta1.DynaKube{}),
		}
		source := versionReconciler.determineSource(updater)
		assert.Equal(t, source, dynatracev1beta1.PublicRegistryVersionSource)
	})
}

func TestUpdateVersionStatus(t *testing.T) {
	ctx := context.TODO()
	testImage := dtclient.LatestImageInfo{
		Source: "some.registry.com",
		Tag:    "1.2.3",
	}
	testDockerCfg := &dockerconfig.DockerConfig{}

	t.Run("missing image", func(t *testing.T) {
		registry := newEmptyFakeRegistry()
		target := dynatracev1beta1.VersionStatus{}
		err := updateVersionStatus(ctx, &target, &testImage, registry.ImageVersionExt, testDockerCfg)
		assert.Error(t, err)
	})

	t.Run("set status", func(t *testing.T) {
		registry := newFakeRegistryForImages(testImage)
		target := dynatracev1beta1.VersionStatus{}
		err := updateVersionStatus(ctx, &target, &testImage, registry.ImageVersionExt, testDockerCfg)
		require.NoError(t, err)
		assertVersionStatusEquals(t, registry, testImage, target)
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
	updater := newBaseUpdater(true)
	updater.On("CustomImage").Return(image)
	updater.On("CustomVersion").Return("")
	updater.On("Target").Return(target)
	return updater
}

func newCustomVersionUpdater(target *dynatracev1beta1.VersionStatus, version string, autoUpdate bool) *mockUpdater {
	updater := newBaseUpdater(autoUpdate)
	updater.On("CustomImage").Return("")
	updater.On("CustomVersion").Return(version)
	updater.On("Target").Return(target)
	return updater
}

func newDefaultUpdater(target *dynatracev1beta1.VersionStatus, autoUpdate bool) *mockUpdater {
	updater := newBaseUpdater(autoUpdate)
	updater.On("CustomImage").Return("")
	updater.On("CustomVersion").Return("")
	updater.On("UseDefaults").Return(nil)
	updater.On("Target").Return(target)
	return updater
}

func newPublicRegistryUpdater(target *dynatracev1beta1.VersionStatus, imageInfo *dtclient.LatestImageInfo, autoUpdate bool) *mockUpdater {
	updater := newBaseUpdater(autoUpdate)
	updater.On("CustomImage").Return("")
	updater.On("CustomVersion").Return("")
	updater.On("LatestImageInfo").Return(imageInfo, nil)
	updater.On("Target").Return(target)
	return updater
}

func newBaseUpdater(autoUpdate bool) *mockUpdater {
	updater := mockUpdater{}
	updater.On("Name").Return("mock")
	updater.On("IsAutoUpdateEnabled").Return(autoUpdate)
	return &updater
}
