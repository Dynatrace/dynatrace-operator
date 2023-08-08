package version

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/src/api/status"
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1/dynakube"
	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
	"github.com/Dynatrace/dynatrace-operator/src/registry"
	"github.com/Dynatrace/dynatrace-operator/src/scheme/fake"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestActiveGateUpdater(t *testing.T) {
	testImage := dtclient.LatestImageInfo{
		Source: "some.registry.com",
		Tag:    "1.2.3",
	}
	t.Run("Getters work as expected", func(t *testing.T) {
		dynakube := &dynatracev1beta1.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					dynatracev1beta1.AnnotationFeatureDisableActiveGateUpdates: "true",
				},
			},
			Spec: dynatracev1beta1.DynaKubeSpec{
				ActiveGate: dynatracev1beta1.ActiveGateSpec{
					Capabilities: []dynatracev1beta1.CapabilityDisplayName{dynatracev1beta1.DynatraceApiCapability.DisplayName},
					CapabilityProperties: dynatracev1beta1.CapabilityProperties{
						Image: testImage.String(),
					},
				},
			},
		}
		mockClient := &dtclient.MockDynatraceClient{}
		mockActiveGateImageInfo(mockClient, testImage)
		registry := newEmptyFakeRegistry()

		updater := newActiveGateUpdater(dynakube, fake.NewClient(), mockClient, registry.ImageVersionExt)

		assert.Equal(t, "activegate", updater.Name())
		assert.True(t, updater.IsEnabled())
		assert.Equal(t, dynakube.Spec.ActiveGate.Image, updater.CustomImage())
		assert.Equal(t, "", updater.CustomVersion())
		assert.False(t, updater.IsAutoUpdateEnabled())
		imageInfo, err := updater.LatestImageInfo()
		require.NoError(t, err)
		assert.Equal(t, testImage, *imageInfo)
	})
}

func TestActiveGateUseDefault(t *testing.T) {
	t.Run("Set according to defaults, unset previous status", func(t *testing.T) {
		dynakube := &dynatracev1beta1.DynaKube{
			Spec: dynatracev1beta1.DynaKubeSpec{
				APIURL: testApiUrl,
				ActiveGate: dynatracev1beta1.ActiveGateSpec{
					CapabilityProperties: dynatracev1beta1.CapabilityProperties{},
				},
			},
			Status: dynatracev1beta1.DynaKubeStatus{
				ActiveGate: dynatracev1beta1.ActiveGateStatus{
					VersionStatus: status.VersionStatus{
						Version: "prev",
					},
				},
			},
		}
		expectedImage := dynakube.DefaultActiveGateImage()
		expectedVersion := "1.2.3"
		mockClient := &dtclient.MockDynatraceClient{}
		registry := newFakeRegistry(map[string]registry.ImageVersion{
			expectedImage: {
				Version: expectedVersion,
			},
		})
		updater := newActiveGateUpdater(dynakube, fake.NewClient(), mockClient, registry.ImageVersionExt)

		err := updater.UseTenantRegistry(context.TODO(), "")
		require.NoError(t, err)
		assert.Equal(t, expectedImage, dynakube.Status.ActiveGate.ImageID)
		assert.Equal(t, expectedVersion, dynakube.Status.ActiveGate.Version)
	})
}
