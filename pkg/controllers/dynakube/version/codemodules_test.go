package version

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/address"
	dtclientmock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/clients/dynatrace"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
