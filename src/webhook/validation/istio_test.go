package validation

import (
	"testing"

	dynatracev1 "github.com/Dynatrace/dynatrace-operator/src/api/v1"
)

func TestNoResourcesAvailable(t *testing.T) {
	t.Run(`no resources`, func(t *testing.T) {
		assertDeniedResponse(t, []string{errorNoResources}, &dynatracev1.DynaKube{
			Spec: dynatracev1.DynaKubeSpec{
				APIURL:      testApiUrl,
				EnableIstio: true,
			},
		})
	})
}
