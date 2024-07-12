package version

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta2/dynakube"
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
		dynakube := &dynakube.DynaKube{
			Spec: dynakube.DynaKubeSpec{
				OneAgent: dynakube.OneAgentSpec{
					ApplicationMonitoring: &dynakube.ApplicationMonitoringSpec{
						Version:      testImage.Tag,
						UseCSIDriver: true,
						AppInjectionSpec: dynakube.AppInjectionSpec{
							CodeModulesImage: testImage.String(),
						},
					},
				},
			},
		}
		mockClient := dtclientmock.NewClient(t)
		mockCodeModulesImageInfo(mockClient, testImage)
		updater := newCodeModulesUpdater(dynakube, mockClient)

		assert.Equal(t, "codemodules", updater.Name())
		assert.True(t, updater.IsEnabled())
		assert.Equal(t, dynakube.Spec.OneAgent.ApplicationMonitoring.CodeModulesImage, updater.CustomImage())
		assert.Equal(t, dynakube.Spec.OneAgent.ApplicationMonitoring.Version, updater.CustomVersion())
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
		dynakube := &dynakube.DynaKube{
			Spec: dynakube.DynaKubeSpec{
				OneAgent: dynakube.OneAgentSpec{
					ApplicationMonitoring: &dynakube.ApplicationMonitoringSpec{
						Version: testVersion,
					},
				},
			},
			Status: dynakube.DynaKubeStatus{
				CodeModules: oldCodeModulesStatus(),
			},
		}
		mockClient := dtclientmock.NewClient(t)
		updater := newCodeModulesUpdater(dynakube, mockClient)

		err := updater.UseTenantRegistry(ctx)
		require.NoError(t, err)
		assertDefaultCodeModulesStatus(t, testVersion, dynakube.Status.CodeModules)
		condition := meta.FindStatusCondition(*dynakube.Conditions(), cmConditionType)
		assert.Equal(t, verificationSkippedReason, condition.Reason)
		assert.Equal(t, metav1.ConditionTrue, condition.Status)
	})
	t.Run("Set according to default, unset previous status", func(t *testing.T) {
		dynakube := &dynakube.DynaKube{
			Spec: dynakube.DynaKubeSpec{
				OneAgent: dynakube.OneAgentSpec{
					ApplicationMonitoring: &dynakube.ApplicationMonitoringSpec{},
				},
			},
			Status: dynakube.DynaKubeStatus{
				CodeModules: oldCodeModulesStatus(),
			},
		}
		mockClient := dtclientmock.NewClient(t)
		mockLatestAgentVersion(mockClient, testVersion)
		updater := newCodeModulesUpdater(dynakube, mockClient)

		err := updater.UseTenantRegistry(ctx)
		require.NoError(t, err)
		assertDefaultCodeModulesStatus(t, testVersion, dynakube.Status.CodeModules)
		condition := meta.FindStatusCondition(*dynakube.Conditions(), cmConditionType)
		assert.Equal(t, verifiedReason, condition.Reason)
		assert.Equal(t, metav1.ConditionTrue, condition.Status)
	})
	t.Run("problem with Dynatrace request => visible in conditions", func(t *testing.T) {
		dynakube := &dynakube.DynaKube{
			Spec: dynakube.DynaKubeSpec{
				OneAgent: dynakube.OneAgentSpec{
					ApplicationMonitoring: &dynakube.ApplicationMonitoringSpec{},
				},
			},
			Status: dynakube.DynaKubeStatus{
				CodeModules: oldCodeModulesStatus(),
			},
		}
		updater := newCodeModulesUpdater(dynakube, createErrorDTClient(t))

		err := updater.UseTenantRegistry(ctx)
		require.Error(t, err)

		condition := meta.FindStatusCondition(*dynakube.Conditions(), cmConditionType)
		assert.Equal(t, conditions.DynatraceApiErrorReason, condition.Reason)
		assert.Equal(t, metav1.ConditionFalse, condition.Status)
	})
}

func TestCodeModulesIsEnabled(t *testing.T) {
	t.Run("cleans up condition if not enabled", func(t *testing.T) {
		dynakube := &dynakube.DynaKube{
			Status: dynakube.DynaKubeStatus{
				CodeModules: dynakube.CodeModulesStatus{
					VersionStatus: status.VersionStatus{
						Version: "prev",
					},
				},
			},
		}
		setVerifiedCondition(dynakube.Conditions(), cmConditionType)

		updater := newCodeModulesUpdater(dynakube, nil)

		isEnabled := updater.IsEnabled()
		require.False(t, isEnabled)

		condition := meta.FindStatusCondition(*dynakube.Conditions(), cmConditionType)
		assert.Nil(t, condition)

		assert.Empty(t, updater.Target())
	})
}

func TestCodeModulesPublicRegistry(t *testing.T) {
	t.Run("sets condition if enabled", func(t *testing.T) {
		dynakube := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					dynakube.AnnotationFeaturePublicRegistry: "true",
				},
			},
		}

		updater := newCodeModulesUpdater(dynakube, nil)

		isEnabled := updater.IsPublicRegistryEnabled()
		require.True(t, isEnabled)

		condition := meta.FindStatusCondition(*dynakube.Conditions(), cmConditionType)
		require.NotNil(t, condition)
		assert.Equal(t, verifiedReason, condition.Reason)
		assert.Equal(t, metav1.ConditionTrue, condition.Status)
	})
	t.Run("ignores conditions if not enabled", func(t *testing.T) {
		dynakube := &dynakube.DynaKube{}

		updater := newCodeModulesUpdater(dynakube, nil)

		isEnabled := updater.IsPublicRegistryEnabled()
		require.False(t, isEnabled)

		condition := meta.FindStatusCondition(*dynakube.Conditions(), cmConditionType)
		require.Nil(t, condition)
	})
}

func TestCodeModulesLatestImageInfo(t *testing.T) {
	t.Run("problem with Dynatrace request => visible in conditions", func(t *testing.T) {
		dynakube := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					dynakube.AnnotationFeaturePublicRegistry: "true",
				},
			},
		}

		updater := newCodeModulesUpdater(dynakube, createErrorDTClient(t))

		_, err := updater.LatestImageInfo(context.Background())
		require.Error(t, err)

		condition := meta.FindStatusCondition(*dynakube.Conditions(), cmConditionType)
		require.NotNil(t, condition)
		assert.Equal(t, conditions.DynatraceApiErrorReason, condition.Reason)
		assert.Equal(t, metav1.ConditionFalse, condition.Status)
	})
}

func oldCodeModulesStatus() dynakube.CodeModulesStatus {
	return dynakube.CodeModulesStatus{
		VersionStatus: status.VersionStatus{
			ImageID: "prev",
		},
	}
}

func assertDefaultCodeModulesStatus(t *testing.T, expectedVersion string, codeModulesStatus dynakube.CodeModulesStatus) {
	assert.Equal(t, expectedVersion, codeModulesStatus.Version)
	assert.Empty(t, codeModulesStatus.ImageID)
}
