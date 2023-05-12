package oneagent_mutation

import (
	"path/filepath"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/src/config"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
)

func TestAddOneAgentVolumeMounts(t *testing.T) {
	t.Run("should add oneagent volume mounts", func(t *testing.T) {
		container := &corev1.Container{}
		installPath := "test/path"

		addOneAgentVolumeMounts(container, installPath)
		require.Len(t, container.VolumeMounts, 3)
	})
}

func TestAddReadOnlyCSIVolumeMounts(t *testing.T) {
	t.Run("should add extra volume mounts for readonly csi", func(t *testing.T) {
		container := &corev1.Container{}
		expectedMounts := map[string]string{
			oneagentConfVolumeName:        OneAgentConfMountPath,
			oneagentDataStorageVolumeName: oneagentDataStorageMountPath,
			oneagentLogVolumeName:         oneagentLogMountPath,
		}

		addVolumeMountsForReadOnlyCSI(container)

		require.Len(t, container.VolumeMounts, 3)
		for expectedVolumeName, expectedMountPath := range expectedMounts {
			mount, err := kubeobjects.GetVolumeMountByName(container.VolumeMounts, expectedVolumeName)
			require.NoError(t, err)
			require.NotNil(t, mount)
			assert.Equal(t, expectedMountPath, mount.MountPath)
		}
	})
}

func TestAddCertVolumeMounts(t *testing.T) {
	t.Run("should add cert volume mounts", func(t *testing.T) {
		container := &corev1.Container{}

		addCertVolumeMounts(container)
		require.Len(t, container.VolumeMounts, 1)
		assert.Equal(t, customCertFileName, container.VolumeMounts[0].SubPath)
	})
}

func TestAddCurlOptionsVolumeMount(t *testing.T) {
	t.Run("should add cert volume mounts", func(t *testing.T) {
		container := &corev1.Container{}

		addCurlOptionsVolumeMount(container)
		require.Len(t, container.VolumeMounts, 1)
		assert.Equal(t, filepath.Join(oneAgentCustomKeysPath, config.AgentCurlOptionsFileName), container.VolumeMounts[0].MountPath)
		assert.Equal(t, config.AgentCurlOptionsFileName, container.VolumeMounts[0].SubPath)
	})
}

func TestAddInitVolumeMounts(t *testing.T) {
	t.Run("should add init volume mounts", func(t *testing.T) {
		container := &corev1.Container{}

		addInitVolumeMounts(container, *getTestDynakube())
		require.Len(t, container.VolumeMounts, 2)
	})
	t.Run("if readonly csi, should add extra init volume mounts for readonly csi", func(t *testing.T) {
		container := &corev1.Container{}

		addInitVolumeMounts(container, *getTestReadOnlyCSIDynakube())
		require.Len(t, container.VolumeMounts, 3)

		mount, err := kubeobjects.GetVolumeMountByName(container.VolumeMounts, oneagentConfVolumeName)
		require.NoError(t, err)
		assert.Equal(t, config.AgentConfInitDirMount, mount.MountPath)
	})
}

func TestAddOneAgentVolumes(t *testing.T) {
	t.Run("should add oneagent volumes, with csi", func(t *testing.T) {
		pod := &corev1.Pod{}
		dynakube := getTestCSIDynakube()

		addOneAgentVolumes(pod, *dynakube)
		require.Len(t, pod.Spec.Volumes, 2)
		assert.NotNil(t, pod.Spec.Volumes[0].VolumeSource.CSI)
		assert.False(t, *pod.Spec.Volumes[0].VolumeSource.CSI.ReadOnly)
	})

	t.Run("should add oneagent volumes, with readonly csi", func(t *testing.T) {
		pod := &corev1.Pod{}
		dynakube := getTestReadOnlyCSIDynakube()

		addOneAgentVolumes(pod, *dynakube)
		require.Len(t, pod.Spec.Volumes, 2)
		assert.NotNil(t, pod.Spec.Volumes[0].VolumeSource.CSI)
		assert.True(t, *pod.Spec.Volumes[0].VolumeSource.CSI.ReadOnly)
	})

	t.Run("should add oneagent volumes, without csi", func(t *testing.T) {
		pod := &corev1.Pod{}
		dynakube := getTestDynakube()

		addOneAgentVolumes(pod, *dynakube)
		require.Len(t, pod.Spec.Volumes, 2)
		assert.NotNil(t, pod.Spec.Volumes[0].VolumeSource.EmptyDir)
	})
}

func TestAddReadOnlyCSIVolumes(t *testing.T) {
	t.Run("if enabled, should add extra volumes for readonly csi", func(t *testing.T) {
		pod := &corev1.Pod{}
		expectedVolumes := []string{oneagentConfVolumeName, oneagentDataStorageVolumeName, oneagentLogVolumeName}

		addVolumesForReadOnlyCSI(pod)
		require.Len(t, pod.Spec.Volumes, 3)
		for _, expectedVolumeName := range expectedVolumes {
			mount, err := kubeobjects.GetVolumeByName(pod.Spec.Volumes, expectedVolumeName)
			require.NoError(t, err)
			require.NotNil(t, mount)
		}
	})
}
