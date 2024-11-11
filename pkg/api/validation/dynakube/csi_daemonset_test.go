package validation

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
)

func TestDisabledCSIForReadonlyCSIVolume(t *testing.T) {
	objectMeta := defaultDynakubeObjectMeta.DeepCopy()
	objectMeta.Annotations = map[string]string{
		dynakube.AnnotationFeatureReadOnlyCsiVolume: "true",
	}

	t.Run("valid cloud-native dynakube specs", func(t *testing.T) {
		assertAllowedWithoutWarnings(t, &dynakube.DynaKube{
			ObjectMeta: *objectMeta,
			Spec: dynakube.DynaKubeSpec{
				APIURL: testApiUrl,
				OneAgent: dynakube.OneAgentSpec{
					CloudNativeFullStack: &dynakube.CloudNativeFullStackSpec{},
				},
			},
		})
	})

	t.Run("invalid dynakube specs, as csi is not supported for feature", func(t *testing.T) {
		assertDenied(t,
			[]string{errorCSIEnabledRequired},
			&dynakube.DynaKube{
				ObjectMeta: *objectMeta,
				Spec: dynakube.DynaKubeSpec{
					APIURL: testApiUrl,
					OneAgent: dynakube.OneAgentSpec{
						ClassicFullStack: &dynakube.HostInjectSpec{},
					},
				},
			})
	})
}
