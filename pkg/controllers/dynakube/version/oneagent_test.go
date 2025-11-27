package version

import (
	"errors"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/exp"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/oneagent"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/conditions"
	dtclientmock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/clients/dynatrace"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestOneAgentUpdater(t *testing.T) {
	ctx := t.Context()
	testImage := dtclient.LatestImageInfo{
		Source: "some.registry.com",
		Tag:    "1.2.3.4-5",
	}

	t.Run("Getters work as expected", func(t *testing.T) {
		dk := &dynakube.DynaKube{
			Spec: dynakube.DynaKubeSpec{
				OneAgent: oneagent.Spec{
					ClassicFullStack: &oneagent.HostInjectSpec{
						Image:   testImage.String(),
						Version: testImage.Tag,
					},
				},
			},
		}
		mockClient := dtclientmock.NewClient(t)
		mockOneAgentImageInfo(mockClient, testImage)

		updater := newOneAgentUpdater(dk, fake.NewClient(), mockClient)

		assert.Equal(t, "oneagent", updater.Name())
		assert.True(t, updater.IsEnabled())
		assert.Equal(t, dk.Spec.OneAgent.ClassicFullStack.Image, updater.CustomImage())
		assert.Equal(t, dk.Spec.OneAgent.ClassicFullStack.Version, updater.CustomVersion())
		assert.False(t, updater.IsAutoUpdateEnabled())
		imageInfo, err := updater.LatestImageInfo(ctx)
		require.NoError(t, err)
		assert.Equal(t, testImage, *imageInfo)
	})
}

func TestOneAgentIsEnabled(t *testing.T) {
	t.Run("cleans up condition if not enabled", func(t *testing.T) {
		dk := &dynakube.DynaKube{
			Status: dynakube.DynaKubeStatus{
				OneAgent: oneagent.Status{
					VersionStatus: status.VersionStatus{
						Version: "prev",
					},
					Healthcheck: newHealthConfig([]string{"run", "this"}),
				},
			},
		}
		setVerifiedCondition(dk.Conditions(), oaConditionType)

		updater := newOneAgentUpdater(dk, nil, nil)

		isEnabled := updater.IsEnabled()
		require.False(t, isEnabled)

		condition := meta.FindStatusCondition(*dk.Conditions(), oaConditionType)
		assert.Nil(t, condition)

		assert.Empty(t, updater.Target())
		assert.Empty(t, dk.Status.OneAgent.Healthcheck)
	})
}

func TestOneAgentPublicRegistry(t *testing.T) {
	t.Run("sets condition if enabled", func(t *testing.T) {
		dk := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					exp.PublicRegistryKey: "true",
				},
			},
		}

		updater := newOneAgentUpdater(dk, nil, nil)

		isEnabled := updater.IsPublicRegistryEnabled()
		require.True(t, isEnabled)

		condition := meta.FindStatusCondition(*dk.Conditions(), oaConditionType)
		require.NotNil(t, condition)
		assert.Equal(t, verifiedReason, condition.Reason)
		assert.Equal(t, metav1.ConditionTrue, condition.Status)
	})
	t.Run("ignores conditions if not enabled", func(t *testing.T) {
		dk := &dynakube.DynaKube{}

		updater := newOneAgentUpdater(dk, nil, nil)

		isEnabled := updater.IsPublicRegistryEnabled()
		require.False(t, isEnabled)

		condition := meta.FindStatusCondition(*dk.Conditions(), oaConditionType)
		require.Nil(t, condition)
	})
}

func TestOneAgentLatestImageInfo(t *testing.T) {
	t.Run("problem with Dynatrace request => visible in conditions", func(t *testing.T) {
		dk := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					exp.PublicRegistryKey: "true",
				},
			},
		}

		mockClient := dtclientmock.NewClient(t)
		mockClient.EXPECT().GetLatestOneAgentImage(mockCtx).Return(nil, errors.New("BOOM")).Once()
		updater := newOneAgentUpdater(dk, nil, mockClient)

		_, err := updater.LatestImageInfo(t.Context())
		require.Error(t, err)

		condition := meta.FindStatusCondition(*dk.Conditions(), oaConditionType)
		require.NotNil(t, condition)
		assert.Equal(t, conditions.DynatraceAPIErrorReason, condition.Reason)
		assert.Equal(t, metav1.ConditionFalse, condition.Status)
	})
}

