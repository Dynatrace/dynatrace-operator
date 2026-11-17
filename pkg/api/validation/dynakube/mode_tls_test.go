// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package validation

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTLS(t *testing.T) {
	t.Run("TLS mode off => warning", func(t *testing.T) {
		dk := &dynakube.DynaKube{
			Spec: dynakube.DynaKubeSpec{
				APIURL: testAPIURL,
				TLS:    &dynakube.TLSSpec{Mode: dynakube.TLSModeOff},
			},
		}

		warnings, err := assertAllowed(t, dk)
		require.NoError(t, err)
		require.Len(t, warnings, 1)
		assert.Equal(t, warnDisabledTLSForPrometheus, warnings[0])
	})
	t.Run("TLS mode automatic => no warnings", func(t *testing.T) {
		dk := &dynakube.DynaKube{
			Spec: dynakube.DynaKubeSpec{
				APIURL: testAPIURL,
				TLS:    &dynakube.TLSSpec{Mode: dynakube.TLSModeAutomatic},
			},
		}

		assertAllowedWithoutWarnings(t, dk)
	})
	t.Run("TLS mode automatic by default => no warnings", func(t *testing.T) {
		dk := &dynakube.DynaKube{
			Spec: dynakube.DynaKubeSpec{
				APIURL: testAPIURL,
			},
		}

		assertAllowedWithoutWarnings(t, dk)
	})
}
