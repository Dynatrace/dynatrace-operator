package validation

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube/activegate"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube/kspm"
)

func TestMissingKSPMDependency(t *testing.T) {
	t.Run("both kspm and kubemon enabled", func(t *testing.T) {
		assertAllowed(t,
			&dynakube.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynakube.DynaKubeSpec{
					APIURL: testApiUrl,
					Kspm: kspm.Spec{
						Enabled: true,
					},
					ActiveGate: activegate.Spec{
						Capabilities: []activegate.CapabilityDisplayName{
							activegate.KubeMonCapability.DisplayName,
						},
					},
				},
			})
	})

	t.Run("missing kubemon but kspm enabled", func(t *testing.T) {
		assertDenied(t,
			[]string{errorKSPMMissingKubemon},
			&dynakube.DynaKube{
				ObjectMeta: defaultDynakubeObjectMeta,
				Spec: dynakube.DynaKubeSpec{
					APIURL: testApiUrl,
					Kspm: kspm.Spec{
						Enabled: true,
					},
					ActiveGate: activegate.Spec{},
				},
			})
	})
}
