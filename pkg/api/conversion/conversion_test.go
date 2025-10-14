package conversion

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/utils/ptr"
)

func TestRemovedFields(t *testing.T) {
	rf := NewRemovedFields(map[string]string{})

	rf.SetAutoUpdate(ptr.To(true))
	require.NotNil(t, rf.GetAutoUpdate())
	assert.True(t, *rf.GetAutoUpdate())

	rf.SetAutoUpdate(ptr.To(false))
	require.NotNil(t, rf.GetAutoUpdate())
	assert.False(t, *rf.GetAutoUpdate())

	rf.SetAutoUpdate(nil)
	assert.Nil(t, rf.GetAutoUpdate())
}

func TestRemovedFieldsWithAnnotation(t *testing.T) {
	t.Run("true", func(t *testing.T) {
		rf := NewRemovedFields(map[string]string{
			AutoUpdateKey: "true",
		})

		require.NotNil(t, rf.GetAutoUpdate())
		assert.True(t, *rf.GetAutoUpdate())
	})

	t.Run("false", func(t *testing.T) {
		rf := NewRemovedFields(map[string]string{
			AutoUpdateKey: "false",
		})

		require.NotNil(t, rf.GetAutoUpdate())
		assert.False(t, *rf.GetAutoUpdate())
	})

	t.Run("invalid bool value", func(t *testing.T) {
		rf := NewRemovedFields(map[string]string{
			AutoUpdateKey: "invalid-bool",
		})

		assert.Nil(t, rf.GetAutoUpdate())
	})
}

func TestRemovedFieldsMutability(t *testing.T) {
	annotations := map[string]string{
		AutoUpdateKey: "true",
	}

	rf := NewRemovedFields(annotations)

	require.NotNil(t, rf.GetAutoUpdate())
	assert.True(t, *rf.GetAutoUpdate())

	rf.SetAutoUpdate(ptr.To(false))
	require.NotNil(t, rf.GetAutoUpdate())
	assert.False(t, *rf.GetAutoUpdate())

	value, exists := annotations[AutoUpdateKey]
	assert.True(t, exists)

	assert.Equal(t, "false", value)
}
