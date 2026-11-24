// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package dynakube

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/conversion"
	dynakubelatest "github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNetworkZoneFallbackModeConversion(t *testing.T) {
	t.Run("is preserved in an annotation when converting to v1beta5", func(t *testing.T) {
		latest := &dynakubelatest.DynaKube{
			Spec: dynakubelatest.DynaKubeSpec{
				NetworkZone:             "zone",
				NetworkZoneFallbackMode: dynakubelatest.NZFallbackModeNone,
			},
		}

		old := &DynaKube{}
		require.NoError(t, old.ConvertFrom(latest))

		assert.Equal(t, "zone", old.Spec.NetworkZone)
		assert.JSONEq(t, `"none"`, old.Annotations[conversion.NetworkZoneFallbackModeKey])
	})

	t.Run("round trip keeps the field and doesn't leak the annotation", func(t *testing.T) {
		latest := &dynakubelatest.DynaKube{
			Annotations: map[string]string{"other": "annotation"},
			Spec: dynakubelatest.DynaKubeSpec{
				NetworkZone:             "zone",
				NetworkZoneFallbackMode: dynakubelatest.NZFallbackModeAny,
			},
		}

		old := &DynaKube{}
		require.NoError(t, old.ConvertFrom(latest))

		converted := &dynakubelatest.DynaKube{}
		require.NoError(t, old.ConvertTo(converted))

		assert.Equal(t, dynakubelatest.NZFallbackModeAny, converted.Spec.NetworkZoneFallbackMode)
		assert.Equal(t, map[string]string{"other": "annotation"}, converted.Annotations)
	})

	t.Run("empty field doesn't create an annotation", func(t *testing.T) {
		old := &DynaKube{}
		require.NoError(t, old.ConvertFrom(&dynakubelatest.DynaKube{}))

		assert.NotContains(t, old.Annotations, conversion.NetworkZoneFallbackModeKey)

		converted := &dynakubelatest.DynaKube{}
		require.NoError(t, old.ConvertTo(converted))
		assert.Empty(t, converted.Spec.NetworkZoneFallbackMode)
	})

	t.Run("v1beta5 object without annotation converts to empty field", func(t *testing.T) {
		converted := &dynakubelatest.DynaKube{}
		require.NoError(t, (&DynaKube{Spec: DynaKubeSpec{NetworkZone: "zone"}}).ConvertTo(converted))

		assert.Equal(t, "zone", converted.Spec.NetworkZone)
		assert.Empty(t, converted.Spec.NetworkZoneFallbackMode)
	})
}
