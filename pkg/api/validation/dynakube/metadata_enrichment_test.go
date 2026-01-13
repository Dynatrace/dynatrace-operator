package validation

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/metadataenrichment"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/oneagent"
	"k8s.io/utils/ptr"
)

func TestDisabledMetadataEnrichmentForInjectionModes(t *testing.T) {
	t.Run("warns if disabled and app monitoring", func(t *testing.T) {
		dk := &dynakube.DynaKube{
			ObjectMeta: defaultDynakubeObjectMeta,
			Spec: dynakube.DynaKubeSpec{
				APIURL: testAPIURL,
				OneAgent: oneagent.Spec{
					ApplicationMonitoring: &oneagent.ApplicationMonitoringSpec{},
				},
				MetadataEnrichment: metadataenrichment.Spec{
					Enabled: ptr.To(false),
				},
			},
		}
		assertAllowedWithWarnings(t, 1, dk)
	})

	t.Run("warns if disabled and cloud native", func(t *testing.T) {
		dk := &dynakube.DynaKube{
			ObjectMeta: defaultDynakubeObjectMeta,
			Spec: dynakube.DynaKubeSpec{
				APIURL: testAPIURL,
				OneAgent: oneagent.Spec{
					CloudNativeFullStack: &oneagent.CloudNativeFullStackSpec{},
				},
				MetadataEnrichment: metadataenrichment.Spec{
					Enabled: ptr.To(false),
				},
			},
		}
		assertAllowedWithWarnings(t, 1, dk)
	})

	t.Run("no warning if enabled", func(t *testing.T) {
		dk := &dynakube.DynaKube{
			ObjectMeta: defaultDynakubeObjectMeta,
			Spec: dynakube.DynaKubeSpec{
				APIURL: testAPIURL,
				OneAgent: oneagent.Spec{
					ApplicationMonitoring: &oneagent.ApplicationMonitoringSpec{},
				},
				MetadataEnrichment: metadataenrichment.Spec{
					Enabled: ptr.To(true),
				},
			},
		}
		assertAllowedWithoutWarnings(t, dk)
	})

	t.Run("no warning if unconfigured", func(t *testing.T) {
		dk := &dynakube.DynaKube{
			ObjectMeta: defaultDynakubeObjectMeta,
			Spec: dynakube.DynaKubeSpec{
				APIURL: testAPIURL,
				OneAgent: oneagent.Spec{
					ApplicationMonitoring: &oneagent.ApplicationMonitoringSpec{},
				},
			},
		}
		assertAllowedWithoutWarnings(t, dk)
	})

	t.Run("no warning if disabled and host monitoring", func(t *testing.T) {
		dk := &dynakube.DynaKube{
			ObjectMeta: defaultDynakubeObjectMeta,
			Spec: dynakube.DynaKubeSpec{
				APIURL: testAPIURL,
				OneAgent: oneagent.Spec{
					HostMonitoring: &oneagent.HostInjectSpec{},
				},
				MetadataEnrichment: metadataenrichment.Spec{
					Enabled: ptr.To(false),
				},
			},
		}
		assertAllowedWithoutWarnings(t, dk)
	})
}
