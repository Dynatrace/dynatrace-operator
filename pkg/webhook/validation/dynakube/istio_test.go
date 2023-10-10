package dynakube

import (
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
)

func TestNoResourcesAvailable(t *testing.T) {
	t.Run(`no resources`, func(t *testing.T) {
		assertDeniedResponse(t, []string{errorNoResources}, &dynatracev1beta1.DynaKube{
			Spec: dynatracev1beta1.DynaKubeSpec{
				APIURL:      testApiUrl,
				EnableIstio: true,
			},
		})
	})
}
