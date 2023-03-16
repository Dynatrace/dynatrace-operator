package version

import (
	"context"
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/dockerconfig"
	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects/address"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOneAgentUpdater(t *testing.T) {
	testImage := dtclient.LatestImageInfo{
		Source: "some.registry.com",
		Tag:    "1.2.3",
	}
	t.Run("Getters work as expected", func(t *testing.T) {
		dynakube := &dynatracev1beta1.DynaKube{
			Spec: dynatracev1beta1.DynaKubeSpec{
				OneAgent: dynatracev1beta1.OneAgentSpec{
					ClassicFullStack: &dynatracev1beta1.HostInjectSpec{
						AutoUpdate: address.Of(false),
						Image:      testImage.Uri(),
						Version:    testImage.Tag,
					},
				},
			},
		}
		registry := newEmptyFakeRegistry()
		mockClient := &dtclient.MockDynatraceClient{}
		mockOneAgentImageInfo(mockClient, testImage)
		updater := newOneAgentUpdater(dynakube, mockClient, registry.ImageVersionExt)

		assert.Equal(t, "oneagent", updater.Name())
		assert.True(t, updater.Enabled())
		assert.Equal(t, dynakube.Spec.OneAgent.ClassicFullStack.Image, updater.CustomImage())
		assert.Equal(t, dynakube.Spec.OneAgent.ClassicFullStack.Version, updater.CustomVersion())
		assert.False(t, updater.IsAutoUpdateEnabled())
		imageInfo, err := updater.LatestImageInfo()
		require.NoError(t, err)
		assert.Equal(t, testImage, *imageInfo)
	})
}

func TestOneAgentUseDefault(t *testing.T) {
	testVersion := "1.2.3"
	t.Run("Set according to version field", func(t *testing.T) {
		dynakube := &dynatracev1beta1.DynaKube{
			Spec: dynatracev1beta1.DynaKubeSpec{
				APIURL: testApiUrl,
				OneAgent: dynatracev1beta1.OneAgentSpec{
					ClassicFullStack: &dynatracev1beta1.HostInjectSpec{
						Version: testVersion,
					},
				},
			},
		}
		expectedImage := dtclient.ImageInfoFromUri(dynakube.DefaultOneAgentImage())
		registry := newFakeRegistryForImages(*expectedImage)

		mockClient := &dtclient.MockDynatraceClient{}
		updater := newOneAgentUpdater(dynakube, mockClient, registry.ImageVersionExt)

		err := updater.UseDefaults(context.TODO(), &dockerconfig.DockerConfig{})

		require.NoError(t, err)
		assertDefaultOneAgentStatus(t, registry, *expectedImage, testVersion, dynakube.Status.OneAgent.VersionStatus)
	})
	t.Run("Set according to default", func(t *testing.T) {
		dynakube := &dynatracev1beta1.DynaKube{
			Spec: dynatracev1beta1.DynaKubeSpec{
				APIURL: testApiUrl,
				OneAgent: dynatracev1beta1.OneAgentSpec{
					ClassicFullStack: &dynatracev1beta1.HostInjectSpec{},
				},
			},
		}
		expectedImage := dtclient.ImageInfoFromUri(dynakube.DefaultOneAgentImage())
		registry := newFakeRegistryForImages(*expectedImage)

		mockClient := &dtclient.MockDynatraceClient{}
		mockLatestAgentVersion(mockClient, testVersion)
		updater := newOneAgentUpdater(dynakube, mockClient, registry.ImageVersionExt)

		err := updater.UseDefaults(context.TODO(), &dockerconfig.DockerConfig{})

		require.NoError(t, err)
		assertDefaultOneAgentStatus(t, registry, *expectedImage, testVersion, dynakube.Status.OneAgent.VersionStatus)
	})
	t.Run("Don't allow downgrades", func(t *testing.T) {
		dynakube := &dynatracev1beta1.DynaKube{
			Spec: dynatracev1beta1.DynaKubeSpec{
				APIURL: testApiUrl,
				OneAgent: dynatracev1beta1.OneAgentSpec{
					ClassicFullStack: &dynatracev1beta1.HostInjectSpec{},
				},
			},
			Status: dynatracev1beta1.DynaKubeStatus{
				OneAgent: dynatracev1beta1.OneAgentStatus{
					VersionStatus: dynatracev1beta1.VersionStatus{
						Version: "999.999.999",
					},
				},
			},
		}
		expectedImage := dtclient.ImageInfoFromUri(dynakube.DefaultOneAgentImage())
		registry := newFakeRegistryForImages(*expectedImage)

		mockClient := &dtclient.MockDynatraceClient{}
		mockLatestAgentVersion(mockClient, testVersion)
		updater := newOneAgentUpdater(dynakube, mockClient, registry.ImageVersionExt)

		err := updater.UseDefaults(context.TODO(), &dockerconfig.DockerConfig{})

		require.Error(t, err)
	})
}

func assertDefaultOneAgentStatus(t *testing.T, registry *fakeRegistry, image dtclient.LatestImageInfo, expectedVersion string, versionStatus dynatracev1beta1.VersionStatus) { //nolint:revive // argument-limit
	assertVersionStatusEquals(t, registry, image, versionStatus)
	assert.Equal(t, expectedVersion, versionStatus.Version)
}
