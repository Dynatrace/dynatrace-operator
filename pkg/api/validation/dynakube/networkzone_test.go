package validation

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestInvalidNetworkZone(t *testing.T) {
	t.Run("empty network zone is allowed", func(t *testing.T) {
		assertAllowed(t, &dynakube.DynaKube{
			Spec: dynakube.DynaKubeSpec{APIURL: testAPIURL},
		})
	})

	t.Run("valid network zone is allowed", func(t *testing.T) {
		assertAllowed(t, &dynakube.DynaKube{
			Spec: dynakube.DynaKubeSpec{APIURL: testAPIURL, NetworkZone: "network-zone"},
		})
	})

	t.Run("network zone with invalid characters is denied", func(t *testing.T) {
		dk := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{Name: testName, Namespace: testNamespace},
			Spec:       dynakube.DynaKubeSpec{APIURL: testAPIURL},
		}

		assertSanitizeArg(t, dk, func(dk *dynakube.DynaKube, value string) {
			dk.Spec.NetworkZone = value
		}, errorInvalidNetworkZone)
	})
}
