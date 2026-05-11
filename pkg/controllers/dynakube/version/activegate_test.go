package version

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/exp"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/activegate"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	imagesclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/image"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8sconditions"
	imageclientmock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/clients/dynatrace/images"
	versionclientmock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/clients/dynatrace/version"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/api/meta"
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
		mockImageClient := imageclientmock.NewClient(t)
		mockVersionClient := versionclientmock.NewClient(t)

		updater := newActiveGateUpdater(dk, fake.NewClient(), mockImageClient, mockVersionClient)

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
		mockImageClient := imageclientmock.NewClient(t)

		expectedVersion := "1.2.3.4-5"
		expectedImage := dk.ActiveGate().GetDefaultImage(expectedVersion)
		mockVersionClient := versionclientmock.NewClient(t)
		mockVersionClient.On("GetLatestActiveGateVersion", mock.AnythingOfType("context.backgroundCtx"), mock.Anything).Return(expectedVersion, nil)

		updater := newActiveGateUpdater(dk, fake.NewClient(), mockImageClient, mockVersionClient)

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

		updater := newActiveGateUpdater(dk, nil, nil, nil)

		isEnabled := updater.IsEnabled()
		require.False(t, isEnabled)

		assert.Empty(t, updater.Target())
	})
}

func TestActiveGateLatestImageInfo(t *testing.T) {
	const testRegistry = "my.custom.registry.com"
	const testTag = "1.2.3.4-5"
	const testImageURI = testRegistry + "/dynatrace/activegate:" + testTag

	newDK := func(registry string) *dynakube.DynaKube {
		return &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					exp.UsePublicRegistryKey: "true",
				},
			},
			Spec: dynakube.DynaKubeSpec{
				ActiveGate: activegate.Spec{
					Capabilities: []activegate.CapabilityDisplayName{activegate.DynatraceAPICapability.DisplayName},
				},
				PublicRegistryOverride: registry,
			},
		}
	}

	t.Run("happy path: image info returned and verified condition set", func(t *testing.T) {
		dk := newDK("")
		mockImageClient := imageclientmock.NewClient(t)
		mockImageClient.EXPECT().ComponentLatestImageInfo(t.Context(), imagesclient.ActiveGate, "").Return(
			&imagesclient.ImageInfo{URI: testImageURI, Tag: testTag}, nil,
		).Once()

		updater := newActiveGateUpdater(dk, fake.NewClient(), mockImageClient, nil)
		imageInfo, err := updater.LatestImageInfo(t.Context())

		require.NoError(t, err)
		require.NotNil(t, imageInfo)
		assert.Equal(t, testTag, imageInfo.Tag)
		assert.Equal(t, testImageURI, imageInfo.URI)

		condition := meta.FindStatusCondition(*dk.Conditions(), activeGateVersionConditionType)
		require.NotNil(t, condition)
		assert.Equal(t, metav1.ConditionTrue, condition.Status)
		assert.Equal(t, verifiedReason, condition.Reason)
	})

	t.Run("registry override forwarded to images client", func(t *testing.T) {
		dk := newDK(testRegistry)
		mockImageClient := imageclientmock.NewClient(t)
		mockImageClient.EXPECT().ComponentLatestImageInfo(t.Context(), imagesclient.ActiveGate, testRegistry).Return(
			&imagesclient.ImageInfo{URI: testImageURI, Tag: testTag, Registry: testRegistry}, nil,
		).Once()

		updater := newActiveGateUpdater(dk, fake.NewClient(), mockImageClient, nil)
		imageInfo, err := updater.LatestImageInfo(t.Context())

		require.NoError(t, err)
		assert.Equal(t, testTag, imageInfo.Tag)
	})

	t.Run("API error: error returned and DynatraceAPIError condition set", func(t *testing.T) {
		dk := newDK("")
		mockImageClient := imageclientmock.NewClient(t)
		mockImageClient.EXPECT().ComponentLatestImageInfo(t.Context(), imagesclient.ActiveGate, "").Return(
			nil, errors.New("BOOM"),
		).Once()

		updater := newActiveGateUpdater(dk, fake.NewClient(), mockImageClient, nil)
		imageInfo, err := updater.LatestImageInfo(t.Context())

		require.Error(t, err)
		assert.Nil(t, imageInfo)

		condition := meta.FindStatusCondition(*dk.Conditions(), activeGateVersionConditionType)
		require.NotNil(t, condition)
		assert.Equal(t, metav1.ConditionFalse, condition.Status)
		assert.Equal(t, k8sconditions.DynatraceAPIErrorReason, condition.Reason)
	})
}