func TestOneAgentUseDefault(t *testing.T) {
	testVersion := "1.2.3.4-5"

	t.Run("Set according to version field", func(t *testing.T) {
		dk := &dynakube.DynaKube{
			Spec: dynakube.DynaKubeSpec{
				APIURL: testAPIURL,
				OneAgent: oneagent.Spec{
					ClassicFullStack: &oneagent.HostInjectSpec{
						Version: testVersion,
					},
				},
			},
		}
		expectedImage := dk.OneAgent().GetDefaultImage(testVersion)

		mockClient := dtclientmock.NewClient(t)

		updater := newOneAgentUpdater(dk, fake.NewClient(), mockClient)

		err := updater.UseTenantRegistry(t.Context())

		require.NoError(t, err)
		assertStatusBasedOnTenantRegistry(t, expectedImage, testVersion, dk.Status.OneAgent.VersionStatus)
		condition := meta.FindStatusCondition(*dk.Conditions(), oaConditionType)
		assert.Equal(t, verifiedReason, condition.Reason)
		assert.Equal(t, metav1.ConditionTrue, condition.Status)
	})
	t.Run("Set according to default", func(t *testing.T) {
		dk := &dynakube.DynaKube{
			Spec: dynakube.DynaKubeSpec{
				APIURL: testAPIURL,
				OneAgent: oneagent.Spec{
					ClassicFullStack: &oneagent.HostInjectSpec{},
				},
			},
		}
		expectedImage := dk.OneAgent().GetDefaultImage(testVersion)

		mockClient := dtclientmock.NewClient(t)
		mockLatestAgentVersion(mockClient, testVersion, 1)

		updater := newOneAgentUpdater(dk, fake.NewClient(), mockClient)

		err := updater.UseTenantRegistry(t.Context())

		require.NoError(t, err)
		assertStatusBasedOnTenantRegistry(t, expectedImage, testVersion, dk.Status.OneAgent.VersionStatus)
		condition := meta.FindStatusCondition(*dk.Conditions(), oaConditionType)
		assert.Equal(t, verifiedReason, condition.Reason)
		assert.Equal(t, metav1.ConditionTrue, condition.Status)
	})
	t.Run("Don't allow downgrades", func(t *testing.T) {
		previousVersion := "999.999.999.999-999"
		dk := &dynakube.DynaKube{
			Spec: dynakube.DynaKubeSpec{
				APIURL: testAPIURL,
				OneAgent: oneagent.Spec{
					ClassicFullStack: &oneagent.HostInjectSpec{},
				},
			},
			Status: dynakube.DynaKubeStatus{
				OneAgent: oneagent.Status{
					VersionStatus: status.VersionStatus{
						ImageID: "some.registry.com:" + previousVersion,
						Version: previousVersion,
						Source:  status.TenantRegistryVersionSource,
					},
				},
			},
		}

		mockClient := dtclientmock.NewClient(t)
		mockLatestAgentVersion(mockClient, testVersion, 1)

		updater := newOneAgentUpdater(dk, fake.NewClient(), mockClient)

		err := updater.UseTenantRegistry(t.Context())
		require.NoError(t, err) // we only log the downgrade problem, not fail the reconcile
		assert.Equal(t, previousVersion, dk.Status.OneAgent.Version)

		condition := meta.FindStatusCondition(*dk.Conditions(), oaConditionType)
		assert.Equal(t, downgradeReason, condition.Reason)
		assert.Equal(t, metav1.ConditionFalse, condition.Status)
	})

	t.Run("Verification fails due to malformed version", func(t *testing.T) {
		previousVersion := "1.2.3.4444-555"
		dk := &dynakube.DynaKube{
			Spec: dynakube.DynaKubeSpec{
				APIURL: testAPIURL,
				OneAgent: oneagent.Spec{
					ClassicFullStack: &oneagent.HostInjectSpec{},
				},
			},
			Status: dynakube.DynaKubeStatus{
				OneAgent: oneagent.Status{
					VersionStatus: status.VersionStatus{
						ImageID: "some.registry.com:" + previousVersion,
						Version: previousVersion,
						Source:  status.TenantRegistryVersionSource,
					},
				},
			},
		}

		mockClient := dtclientmock.NewClient(t)
		mockLatestAgentVersion(mockClient, "BOOM", 1)

		updater := newOneAgentUpdater(dk, fake.NewClient(), mockClient)

		err := updater.UseTenantRegistry(t.Context())
		require.Error(t, err)
		assert.Equal(t, previousVersion, dk.Status.OneAgent.Version)

		condition := meta.FindStatusCondition(*dk.Conditions(), oaConditionType)
		assert.Equal(t, verificationFailedReason, condition.Reason)
		assert.Equal(t, metav1.ConditionFalse, condition.Status)
	})
	t.Run("Verification fails due to malformed ImageID", func(t *testing.T) {
		dk := &dynakube.DynaKube{
			Spec: dynakube.DynaKubeSpec{
				APIURL: testAPIURL,
				OneAgent: oneagent.Spec{
					CloudNativeFullStack: &oneagent.CloudNativeFullStackSpec{},
				},
			},
			Status: dynakube.DynaKubeStatus{
				OneAgent: oneagent.Status{
					VersionStatus: status.VersionStatus{
						ImageID: "BOOM",
						Source:  status.PublicRegistryVersionSource,
					},
				},
			},
		}

		mockClient := dtclientmock.NewClient(t)
		mockLatestAgentVersion(mockClient, testVersion, 1)

		updater := newOneAgentUpdater(dk, fake.NewClient(), mockClient)

		err := updater.UseTenantRegistry(t.Context())
		require.Error(t, err)

		condition := meta.FindStatusCondition(*dk.Conditions(), oaConditionType)
		assert.Equal(t, verificationFailedReason, condition.Reason)
		assert.Equal(t, metav1.ConditionFalse, condition.Status)
	})
}

