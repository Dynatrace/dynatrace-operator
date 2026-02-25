package version

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/exp"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/activegate"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	dtclientmock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/clients/dynatrace"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestActiveGateUpdater(t *testing.T) {
	testImage := "some.registry.com:1.2.3.4-5"

	t.Run("Getters work as expected", func(t *testing.T) {
		dk := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					exp.AGDisableUpdatesKey: "true", //nolint:staticcheck
				},
			},
			Spec: dynakube.DynaKubeSpec{
				ActiveGate: activegate.Spec{
					Capabilities: []activegate.CapabilityDisplayName{activegate.DynatraceAPICapability.DisplayName},
					CapabilityProperties: activegate.CapabilityProperties{
						Image: testImage,
					},
				},
			},
		}
		mockClient := dtclientmock.NewClient(t)

		updater := newActiveGateUpdater(dk, fake.NewClient(), mockClient)

		assert.Equal(t, "activegate", updater.Name())
		assert.True(t, updater.IsEnabled())
		assert.Equal(t, dk.Spec.ActiveGate.Image, updater.CustomImage())
		assert.Empty(t, updater.CustomVersion())
		assert.False(t, updater.IsAutoUpdateEnabled())
	})
}

func TestActiveGateUseDefault(t *testing.T) {
	t.Run("Set according to defaults, unset previous status", func(t *testing.T) {
		dk := &dynakube.DynaKube{
			Spec: dynakube.DynaKubeSpec{
				APIURL: testAPIURL,
				ActiveGate: activegate.Spec{
					CapabilityProperties: activegate.CapabilityProperties{},
				},
			},
			Status: dynakube.DynaKubeStatus{
				ActiveGate: activegate.Status{
					VersionStatus: status.VersionStatus{
						Version: "prev",
					},
				},
			},
		}
		expectedVersion := "1.2.3.4-5"
		expectedImage := dk.ActiveGate().GetDefaultImage(expectedVersion)
		mockClient := dtclientmock.NewClient(t)

		mockClient.On("GetLatestActiveGateVersion", mock.AnythingOfType("context.backgroundCtx"), mock.Anything).Return(expectedVersion, nil)

		updater := newActiveGateUpdater(dk, fake.NewClient(), mockClient)

		err := updater.UseTenantRegistry(context.Background())
		require.NoError(t, err)
		assert.Equal(t, expectedImage, dk.Status.ActiveGate.ImageID)
		assert.Equal(t, expectedVersion, dk.Status.ActiveGate.Version)
	})
}

func TestActiveGateIsEnabled(t *testing.T) {
	t.Run("cleans up if not enabled", func(t *testing.T) {
		dk := &dynakube.DynaKube{
			Status: dynakube.DynaKubeStatus{
				ActiveGate: activegate.Status{
					VersionStatus: status.VersionStatus{
						Version: "prev",
					},
				},
			},
		}

		updater := newActiveGateUpdater(dk, nil, nil)

		isEnabled := updater.IsEnabled()
		require.False(t, isEnabled)

		assert.Empty(t, updater.Target())
	})
}
