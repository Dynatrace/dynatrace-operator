package version

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/src/api/status"
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1/dynakube"
	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects/address"
	"github.com/Dynatrace/dynatrace-operator/src/registry"
	"github.com/Dynatrace/dynatrace-operator/src/scheme/fake"
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
		updater := newOneAgentUpdater(dynakube, fake.NewClient(), mockClient, registry.ImageVersionExt)

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
	testDigest := getTestDigest()
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
		registry := newFakeRegistry(map[string]registry.ImageVersion{
			expectedImage: {
				Version: testVersion,
				Digest:  testDigest,
			},
		})

		mockClient := &dtclient.MockDynatraceClient{}
		updater := newOneAgentUpdater(dynakube, fake.NewClient(), mockClient, registry.ImageVersionExt)

		err := updater.UseTenantRegistry(context.TODO())

		require.NoError(t, err)
		assertStatusBasedOnTenantRegistry(t, expectedImage, testVersion, dynakube.Status.OneAgent.VersionStatus)
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
		registry := newFakeRegistry(map[string]registry.ImageVersion{
			expectedImage: {
				Version: testVersion,
				Digest:  testDigest,
			},
		})

		mockClient := &dtclient.MockDynatraceClient{}
		mockLatestAgentVersion(mockClient, testVersion)
		updater := newOneAgentUpdater(dynakube, fake.NewClient(), mockClient, registry.ImageVersionExt)

		err := updater.UseTenantRegistry(context.TODO())

		require.NoError(t, err)
		assertStatusBasedOnTenantRegistry(t, expectedImage, testVersion, dynakube.Status.OneAgent.VersionStatus)
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
					VersionStatus: status.VersionStatus{
						ImageID: "some.registry.com:999.999.999.999-999",
						Version: "999.999.999.999-999",
						Source:  status.TenantRegistryVersionSource,
					},
				},
			},
		}

		expectedImage := dynakube.DefaultOneAgentImage()
		registry := newFakeRegistry(map[string]registry.ImageVersion{
			expectedImage: {
				Version: testVersion,
				Digest:  testDigest,
			},
		})

		mockClient := &dtclient.MockDynatraceClient{}
		mockLatestAgentVersion(mockClient, testVersion)
		updater := newOneAgentUpdater(dynakube, fake.NewClient(), mockClient, registry.ImageVersionExt)

		err := updater.UseTenantRegistry(context.TODO())
		require.Error(t, err)

		dynakube.Status.OneAgent.Version = ""
		dynakube.Status.OneAgent.Source = status.PublicRegistryVersionSource

		err = updater.UseTenantRegistry(context.TODO())
		require.Error(t, err)
	})
}

type CheckForDowngradeTestCase struct {
	testName    string
	dynakube    *dynatracev1beta1.DynaKube
	newVersion  string
	isDowngrade bool
}

func newDynakubeWithOneAgentStatus(status status.VersionStatus) *dynatracev1beta1.DynaKube {
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
			dynakube: newDynakubeWithOneAgentStatus(status.VersionStatus{
				ImageID: "does-not-matter",
				Version: newerVersion,
				Source:  status.TenantRegistryVersionSource,
			}),
			newVersion:  olderVersion,
			isDowngrade: true,
		},
		{
			testName: "is downgrade, public registry",
			dynakube: newDynakubeWithOneAgentStatus(status.VersionStatus{
				ImageID: "some.registry.com:" + newerVersion,
				Source:  status.PublicRegistryVersionSource,
			}),
			newVersion:  olderVersion,
			isDowngrade: true,
		},
		{
			testName: "is NOT downgrade, tenant registry",
			dynakube: newDynakubeWithOneAgentStatus(status.VersionStatus{
				ImageID: "does-not-matter",
				Version: olderVersion,
				Source:  status.TenantRegistryVersionSource,
			}),
			newVersion:  newerVersion,
			isDowngrade: false,
		},
		{
			testName: "is NOT downgrade, public registry",
			dynakube: newDynakubeWithOneAgentStatus(status.VersionStatus{
				ImageID: "some.registry.com:" + olderVersion,
				Source:  status.PublicRegistryVersionSource,
			}),
			newVersion:  newerVersion,
			isDowngrade: false,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.testName, func(t *testing.T) {
			updater := newOneAgentUpdater(testCase.dynakube, fake.NewClient(), nil, nil)

			isDowngrade, err := updater.CheckForDowngrade(testCase.newVersion)
			require.NoError(t, err)
			assert.Equal(t, testCase.isDowngrade, isDowngrade)
		})
	}
}
