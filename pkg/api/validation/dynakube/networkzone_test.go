// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package validation

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
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
			Name: testName, Namespace: testNamespace,
			Spec: dynakube.DynaKubeSpec{APIURL: testAPIURL},
		}

		assertSanitizeArg(t, dk, func(dk *dynakube.DynaKube, value string) {
			dk.Spec.NetworkZone = value
		}, errorInvalidNetworkZone)
	})

	t.Run("valid fallback modes are allowed", func(t *testing.T) {
		for _, mode := range []dynakube.NZFallbackMode{"", dynakube.NZFallbackModeDefault, dynakube.NZFallbackModeNone, dynakube.NZFallbackModeAny} {
			assertAllowed(t, &dynakube.DynaKube{
				Spec: dynakube.DynaKubeSpec{APIURL: testAPIURL, NetworkZone: "network-zone", NetworkZoneFallbackMode: mode},
			})
		}
	})

	t.Run("fallback mode with default network zone is denied", func(t *testing.T) {
		assertDenied(t, []string{errorFallbackModeWithDefaultNetworkZone}, &dynakube.DynaKube{
			Spec: dynakube.DynaKubeSpec{APIURL: testAPIURL, NetworkZone: "default", NetworkZoneFallbackMode: dynakube.NZFallbackModeNone},
		})
	})

	t.Run("default network zone without fallback mode is allowed", func(t *testing.T) {
		assertAllowed(t, &dynakube.DynaKube{
			Spec: dynakube.DynaKubeSpec{APIURL: testAPIURL, NetworkZone: "default"},
		})
	})
}
