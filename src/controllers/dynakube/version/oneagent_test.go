package version

import (
	"context"
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/dockerconfig"
	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects/address"
	"github.com/containers/image/v5/docker/reference"
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
						Image:      testImage.String(),
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
		assert.True(t, updater.IsEnabled())
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
		expectedImage := dynakube.DefaultOneAgentImage()
		registry := newFakeRegistryForImages(expectedImage)

		mockClient := &dtclient.MockDynatraceClient{}
		updater := newOneAgentUpdater(dynakube, mockClient, registry.ImageVersionExt)

		err := updater.UseDefaults(context.TODO(), &dockerconfig.DockerConfig{})

		require.NoError(t, err)
		assertDefaultOneAgentStatus(t, registry, getTaggedReference(t, expectedImage), testVersion, dynakube.Status.OneAgent.VersionStatus)
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
		expectedImage := dynakube.DefaultOneAgentImage()
		registry := newFakeRegistryForImages(expectedImage)

		mockClient := &dtclient.MockDynatraceClient{}
		mockLatestAgentVersion(mockClient, testVersion)
		updater := newOneAgentUpdater(dynakube, mockClient, registry.ImageVersionExt)

		err := updater.UseDefaults(context.TODO(), &dockerconfig.DockerConfig{})

		require.NoError(t, err)
		assertDefaultOneAgentStatus(t, registry, getTaggedReference(t, expectedImage), testVersion, dynakube.Status.OneAgent.VersionStatus)
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
						ImageID: "some.registry.com:999.999.999.999-999",
						Version: "999.999.999.999-999",
						Source:  dynatracev1beta1.TenantRegistryVersionSource,
					},
				},
			},
		}

		expectedImage := dynakube.DefaultOneAgentImage()
		registry := newFakeRegistryForImages(expectedImage)

		mockClient := &dtclient.MockDynatraceClient{}
		mockLatestAgentVersion(mockClient, testVersion)
		updater := newOneAgentUpdater(dynakube, mockClient, registry.ImageVersionExt)

		err := updater.UseDefaults(context.TODO(), &dockerconfig.DockerConfig{})
		require.Error(t, err)

		dynakube.Status.OneAgent.Version = ""
		dynakube.Status.OneAgent.Source = dynatracev1beta1.PublicRegistryVersionSource

		err = updater.UseDefaults(context.TODO(), &dockerconfig.DockerConfig{})
		require.Error(t, err)
	})
}

type CheckForDowngradeTestCase struct {
	testName    string
	dynakube    *dynatracev1beta1.DynaKube
	newVersion  string
	isDowngrade bool
}

func newDynakubeWithOneAgentStatus(status dynatracev1beta1.VersionStatus) *dynatracev1beta1.DynaKube {
	return &dynatracev1beta1.DynaKube{
		Status: dynatracev1beta1.DynaKubeStatus{
			OneAgent: dynatracev1beta1.OneAgentStatus{
				VersionStatus: status,
			},
		},
	}
}

func TestCheckForDowngrade(t *testing.T) {
	olderVersion := "1.2.3.4-5"
	newerVersion := "5.4.3.2-1"
	testCases := []CheckForDowngradeTestCase{
		{
			testName: "is downgrade, tenant registry",
			dynakube: newDynakubeWithOneAgentStatus(dynatracev1beta1.VersionStatus{
				ImageID: "does-not-matter",
				Version: newerVersion,
				Source:  dynatracev1beta1.TenantRegistryVersionSource,
			}),
			newVersion:  olderVersion,
			isDowngrade: true,
		},
		{
			testName: "is downgrade, public registry",
			dynakube: newDynakubeWithOneAgentStatus(dynatracev1beta1.VersionStatus{
				ImageID: "some.registry.com:" + newerVersion,
				Source:  dynatracev1beta1.PublicRegistryVersionSource,
			}),
			newVersion:  olderVersion,
			isDowngrade: true,
		},
		{
			testName: "is NOT downgrade, tenant registry",
			dynakube: newDynakubeWithOneAgentStatus(dynatracev1beta1.VersionStatus{
				ImageID: "does-not-matter",
				Version: olderVersion,
				Source:  dynatracev1beta1.TenantRegistryVersionSource,
			}),
			newVersion:  newerVersion,
			isDowngrade: false,
		},
		{
			testName: "is NOT downgrade, public registry",
			dynakube: newDynakubeWithOneAgentStatus(dynatracev1beta1.VersionStatus{
				ImageID: "some.registry.com:" + olderVersion,
				Source:  dynatracev1beta1.PublicRegistryVersionSource,
			}),
			newVersion:  newerVersion,
			isDowngrade: false,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.testName, func(t *testing.T) {
			updater := newOneAgentUpdater(testCase.dynakube, nil, nil)

			isDowngrade, err := updater.CheckForDowngrade(testCase.newVersion)
			require.NoError(t, err)
			assert.Equal(t, testCase.isDowngrade, isDowngrade)
		})
	}
}

func assertDefaultOneAgentStatus(t *testing.T, registry *fakeRegistry, imageRef reference.NamedTagged, expectedVersion string, versionStatus dynatracev1beta1.VersionStatus) { //nolint:revive // argument-limit
	assertVersionStatusEquals(t, registry, imageRef, versionStatus)
	assert.Equal(t, expectedVersion, versionStatus.Version)
}