type CheckForDowngradeTestCase struct {
	testName    string
	dk          *dynakube.DynaKube
	newVersion  string
	isDowngrade bool
}

func newDynakubeWithOneAgentStatus(status status.VersionStatus) *dynakube.DynaKube {
	return &dynakube.DynaKube{
		Status: dynakube.DynaKubeStatus{
			OneAgent: oneagent.Status{
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
			dk: newDynakubeWithOneAgentStatus(status.VersionStatus{
				ImageID: "does-not-matter",
				Version: newerVersion,
				Source:  status.TenantRegistryVersionSource,
			}),
			newVersion:  olderVersion,
			isDowngrade: true,
		},
		{
			testName: "is downgrade, public registry",
			dk: newDynakubeWithOneAgentStatus(status.VersionStatus{
				ImageID: "some.registry.com:" + newerVersion,
				Source:  status.PublicRegistryVersionSource,
			}),
			newVersion:  olderVersion,
			isDowngrade: true,
		},
		{
			testName: "is NOT downgrade, tenant registry",
			dk: newDynakubeWithOneAgentStatus(status.VersionStatus{
				ImageID: "does-not-matter",
				Version: olderVersion,
				Source:  status.TenantRegistryVersionSource,
			}),
			newVersion:  newerVersion,
			isDowngrade: false,
		},
		{
			testName: "is NOT downgrade, public registry",
			dk: newDynakubeWithOneAgentStatus(status.VersionStatus{
				ImageID: "some.registry.com:" + olderVersion,
				Source:  status.PublicRegistryVersionSource,
			}),
			newVersion:  newerVersion,
			isDowngrade: false,
		},
		{
			testName: "is NOT downgrade, custom image - no logic",
			dk: newDynakubeWithOneAgentStatus(status.VersionStatus{
				ImageID: "some.registry.com:" + newerVersion,
				Source:  status.CustomImageVersionSource,
			}),
			newVersion:  olderVersion,
			isDowngrade: false,
		},
		{
			testName: "is NOT downgrade, custom version - no logic",
			dk: newDynakubeWithOneAgentStatus(status.VersionStatus{
				ImageID: "some.registry.com:" + newerVersion,
				Source:  status.CustomVersionVersionSource,
			}),
			newVersion:  olderVersion,
			isDowngrade: false,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.testName, func(t *testing.T) {
			updater := newOneAgentUpdater(testCase.dk, fake.NewClient(), nil)

			isDowngrade, err := updater.CheckForDowngrade(testCase.newVersion)
			require.NoError(t, err)
			assert.Equal(t, testCase.isDowngrade, isDowngrade)
		})
	}
}

func newDynakubeForCheckLabelTest(versionStatus status.VersionStatus) *dynakube.DynaKube {
	return &dynakube.DynaKube{
		Spec: dynakube.DynaKubeSpec{
			OneAgent: oneagent.Spec{},
		},
		Status: dynakube.DynaKubeStatus{
			OneAgent: oneagent.Status{
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
		dk := newDynakubeForCheckLabelTest(versionStatus)
		dk.Spec.OneAgent.CloudNativeFullStack = &oneagent.CloudNativeFullStackSpec{}
		updater := newOneAgentUpdater(dk, fake.NewClient(), nil)
		require.NoError(t, updater.ValidateStatus())
	})
	t.Run("Validate immutable oneAgent image with classicFullStack", func(t *testing.T) {
		dk := newDynakubeForCheckLabelTest(versionStatus)
		dk.Spec.OneAgent.ClassicFullStack = &oneagent.HostInjectSpec{}
		updater := newOneAgentUpdater(dk, fake.NewClient(), nil)
		require.Error(t, updater.ValidateStatus())
	})
	t.Run("Validate immutable oneAgent image when image version is not set", func(t *testing.T) {
		dk := newDynakubeForCheckLabelTest(versionStatus)
		dk.Spec.OneAgent.CloudNativeFullStack = &oneagent.CloudNativeFullStackSpec{}
		dk.Status.OneAgent.Version = ""
		updater := newOneAgentUpdater(dk, fake.NewClient(), nil)
		require.Error(t, updater.ValidateStatus())
	})
	t.Run("Validate mutable oneAgent image with classicFullStack", func(t *testing.T) {
		dk := newDynakubeForCheckLabelTest(versionStatus)
		dk.Spec.OneAgent.ClassicFullStack = &oneagent.HostInjectSpec{}
		dk.Status.OneAgent.Type = "mutable"
		updater := newOneAgentUpdater(dk, fake.NewClient(), nil)
		require.NoError(t, updater.ValidateStatus())
	})
}
