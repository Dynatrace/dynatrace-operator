package validation

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
)

func TestNoIstioInstalled(t *testing.T) {
	t.Run("no resources", func(t *testing.T) {
		assertDenied(t, []string{errorNoIstioInstalled}, &dynakube.DynaKube{
			Spec: dynakube.DynaKubeSpec{
				APIURL:      testAPIURL,
				EnableIstio: true,
			},
		})
	})
}
