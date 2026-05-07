package validation

import (
	"fmt"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/exp"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestPublicRegistryOverrideWithoutPublicRegistry(t *testing.T) {
	newDynakube := func() *dynakube.DynaKube {
		return &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test",
			},
			Spec: dynakube.DynaKubeSpec{
				APIURL: testAPIURL,
			},
		}
	}
	t.Run("publicRegistryOverride set without use-public-registry flag returns error", func(t *testing.T) {
		dk := newDynakube()
		dk.Spec.PublicRegistryOverride = "my.custom.registry.com"

		assertDenied(t, []string{fmt.Sprintf(errorPublicRegistryOverrideWithoutPublicRegistry, exp.UsePublicRegistryKey)}, dk)
	})

	t.Run("publicRegistryOverride set with use-public-registry=false returns error", func(t *testing.T) {
		dk := newDynakube()
		dk.ObjectMeta.Annotations = map[string]string{exp.UsePublicRegistryKey: "false"}
		dk.Spec.PublicRegistryOverride = "my.custom.registry.com"

		assertDenied(t, []string{fmt.Sprintf(errorPublicRegistryOverrideWithoutPublicRegistry, exp.UsePublicRegistryKey)}, dk)
	})

	t.Run("publicRegistryOverride set with use-public-registry=true returns no error", func(t *testing.T) {
		dk := newDynakube()
		dk.ObjectMeta.Annotations = map[string]string{exp.UsePublicRegistryKey: "true"}
		dk.Spec.PublicRegistryOverride = "my.custom.registry.com"

		assertAllowed(t, dk)
	})

	t.Run("publicRegistryOverride not set returns no error", func(t *testing.T) {
		dk := newDynakube()
		assertAllowed(t, dk)
	})
}
