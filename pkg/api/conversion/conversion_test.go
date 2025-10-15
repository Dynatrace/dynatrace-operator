package conversion

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/utils/ptr"
)

func TestRemovedFields(t *testing.T) {
	rf := NewRemovedFields(map[string]string{})

	rf.AutoUpdate.Set(ptr.To(true))
	require.NotNil(t, rf.AutoUpdate.Get())
	assert.True(t, *rf.AutoUpdate.Get())

	rf.AutoUpdate.Set(ptr.To(false))
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

	rf.AutoUpdate.Set(ptr.To(false))
	require.NotNil(t, rf.AutoUpdate.Get())
	assert.False(t, *rf.AutoUpdate.Get())

	value, exists := annotations[AutoUpdateKey]
	assert.True(t, exists)

	assert.Equal(t, "false", value)
}
