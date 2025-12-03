package k8smount

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
)

func TestFind(t *testing.T) {
	mounts := []corev1.VolumeMount{
		{Name: "vol1", MountPath: "/mnt/vol1"},
		{Name: "vol2", MountPath: "/mnt/vol2"},
	}

	t.Run("found", func(t *testing.T) {
		vm, err := Find(mounts, "vol1")
		require.NoError(t, err)
		assert.NotNil(t, vm)
		assert.Equal(t, "vol1", vm.Name)
	})

	t.Run("not found", func(t *testing.T) {
		vm, err := Find(mounts, "vol3")
		require.Error(t, err)
		assert.Nil(t, vm)
	})
}

func TestContainsPath(t *testing.T) {
	mounts := []corev1.VolumeMount{
		{Name: "vol1", MountPath: "/mnt/vol1"},
		{Name: "vol2", MountPath: "/mnt/vol2"},
	}

	t.Run("contains", func(t *testing.T) {
		assert.True(t, ContainsPath(mounts, "/mnt/vol1"))
	})

	t.Run("not contains", func(t *testing.T) {
		assert.False(t, ContainsPath(mounts, "/mnt/vol3"))
	})
}

func TestContains(t *testing.T) {
	mounts := []corev1.VolumeMount{
		{Name: "vol1", MountPath: "/mnt/vol1"},
		{Name: "vol2", MountPath: "/mnt/vol2"},
	}

	t.Run("contains", func(t *testing.T) {
		assert.True(t, Contains(mounts, "vol1"))
	})

	t.Run("not contains", func(t *testing.T) {
		assert.False(t, Contains(mounts, "vol3"))
	})
}

func TestAppend(t *testing.T) {
	mounts := []corev1.VolumeMount{
		{Name: "vol1", MountPath: "/mnt/vol1"},
	}

	t.Run("append new", func(t *testing.T) {
		newMount := corev1.VolumeMount{Name: "vol2", MountPath: "/mnt/vol2"}
		result := Append(mounts, newMount)
		assert.Len(t, result, 2)
		assert.Equal(t, "vol2", result[1].Name)
	})

	t.Run("append existing", func(t *testing.T) {
		existingMount := corev1.VolumeMount{Name: "vol1-duplicate", MountPath: "/mnt/vol1"}
		result := Append(mounts, existingMount)
		assert.Len(t, result, 1)
	})

	t.Run("append multiple", func(t *testing.T) {
		newMount1 := corev1.VolumeMount{Name: "vol2", MountPath: "/mnt/vol2"}
		newMount2 := corev1.VolumeMount{Name: "vol3", MountPath: "/mnt/vol3"}
		existingMount := corev1.VolumeMount{Name: "vol1-duplicate", MountPath: "/mnt/vol1"}

		result := Append(mounts, newMount1, existingMount, newMount2)
		assert.Len(t, result, 3)
		assert.Equal(t, "vol2", result[1].Name)
		assert.Equal(t, "vol3", result[2].Name)
	})
}
