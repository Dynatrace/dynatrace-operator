package version

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/exp"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/oneagent"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	imagesclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/image"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/installer"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8sconditions"
	imageclientmock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/clients/dynatrace/images"
	versionclientmock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/clients/dynatrace/version"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestCodeModulesUpdater(t *testing.T) {
	testVersion := "1.2.3.4-5"
	testImage := "some.registry.com:" + testVersion

	t.Run("Getters work as expected", func(t *testing.T) {
		dk := &dynakube.DynaKube{
			Spec: dynakube.DynaKubeSpec{
				OneAgent: oneagent.Spec{
					ApplicationMonitoring: &oneagent.ApplicationMonitoringSpec{
						Version: testVersion,
						AppInjectionSpec: oneagent.AppInjectionSpec{
							CodeModulesImage: testImage,
						},
					},
				},
			},
		}
		mockVersionClient := versionclientmock.NewClient(t)
		mockImageClient := imageclientmock.NewClient(t)

		updater := newCodeModulesUpdater(dk, mockImageClient, mockVersionClient)

		assert.Equal(t, "codemodules", updater.Name())
		assert.True(t, updater.IsEnabled())
		assert.Equal(t, dk.Spec.OneAgent.ApplicationMonitoring.CodeModulesImage, updater.CustomImage())
		assert.Equal(t, dk.Spec.OneAgent.ApplicationMonitoring.Version, updater.CustomVersion()) //nolint:staticcheck
		assert.True(t, updater.IsAutoUpdateEnabled())
	})
}

func TestCodeModulesUseDefault(t *testing.T) {
	ctx := context.Background()
	testVersion := "1.2.3.4-5"

	t.Run("Set according to version field, unset previous status", func(t *testing.T) {
		dk := &dynakube.DynaKube{
			Spec: dynakube.DynaKubeSpec{
				OneAgent: oneagent.Spec{
					ApplicationMonitoring: &oneagent.ApplicationMonitoringSpec{
						Version: testVersion,
					},
				},
			},
			Status: dynakube.DynaKubeStatus{
				CodeModules: oldCodeModulesStatus(),
			},
		}
		mockImageClient := imageclientmock.NewClient(t)
		mockVersionClient := versionclientmock.NewClient(t)

		updater := newCodeModulesUpdater(dk, mockImageClient, mockVersionClient)

		err := updater.UseTenantRegistry(ctx)
		require.NoError(t, err)
		assertDefaultCodeModulesStatus(t, testVersion, dk.Status.CodeModules)
		condition := meta.FindStatusCondition(*dk.Conditions(), cmConditionType)
		assert.Equal(t, verificationSkippedReason, condition.Reason)
		assert.Equal(t, metav1.ConditionTrue, condition.Status)
	})
	t.Run("Set according to default, unset previous status", func(t *testing.T) {
		dk := &dynakube.DynaKube{
			Spec: dynakube.DynaKubeSpec{
				OneAgent: oneagent.Spec{
					ApplicationMonitoring: &oneagent.ApplicationMonitoringSpec{},
				},
			},
			Status: dynakube.DynaKubeStatus{
				CodeModules: oldCodeModulesStatus(),
			},
		}
		mockImageClient := imageclientmock.NewClient(t)
		mockVersionClient := versionclientmock.NewClient(t)
		mockLatestAgentVersion(mockVersionClient, testVersion, 1)

		updater := newCodeModulesUpdater(dk, mockImageClient, mockVersionClient)

		err := updater.UseTenantRegistry(ctx)
		require.NoError(t, err)
		assertDefaultCodeModulesStatus(t, testVersion, dk.Status.CodeModules)
		condition := meta.FindStatusCondition(*dk.Conditions(), cmConditionType)
		assert.Equal(t, verifiedReason, condition.Reason)
		assert.Equal(t, metav1.ConditionTrue, condition.Status)
	})
	t.Run("problem with Dynatrace request => visible in conditions", func(t *testing.T) {
		dk := &dynakube.DynaKube{
			Spec: dynakube.DynaKubeSpec{
				OneAgent: oneagent.Spec{
					ApplicationMonitoring: &oneagent.ApplicationMonitoringSpec{},
				},
			},
			Status: dynakube.DynaKubeStatus{
				CodeModules: oldCodeModulesStatus(),
			},
		}
		mockImageClient := imageclientmock.NewClient(t)
		mockVersionClient := versionclientmock.NewClient(t)
		mockVersionClient.EXPECT().GetLatestAgentVersion(anyCtx, installer.OSUnix, installer.TypePaaS).Return("", errors.New("BOOM")).Once()

		updater := newCodeModulesUpdater(dk, mockImageClient, mockVersionClient)

		err := updater.UseTenantRegistry(ctx)
		require.Error(t, err)

		condition := meta.FindStatusCondition(*dk.Conditions(), cmConditionType)
		assert.Equal(t, k8sconditions.DynatraceAPIErrorReason, condition.Reason)
		assert.Equal(t, metav1.ConditionFalse, condition.Status)
	})
}

