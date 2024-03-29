package version

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/conditions"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/address"
	dtclientmock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/clients/dynatrace"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
		mockClient := dtclientmock.NewClient(t)
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

func TestOneAgentIsEnabled(t *testing.T) {
	t.Run("cleans up condition if not enabled", func(t *testing.T) {
		dynakube := &dynatracev1beta1.DynaKube{
			Status: dynatracev1beta1.DynaKubeStatus{
				OneAgent: dynatracev1beta1.OneAgentStatus{
					VersionStatus: status.VersionStatus{
						Version: "prev",
					},
					Healthcheck: newHealthConfig([]string{"run", "this"}),
				},
			},
		}
		setVerifiedCondition(dynakube.Conditions(), oaConditionType)

		updater := newOneAgentUpdater(dynakube, nil, nil)

		isEnabled := updater.IsEnabled()
		require.False(t, isEnabled)

		condition := meta.FindStatusCondition(*dynakube.Conditions(), oaConditionType)
		assert.Nil(t, condition)

		assert.Empty(t, updater.Target())
		assert.Empty(t, dynakube.Status.OneAgent.Healthcheck)
	})
}

func TestOneAgentPublicRegistry(t *testing.T) {
	t.Run("sets condition if enabled", func(t *testing.T) {
		dynakube := &dynatracev1beta1.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					dynatracev1beta1.AnnotationFeaturePublicRegistry: "true",
				},
			},
		}

		updater := newOneAgentUpdater(dynakube, nil, nil)

		isEnabled := updater.IsPublicRegistryEnabled()
		require.True(t, isEnabled)

		condition := meta.FindStatusCondition(*dynakube.Conditions(), oaConditionType)
		require.NotNil(t, condition)
		assert.Equal(t, verifiedReason, condition.Reason)
		assert.Equal(t, metav1.ConditionTrue, condition.Status)
	})
	t.Run("ignores conditions if not enabled", func(t *testing.T) {
		dynakube := &dynatracev1beta1.DynaKube{}

		updater := newOneAgentUpdater(dynakube, nil, nil)

		isEnabled := updater.IsPublicRegistryEnabled()
		require.False(t, isEnabled)

		condition := meta.FindStatusCondition(*dynakube.Conditions(), oaConditionType)
		require.Nil(t, condition)
	})
}

func TestOneAgentLatestImageInfo(t *testing.T) {
	t.Run("problem with Dynatrace request => visible in conditions", func(t *testing.T) {
		dynakube := &dynatracev1beta1.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					dynatracev1beta1.AnnotationFeaturePublicRegistry: "true",
				},
			},
		}

		updater := newOneAgentUpdater(dynakube, nil, createErrorDTClient(t))

		_, err := updater.LatestImageInfo(context.Background())
		require.Error(t, err)

		condition := meta.FindStatusCondition(*dynakube.Conditions(), oaConditionType)
		require.NotNil(t, condition)
		assert.Equal(t, conditions.DynatraceApiErrorReason, condition.Reason)
		assert.Equal(t, metav1.ConditionFalse, condition.Status)
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
		expectedImage := dynakube.DefaultOneAgentImage(testVersion)

		mockClient := dtclientmock.NewClient(t)

		updater := newOneAgentUpdater(dynakube, fake.NewClient(), mockClient)

		err := updater.UseTenantRegistry(context.TODO())

		require.NoError(t, err)
		assertStatusBasedOnTenantRegistry(t, expectedImage, testVersion, dynakube.Status.OneAgent.VersionStatus)
		condition := meta.FindStatusCondition(*dynakube.Conditions(), oaConditionType)
		assert.Equal(t, verifiedReason, condition.Reason)
		assert.Equal(t, metav1.ConditionTrue, condition.Status)
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
		expectedImage := dynakube.DefaultOneAgentImage(testVersion)

		mockClient := dtclientmock.NewClient(t)
		mockLatestAgentVersion(mockClient, testVersion)

		updater := newOneAgentUpdater(dynakube, fake.NewClient(), mockClient)

		err := updater.UseTenantRegistry(context.Background())

		require.NoError(t, err)
		assertStatusBasedOnTenantRegistry(t, expectedImage, testVersion, dynakube.Status.OneAgent.VersionStatus)
		condition := meta.FindStatusCondition(*dynakube.Conditions(), oaConditionType)
		assert.Equal(t, verifiedReason, condition.Reason)
		assert.Equal(t, metav1.ConditionTrue, condition.Status)
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

		mockClient := dtclientmock.NewClient(t)
		mockLatestAgentVersion(mockClient, testVersion)

		updater := newOneAgentUpdater(dynakube, fake.NewClient(), mockClient)

		err := updater.UseTenantRegistry(context.Background())
		require.NoError(t, err) // we only log the downgrade problem, not fail the reconcile
		assert.Equal(t, previousVersion, dynakube.Status.OneAgent.Version)

		condition := meta.FindStatusCondition(*dynakube.Conditions(), oaConditionType)
		assert.Equal(t, downgradeReason, condition.Reason)
		assert.Equal(t, metav1.ConditionFalse, condition.Status)
	})

	t.Run("Verification fails due to malformed version", func(t *testing.T) {
		previousVersion := "1.2.3.4444-555"
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

		mockClient := dtclientmock.NewClient(t)
		mockLatestAgentVersion(mockClient, "BOOM")

		updater := newOneAgentUpdater(dynakube, fake.NewClient(), mockClient)

		err := updater.UseTenantRegistry(context.Background())
		require.Error(t, err)
		assert.Equal(t, previousVersion, dynakube.Status.OneAgent.Version)

		condition := meta.FindStatusCondition(*dynakube.Conditions(), oaConditionType)
		assert.Equal(t, verificationFailedReason, condition.Reason)
		assert.Equal(t, metav1.ConditionFalse, condition.Status)
	})
	t.Run("Verification fails due to malformed ImageID", func(t *testing.T) {
		dynakube := &dynatracev1beta1.DynaKube{
			Spec: dynatracev1beta1.DynaKubeSpec{
				APIURL: testApiUrl,
				OneAgent: dynatracev1beta1.OneAgentSpec{
					CloudNativeFullStack: &dynatracev1beta1.CloudNativeFullStackSpec{},
				},
			},
			Status: dynatracev1beta1.DynaKubeStatus{
				OneAgent: dynatracev1beta1.OneAgentStatus{
					VersionStatus: status.VersionStatus{
						ImageID: "BOOM",
						Source:  status.PublicRegistryVersionSource,
					},
				},
			},
		}

		mockClient := dtclientmock.NewClient(t)
		mockLatestAgentVersion(mockClient, testVersion)

		updater := newOneAgentUpdater(dynakube, fake.NewClient(), mockClient)

		err := updater.UseTenantRegistry(context.Background())
		require.Error(t, err)

		condition := meta.FindStatusCondition(*dynakube.Conditions(), oaConditionType)
		assert.Equal(t, verificationFailedReason, condition.Reason)
		assert.Equal(t, metav1.ConditionFalse, condition.Status)
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
