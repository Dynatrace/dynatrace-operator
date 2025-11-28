package version

import (
	"context"
	"errors"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/exp"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/oneagent"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/conditions"
	dtclientmock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/clients/dynatrace"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestCodeModulesUpdater(t *testing.T) {
	ctx := context.Background()
	testImage := dtclient.LatestImageInfo{
		Source: "some.registry.com",
		Tag:    "1.2.3.4-5",
	}

	t.Run("Getters work as expected", func(t *testing.T) {
		dk := &dynakube.DynaKube{
			Spec: dynakube.DynaKubeSpec{
				OneAgent: oneagent.Spec{
					ApplicationMonitoring: &oneagent.ApplicationMonitoringSpec{
						Version: testImage.Tag,
						AppInjectionSpec: oneagent.AppInjectionSpec{
							CodeModulesImage: testImage.String(),
						},
					},
				},
			},
		}
		mockClient := dtclientmock.NewClient(t)
		mockCodeModulesImageInfo(mockClient, testImage)
		updater := newCodeModulesUpdater(dk, mockClient)

		assert.Equal(t, "codemodules", updater.Name())
		assert.True(t, updater.IsEnabled())
		assert.Equal(t, dk.Spec.OneAgent.ApplicationMonitoring.CodeModulesImage, updater.CustomImage())
		assert.Equal(t, dk.Spec.OneAgent.ApplicationMonitoring.Version, updater.CustomVersion())
		assert.True(t, updater.IsAutoUpdateEnabled())
		imageInfo, err := updater.LatestImageInfo(ctx)
		require.NoError(t, err)
		assert.Equal(t, testImage, *imageInfo)
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
		mockClient := dtclientmock.NewClient(t)
		updater := newCodeModulesUpdater(dk, mockClient)

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
		mockClient := dtclientmock.NewClient(t)
		mockLatestAgentVersion(mockClient, testVersion, 1)
		updater := newCodeModulesUpdater(dk, mockClient)

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
		mockClient := dtclientmock.NewClient(t)
		mockClient.EXPECT().
			GetLatestAgentVersion(mockCtx, dtclient.OsUnix, dtclient.InstallerTypePaaS).
			Return("", errors.New("BOOM")).Once()
		updater := newCodeModulesUpdater(dk, mockClient)

		err := updater.UseTenantRegistry(ctx)
		require.Error(t, err)

		condition := meta.FindStatusCondition(*dk.Conditions(), cmConditionType)
		assert.Equal(t, conditions.DynatraceAPIErrorReason, condition.Reason)
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

		updater := newCodeModulesUpdater(dk, nil)

		isEnabled := updater.IsEnabled()
		require.False(t, isEnabled)

		condition := meta.FindStatusCondition(*dk.Conditions(), cmConditionType)
		assert.Nil(t, condition)

		assert.Empty(t, updater.Target())
	})
}

func TestCodeModulesPublicRegistry(t *testing.T) {
	t.Run("sets condition if enabled", func(t *testing.T) {
		dk := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					exp.PublicRegistryKey: "true",
				},
			},
		}

		updater := newCodeModulesUpdater(dk, nil)

		isEnabled := updater.IsPublicRegistryEnabled()
		require.True(t, isEnabled)

		condition := meta.FindStatusCondition(*dk.Conditions(), cmConditionType)
		require.NotNil(t, condition)
		assert.Equal(t, verifiedReason, condition.Reason)
		assert.Equal(t, metav1.ConditionTrue, condition.Status)
	})
	t.Run("ignores conditions if not enabled", func(t *testing.T) {
		dk := &dynakube.DynaKube{}

		updater := newCodeModulesUpdater(dk, nil)

		isEnabled := updater.IsPublicRegistryEnabled()
		require.False(t, isEnabled)

		condition := meta.FindStatusCondition(*dk.Conditions(), cmConditionType)
		require.Nil(t, condition)
	})
}

func TestCodeModulesLatestImageInfo(t *testing.T) {
	t.Run("problem with Dynatrace request => visible in conditions", func(t *testing.T) {
		dk := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					exp.PublicRegistryKey: "true",
				},
			},
		}

		mockClient := dtclientmock.NewClient(t)
		mockClient.EXPECT().GetLatestCodeModulesImage(mockCtx).Return(nil, errors.New("BOOM")).Once()
		updater := newCodeModulesUpdater(dk, mockClient)

		_, err := updater.LatestImageInfo(context.Background())
		require.Error(t, err)

		condition := meta.FindStatusCondition(*dk.Conditions(), cmConditionType)
		require.NotNil(t, condition)
		assert.Equal(t, conditions.DynatraceAPIErrorReason, condition.Reason)
		assert.Equal(t, metav1.ConditionFalse, condition.Status)
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
