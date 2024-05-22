package dynakube

import (
	"testing"

	dynatracev1beta2 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta2/dynakube"
)

func TestNoResourcesAvailable(t *testing.T) {
	t.Run(`no resources`, func(t *testing.T) {
		assertDeniedResponse(t, []string{errorNoResources}, &dynatracev1beta2.DynaKube{
			Spec: dynatracev1beta2.DynaKubeSpec{
				APIURL:      testApiUrl,
				EnableIstio: true,
			},
		})
	})
}
