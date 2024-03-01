package version

import (
	"context"
	"testing"

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

func TestCodeModulesUpdater(t *testing.T) {
	ctx := context.Background()
	testImage := dtclient.LatestImageInfo{
		Source: "some.registry.com",
		Tag:    "1.2.3.4-5",
	}

	t.Run("Getters work as expected", func(t *testing.T) {
		dynakube := &dynatracev1beta1.DynaKube{
			Spec: dynatracev1beta1.DynaKubeSpec{
				OneAgent: dynatracev1beta1.OneAgentSpec{
					ApplicationMonitoring: &dynatracev1beta1.ApplicationMonitoringSpec{
						Version:      testImage.Tag,
						UseCSIDriver: address.Of(true),
						AppInjectionSpec: dynatracev1beta1.AppInjectionSpec{
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
		dynakube := &dynatracev1beta1.DynaKube{
			Spec: dynatracev1beta1.DynaKubeSpec{
				OneAgent: dynatracev1beta1.OneAgentSpec{
					ApplicationMonitoring: &dynatracev1beta1.ApplicationMonitoringSpec{
						Version: testVersion,
					},
				},
			},
			Status: dynatracev1beta1.DynaKubeStatus{
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
		dynakube := &dynatracev1beta1.DynaKube{
			Spec: dynatracev1beta1.DynaKubeSpec{
				OneAgent: dynatracev1beta1.OneAgentSpec{
					ApplicationMonitoring: &dynatracev1beta1.ApplicationMonitoringSpec{},
				},
			},
			Status: dynatracev1beta1.DynaKubeStatus{
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
		dynakube := &dynatracev1beta1.DynaKube{
			Spec: dynatracev1beta1.DynaKubeSpec{
				OneAgent: dynatracev1beta1.OneAgentSpec{
					ApplicationMonitoring: &dynatracev1beta1.ApplicationMonitoringSpec{},
				},
			},
			Status: dynatracev1beta1.DynaKubeStatus{
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
		dynakube := &dynatracev1beta1.DynaKube{}
		setVerifiedCondition(dynakube.Conditions(), cmConditionType)

		updater := newCodeModulesUpdater(dynakube, nil)

		isEnabled := updater.IsEnabled()
		require.False(t, isEnabled)

		condition := meta.FindStatusCondition(*dynakube.Conditions(), cmConditionType)
		assert.Nil(t, condition)
	})
}

func TestCodeModulesPublicRegistry(t *testing.T) {
	t.Run("sets condition if enabled", func(t *testing.T) {
		dynakube := &dynatracev1beta1.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					dynatracev1beta1.AnnotationFeaturePublicRegistry: "true",
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
		dynakube := &dynatracev1beta1.DynaKube{}

		updater := newCodeModulesUpdater(dynakube, nil)

		isEnabled := updater.IsPublicRegistryEnabled()
		require.False(t, isEnabled)

		condition := meta.FindStatusCondition(*dynakube.Conditions(), cmConditionType)
		require.Nil(t, condition)
	})
}

func TestCodeModulesLatestImageInfo(t *testing.T) {
	t.Run("problem with Dynatrace request => visible in conditions", func(t *testing.T) {
		dynakube := &dynatracev1beta1.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					dynatracev1beta1.AnnotationFeaturePublicRegistry: "true",
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

func oldCodeModulesStatus() dynatracev1beta1.CodeModulesStatus {
	return dynatracev1beta1.CodeModulesStatus{
		VersionStatus: status.VersionStatus{
			ImageID: "prev",
		},
	}
}

func assertDefaultCodeModulesStatus(t *testing.T, expectedVersion string, codeModulesStatus dynatracev1beta1.CodeModulesStatus) {
	assert.Equal(t, expectedVersion, codeModulesStatus.Version)
	assert.Empty(t, codeModulesStatus.ImageID)
}