func TestCodeModulesIsEnabled(t *testing.T) {
	t.Run("cleans up condition if not enabled", func(t *testing.T) {
		dk := &dynakube.DynaKube{
			Status: dynakube.DynaKubeStatus{
				CodeModules: oneagent.CodeModulesStatus{
					VersionStatus: status.VersionStatus{
						Version: "prev",
					},
				},
			},
		}
		setVerifiedCondition(dk.Conditions(), cmConditionType)

		updater := newCodeModulesUpdater(dk, nil, nil)

		isEnabled := updater.IsEnabled()
		require.False(t, isEnabled)

		condition := meta.FindStatusCondition(*dk.Conditions(), cmConditionType)
		assert.Nil(t, condition)

		assert.Empty(t, updater.Target())
	})
}

func oldCodeModulesStatus() oneagent.CodeModulesStatus {
	return oneagent.CodeModulesStatus{
		VersionStatus: status.VersionStatus{
			ImageID: "prev",
		},
	}
}

func assertDefaultCodeModulesStatus(t *testing.T, expectedVersion string, codeModulesStatus oneagent.CodeModulesStatus) {
	assert.Equal(t, expectedVersion, codeModulesStatus.Version)
	assert.Empty(t, codeModulesStatus.ImageID)
}

func TestCodeModulesLatestImageInfo(t *testing.T) {
	const testRegistry = "my.custom.registry.com"
	const testTag = "1.2.3.4-5"
	const testImageURI = testRegistry + "/dynatrace/dynatrace-codemodules:" + testTag

	newDK := func(registry string) *dynakube.DynaKube {
		return &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					exp.UsePublicRegistryKey: "true",
				},
			},
			Spec: dynakube.DynaKubeSpec{
				OneAgent: oneagent.Spec{
					ApplicationMonitoring: &oneagent.ApplicationMonitoringSpec{},
				},
				PublicRegistryOverride: registry,
			},
		}
	}

	t.Run("happy path: image info returned and verified condition set", func(t *testing.T) {
		dk := newDK("")
		mockImageClient := imageclientmock.NewClient(t)
		mockImageClient.EXPECT().ComponentLatestImageInfo(t.Context(), imagesclient.CodeModules, "").Return(
			&imagesclient.ImageInfo{URI: testImageURI, Tag: testTag}, nil,
		).Once()

		updater := newCodeModulesUpdater(dk, mockImageClient, nil)
		imageInfo, err := updater.LatestImageInfo(t.Context())

		require.NoError(t, err)
		require.NotNil(t, imageInfo)
		assert.Equal(t, testTag, imageInfo.Tag)
		assert.Equal(t, testImageURI, imageInfo.URI)

		condition := meta.FindStatusCondition(*dk.Conditions(), cmConditionType)
		require.NotNil(t, condition)
		assert.Equal(t, metav1.ConditionTrue, condition.Status)
		assert.Equal(t, verifiedReason, condition.Reason)
	})

	t.Run("registry override forwarded to images client", func(t *testing.T) {
		dk := newDK(testRegistry)
		mockImageClient := imageclientmock.NewClient(t)
		mockImageClient.EXPECT().ComponentLatestImageInfo(t.Context(), imagesclient.CodeModules, testRegistry).Return(
			&imagesclient.ImageInfo{URI: testImageURI, Tag: testTag, Registry: testRegistry}, nil,
		).Once()

		updater := newCodeModulesUpdater(dk, mockImageClient, nil)
		imageInfo, err := updater.LatestImageInfo(t.Context())

		require.NoError(t, err)
		assert.Equal(t, testTag, imageInfo.Tag)
	})

	t.Run("API error: error returned and DynatraceAPIError condition set", func(t *testing.T) {
		dk := newDK("")
		mockImageClient := imageclientmock.NewClient(t)
		mockImageClient.EXPECT().ComponentLatestImageInfo(t.Context(), imagesclient.CodeModules, "").Return(
			nil, errors.New("BOOM"),
		).Once()

		updater := newCodeModulesUpdater(dk, mockImageClient, nil)
		imageInfo, err := updater.LatestImageInfo(t.Context())

		require.Error(t, err)
		assert.Nil(t, imageInfo)

		condition := meta.FindStatusCondition(*dk.Conditions(), cmConditionType)
		require.NotNil(t, condition)
		assert.Equal(t, metav1.ConditionFalse, condition.Status)
		assert.Equal(t, k8sconditions.DynatraceAPIErrorReason, condition.Reason)
	})
}
