package validation

import (
	"fmt"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/exp"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/stretchr/testify/assert"
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
		dk.Annotations = map[string]string{exp.UsePublicRegistryKey: "false"}
		dk.Spec.PublicRegistryOverride = "my.custom.registry.com"

		assertDenied(t, []string{fmt.Sprintf(errorPublicRegistryOverrideWithoutPublicRegistry, exp.UsePublicRegistryKey)}, dk)
	})

	t.Run("publicRegistryOverride set with use-public-registry=true returns no error", func(t *testing.T) {
		dk := newDynakube()
		dk.Annotations = map[string]string{exp.UsePublicRegistryKey: "true"}
		dk.Spec.PublicRegistryOverride = "my.custom.registry.com"

		assertAllowed(t, dk)
	})

	t.Run("publicRegistryOverride not set returns no error", func(t *testing.T) {
		dk := newDynakube()
		assertAllowed(t, dk)
	})

	t.Run("publicRegistryOverride set with platform token and no FF returns no error", func(t *testing.T) {
		dk := newDynakube()
		dk.Name = testName
		dk.Namespace = testNamespace
		dk.Spec.PublicRegistryOverride = "my.custom.registry.com"

		assertAllowedWithoutWarnings(t, dk, platformTokenSecret())
	})
}

func TestPublicRegistryFlagIgnoredForPlatformToken(t *testing.T) {
	newDynakube := func(annotations map[string]string) *dynakube.DynaKube {
		return &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name:        testName,
				Namespace:   testNamespace,
				Annotations: annotations,
			},
			Spec: dynakube.DynaKubeSpec{
				APIURL: testAPIURL,
			},
		}
	}

	t.Run("FF set with platform token returns warning", func(t *testing.T) {
		dk := newDynakube(map[string]string{exp.UsePublicRegistryKey: "true"})
		warnings, _ := assertAllowed(t, dk, platformTokenSecret())
		assert.Contains(t, warnings, fmt.Sprintf(warningPublicRegistryFlagIgnoredForPlatformToken, exp.UsePublicRegistryKey))
	})

	t.Run("FF set with regular token returns no warning", func(t *testing.T) {
		dk := newDynakube(map[string]string{exp.UsePublicRegistryKey: "true"})
		warnings, _ := assertAllowed(t, dk, regularTokenSecret())
		assert.Empty(t, warnings)
	})

	t.Run("FF not set with platform token returns no warning", func(t *testing.T) {
		dk := newDynakube(nil)
		warnings, _ := assertAllowed(t, dk, platformTokenSecret())
		assert.Empty(t, warnings)
	})
}
