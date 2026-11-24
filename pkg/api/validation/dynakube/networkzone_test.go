// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package validation

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
)

func TestFallbackModeWithDefaultNetworkZone(t *testing.T) {
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
