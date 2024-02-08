package version

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/address"
	mockedclient "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/clients/dynatrace"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOneAgentUpdater(t *testing.T) {
	ctx := context.Background()
	testImage := dtclient.LatestImageInfo{
		Source: "some.registry.com",
		Tag:    "1.2.3.4-5",
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
		mockClient := mockedclient.NewClient(t)
		mockOneAgentImageInfo(mockClient, testImage)

		updater := newOneAgentUpdater(dynakube, fake.NewClient(), mockClient)

		assert.Equal(t, "oneagent", updater.Name())
		assert.True(t, updater.IsEnabled())
		assert.Equal(t, dynakube.Spec.OneAgent.ClassicFullStack.Image, updater.CustomImage())
		assert.Equal(t, dynakube.Spec.OneAgent.ClassicFullStack.Version, updater.CustomVersion())
		assert.False(t, updater.IsAutoUpdateEnabled())
		imageInfo, err := updater.LatestImageInfo(ctx)
		require.NoError(t, err)
		assert.Equal(t, testImage, *imageInfo)
	})
}

func TestOneAgentUseDefault(t *testing.T) {
	testVersion := "1.2.3.4-5"

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

		mockClient := mockedclient.NewClient(t)

		updater := newOneAgentUpdater(dynakube, fake.NewClient(), mockClient)

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

		mockClient := mockedclient.NewClient(t)
		mockLatestAgentVersion(mockClient, testVersion)

		updater := newOneAgentUpdater(dynakube, fake.NewClient(), mockClient)

		err := updater.UseTenantRegistry(context.Background())

		require.NoError(t, err)
		assertStatusBasedOnTenantRegistry(t, expectedImage, testVersion, dynakube.Status.OneAgent.VersionStatus)
	})
	t.Run("Don't allow downgrades", func(t *testing.T) {
		previousVersion := "999.999.999.999-999"
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
						ImageID: "some.registry.com:" + previousVersion,
						Version: previousVersion,
						Source:  status.TenantRegistryVersionSource,
					},
				},
			},
		}

		mockClient := mockedclient.NewClient(t)
		mockLatestAgentVersion(mockClient, testVersion)

		updater := newOneAgentUpdater(dynakube, fake.NewClient(), mockClient)

		err := updater.UseTenantRegistry(context.Background())
		require.NoError(t, err) // we only log the downgrade problem, not fail the reconcile
		assert.Equal(t, previousVersion, dynakube.Status.OneAgent.Version)
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
		{
			testName: "is NOT downgrade, custom image - no logic",
			dynakube: newDynakubeWithOneAgentStatus(status.VersionStatus{
				ImageID: "some.registry.com:" + newerVersion,
				Source:  status.CustomImageVersionSource,
			}),
			newVersion:  olderVersion,
			isDowngrade: false,
		},
		{
			testName: "is NOT downgrade, custom version - no logic",
			dynakube: newDynakubeWithOneAgentStatus(status.VersionStatus{
				ImageID: "some.registry.com:" + newerVersion,
				Source:  status.CustomVersionVersionSource,
			}),
			newVersion:  olderVersion,
			isDowngrade: false,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.testName, func(t *testing.T) {
			updater := newOneAgentUpdater(testCase.dynakube, fake.NewClient(), nil)

			isDowngrade, err := updater.CheckForDowngrade(testCase.newVersion)
			require.NoError(t, err)
			assert.Equal(t, testCase.isDowngrade, isDowngrade)
		})
	}
}

func newDynakubeForCheckLabelTest(versionStatus status.VersionStatus) *dynatracev1beta1.DynaKube {
	return &dynatracev1beta1.DynaKube{
		Spec: dynatracev1beta1.DynaKubeSpec{
			OneAgent: dynatracev1beta1.OneAgentSpec{},
		},
		Status: dynatracev1beta1.DynaKubeStatus{
			OneAgent: dynatracev1beta1.OneAgentStatus{
				VersionStatus: versionStatus,
			},
		},
	}
}

func TestCheckLabels(t *testing.T) {
	imageType := "immutable"
	imageVersion := "1.2.3.4-5"
	versionStatus := status.VersionStatus{
		ImageID: "some.registry.com:" + imageVersion,
		Source:  status.TenantRegistryVersionSource,
		Type:    imageType,
		Version: imageVersion,
	}

	t.Run("Validate immutable oneAgent image with default cloudNative", func(t *testing.T) {
		dynakube := newDynakubeForCheckLabelTest(versionStatus)
		dynakube.Spec.OneAgent.CloudNativeFullStack = &dynatracev1beta1.CloudNativeFullStackSpec{}
		updater := newOneAgentUpdater(dynakube, fake.NewClient(), nil)
		require.NoError(t, updater.ValidateStatus())
	})
	t.Run("Validate immutable oneAgent image with classicFullStack", func(t *testing.T) {
		dynakube := newDynakubeForCheckLabelTest(versionStatus)
		dynakube.Spec.OneAgent.ClassicFullStack = &dynatracev1beta1.HostInjectSpec{}
		updater := newOneAgentUpdater(dynakube, fake.NewClient(), nil)
		require.Error(t, updater.ValidateStatus())
	})
	t.Run("Validate immutable oneAgent image when image version is not set", func(t *testing.T) {
		dynakube := newDynakubeForCheckLabelTest(versionStatus)
		dynakube.Spec.OneAgent.CloudNativeFullStack = &dynatracev1beta1.CloudNativeFullStackSpec{}
		dynakube.Status.OneAgent.VersionStatus.Version = ""
		updater := newOneAgentUpdater(dynakube, fake.NewClient(), nil)
		require.Error(t, updater.ValidateStatus())
	})
	t.Run("Validate mutable oneAgent image with classicFullStack", func(t *testing.T) {
		dynakube := newDynakubeForCheckLabelTest(versionStatus)
		dynakube.Spec.OneAgent.ClassicFullStack = &dynatracev1beta1.HostInjectSpec{}
		dynakube.Status.OneAgent.VersionStatus.Type = "mutable"
		updater := newOneAgentUpdater(dynakube, fake.NewClient(), nil)
		require.NoError(t, updater.ValidateStatus())
	})
}
