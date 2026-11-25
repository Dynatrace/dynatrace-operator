// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package conversion

import (
	"testing"

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
	annotations := map[string]string{}
	af := NewAddedFields(annotations)

	assert.Nil(t, af.NetworkZoneFallbackMode.Get())

	af.NetworkZoneFallbackMode.Set(new("none"))
	require.NotNil(t, af.NetworkZoneFallbackMode.Get())
	assert.Equal(t, "none", *af.NetworkZoneFallbackMode.Get())

	af.Cleanup()
	assert.Empty(t, annotations)
}

func TestRemovedFieldsCleanup(t *testing.T) {
	annotations := map[string]string{"other": "annotation"}
	rf := NewRemovedFields(annotations)

	rf.AutoUpdate.Set(new(true))
	rf.DefaultOTelColImage.Set(new(true))

	rf.Cleanup()

	assert.Equal(t, map[string]string{"other": "annotation"}, annotations)
}
