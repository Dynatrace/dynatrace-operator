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

	t.Run("network zone with newline is denied", func(t *testing.T) {
		assertDenied(t, []string{errorInvalidNetworkZone}, &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{Name: testName, Namespace: testNamespace},
			Spec:       dynakube.DynaKubeSpec{APIURL: testAPIURL, NetworkZone: "network\nzone"},
		})
	})

	t.Run("network zone with tab is denied", func(t *testing.T) {
		assertDenied(t, []string{errorInvalidNetworkZone}, &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{Name: testName, Namespace: testNamespace},
			Spec:       dynakube.DynaKubeSpec{APIURL: testAPIURL, NetworkZone: "network\tzone"},
		})
	})

	t.Run("network zone with carriage return is denied", func(t *testing.T) {
		assertDenied(t, []string{errorInvalidNetworkZone}, &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{Name: testName, Namespace: testNamespace},
			Spec:       dynakube.DynaKubeSpec{APIURL: testAPIURL, NetworkZone: "network\rzone"},
		})
	})

	t.Run("network zone with null byte is denied", func(t *testing.T) {
		assertDenied(t, []string{errorInvalidNetworkZone}, &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{Name: testName, Namespace: testNamespace},
			Spec:       dynakube.DynaKubeSpec{APIURL: testAPIURL, NetworkZone: "network\x00zone"},
		})
	})
}
