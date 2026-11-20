// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package conversion

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRemovedFields(t *testing.T) {
	rf := NewRemovedFields(map[string]string{})

	rf.AutoUpdate.Set(new(true))
	require.NotNil(t, rf.AutoUpdate.Get())
	assert.True(t, *rf.AutoUpdate.Get())

	rf.AutoUpdate.Set(new(false))
	require.NotNil(t, rf.AutoUpdate.Get())
	assert.False(t, *rf.AutoUpdate.Get())

	rf.AutoUpdate.Set(nil)
	assert.Nil(t, rf.AutoUpdate.Get())
}

func TestRemovedFieldsWithAnnotation(t *testing.T) {
	t.Run("true", func(t *testing.T) {
		rf := NewRemovedFields(map[string]string{
			AutoUpdateKey: "true",
		})

		require.NotNil(t, rf.AutoUpdate.Get())
		assert.True(t, *rf.AutoUpdate.Get())
	})

	t.Run("false", func(t *testing.T) {
		rf := NewRemovedFields(map[string]string{
			AutoUpdateKey: "false",
		})

		require.NotNil(t, rf.AutoUpdate.Get())
		assert.False(t, *rf.AutoUpdate.Get())
	})

	t.Run("invalid bool value", func(t *testing.T) {
		rf := NewRemovedFields(map[string]string{
			AutoUpdateKey: "invalid-bool",
		})

		assert.Nil(t, rf.AutoUpdate.Get())
	})
}

func TestRemovedFieldsMutability(t *testing.T) {
	annotations := map[string]string{
		AutoUpdateKey: "true",
	}

	rf := NewRemovedFields(annotations)

	require.NotNil(t, rf.AutoUpdate.Get())
	assert.True(t, *rf.AutoUpdate.Get())

	rf.AutoUpdate.Set(new(false))
	require.NotNil(t, rf.AutoUpdate.Get())
	assert.False(t, *rf.AutoUpdate.Get())

	value, exists := annotations[AutoUpdateKey]
	assert.True(t, exists)

	assert.Equal(t, "false", value)
}

func TestAddedFields(t *testing.T) {
	af := NewAddedFields(map[string]string{})

	af.TLS.Set(&dynakube.TLSSpec{Mode: "Off"})
	require.NotNil(t, af.TLS.Get())
	assert.Equal(t, dynakube.TLSMode("Off"), af.TLS.Get().Mode)

	af.TLS.Set(&dynakube.TLSSpec{Mode: "Automatic"})
	require.NotNil(t, af.TLS.Get())
	assert.Equal(t, dynakube.TLSMode("Automatic"), af.TLS.Get().Mode)

	af.TLS.Set(nil)
	assert.Nil(t, af.TLS.Get())
}

func TestAddedFieldsWithAnnotation(t *testing.T) {
	t.Run("off", func(t *testing.T) {
		af := NewAddedFields(map[string]string{
			TLSModeKey: "{\"mode\":\"Off\"}",
		})

		require.NotNil(t, af.TLS.Get())
		assert.Equal(t, dynakube.TLSMode("Off"), af.TLS.Get().Mode)
	})
	t.Run("automatic", func(t *testing.T) {
		af := NewAddedFields(map[string]string{
			TLSModeKey: "{\"mode\":\"Automatic\"}",
		})

		require.NotNil(t, af.TLS.Get())
		assert.Equal(t, dynakube.TLSMode("Automatic"), af.TLS.Get().Mode)
	})
}
