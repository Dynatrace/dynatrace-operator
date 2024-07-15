package validation

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
)

func TestNoResourcesAvailable(t *testing.T) {
	t.Run(`no resources`, func(t *testing.T) {
		assertDenied(t, []string{errorNoResources}, &dynakube.DynaKube{
			Spec: dynakube.DynaKubeSpec{
				APIURL:      testApiUrl,
				EnableIstio: true,
			},
		})
	})
}
