package daemonset

import (
	"path/filepath"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/kspm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"
)

const (
	expectedMountLen = 1

	hostPathA = "/a"
	hostPathB = "/b"
)

func TestGetMounts(t *testing.T) {
	testMappedVolumeMounts := func(mounts []corev1.VolumeMount) {
		require.NotEmpty(t, mounts)
		assert.Len(t, mounts, expectedMountLen+2)

		for _, mount := range mounts {
			assert.NotEmpty(t, mount.Name)
			assert.NotEmpty(t, mount.MountPath)
		}

		assert.Contains(t, mounts, corev1.VolumeMount{
			Name:      getVolumeName(0),
			MountPath: filepath.Join(nodeRootMountPath, hostPathA),
			ReadOnly:  true,
		})
		assert.Contains(t, mounts, corev1.VolumeMount{
			Name:      getVolumeName(1),
			MountPath: filepath.Join(nodeRootMountPath, hostPathB),
			ReadOnly:  true,
		})
	}

	t.Run("get volume mounts", func(t *testing.T) {
		dk := dynakube.DynaKube{
			Spec: dynakube.DynaKubeSpec{
				Kspm: &kspm.Spec{},
			},
		}
		mounts := getMounts(dk)

		require.NotEmpty(t, mounts)
		assert.Len(t, mounts, expectedMountLen)

		for _, mount := range mounts {
			assert.NotEmpty(t, mount.Name)
			assert.NotEmpty(t, mount.MountPath)
		}
	})

	t.Run("get volume mounts with mapped paths", func(t *testing.T) {
		dk := dynakube.DynaKube{
			Spec: dynakube.DynaKubeSpec{
				Kspm: &kspm.Spec{
					MappedHostPaths: []string{
						hostPathA,
						hostPathB,
					},
				},
			},
		}
		mounts := getMounts(dk)

		testMappedVolumeMounts(mounts)
	})

	t.Run("get volume mounts with duplicated mapped paths", func(t *testing.T) {
		dk := dynakube.DynaKube{
			Spec: dynakube.DynaKubeSpec{
				Kspm: &kspm.Spec{
					MappedHostPaths: []string{
						hostPathA,
						hostPathB,
						hostPathA,
						hostPathB,
					},
				},
			},
		}
		mounts := getMounts(dk)

		testMappedVolumeMounts(mounts)
	})

	t.Run("get cert mount", func(t *testing.T) {
		dk := getDynaKubeWithCerts(t)
		dk.Spec.Kspm = &kspm.Spec{}
		mounts := getMounts(dk)

		require.NotEmpty(t, mounts)
		assert.Len(t, mounts, expectedMountLen+1)

		for _, mount := range mounts {
			assert.NotEmpty(t, mount.Name)
			assert.NotEmpty(t, mount.MountPath)
		}
	})

	t.Run("get cert mount with automatic AG cert", func(t *testing.T) {
		dk := getDynaKubeWithAutomaticCerts(t)
		dk.Spec.Kspm = &kspm.Spec{}
		mounts := getMounts(dk)

		require.NotEmpty(t, mounts)
		assert.Len(t, mounts, expectedMountLen+1)

		for _, mount := range mounts {
			assert.NotEmpty(t, mount.Name)
			assert.NotEmpty(t, mount.MountPath)
		}
	})
}

func TestGetVolumes(t *testing.T) {
	testMappedVolumes := func(volumes []corev1.Volume) {
		require.NotEmpty(t, volumes)
		assert.Len(t, volumes, expectedMountLen+2)

		for _, volume := range volumes {
			assert.NotEmpty(t, volume.Name)
			assert.NotEmpty(t, volume.VolumeSource)
		}

		assert.Contains(t, volumes, corev1.Volume{
			Name: getVolumeName(0),
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: hostPathA,
					Type: ptr.To(corev1.HostPathDirectory),
				},
			},
		})
		assert.Contains(t, volumes, corev1.Volume{
			Name: getVolumeName(1),
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: hostPathB,
					Type: ptr.To(corev1.HostPathDirectory),
				},
			},
		})
	}

	t.Run("get volumes", func(t *testing.T) {
		dk := dynakube.DynaKube{
			Spec: dynakube.DynaKubeSpec{
				Kspm: &kspm.Spec{},
			},
		}
		volumes := getVolumes(dk)

		require.NotEmpty(t, volumes)
		assert.Len(t, volumes, expectedMountLen)

		for _, volume := range volumes {
			assert.NotEmpty(t, volume.Name)
			assert.NotEmpty(t, volume.VolumeSource)
		}
	})

	t.Run("get volume mounts with mapped paths", func(t *testing.T) {
		dk := dynakube.DynaKube{
			Spec: dynakube.DynaKubeSpec{
				Kspm: &kspm.Spec{
					MappedHostPaths: []string{
						hostPathA,
						hostPathB,
					},
				},
			},
		}
		volumes := getVolumes(dk)

		testMappedVolumes(volumes)
	})

	t.Run("get volume mounts with duplicated mapped paths", func(t *testing.T) {
		dk := dynakube.DynaKube{
			Spec: dynakube.DynaKubeSpec{
				Kspm: &kspm.Spec{
					MappedHostPaths: []string{
						hostPathA,
						hostPathB,
						hostPathA,
						hostPathB,
					},
				},
			},
		}
		volumes := getVolumes(dk)

		testMappedVolumes(volumes)
	})

	t.Run("add cert volume", func(t *testing.T) {
		dk := getDynaKubeWithCerts(t)
		dk.Spec.Kspm = &kspm.Spec{}
		volumes := getVolumes(dk)

		require.NotEmpty(t, volumes)
		assert.Len(t, volumes, expectedMountLen+1)

		for _, volume := range volumes {
			assert.NotEmpty(t, volume.Name)
			require.NotEmpty(t, volume.VolumeSource)

			if volume.Name == certVolumeName {
				assert.NotEmpty(t, volume.VolumeSource.Secret.SecretName)
			}
		}
	})

	t.Run("add cert volume with automatic AG cert", func(t *testing.T) {
		dk := getDynaKubeWithAutomaticCerts(t)
		dk.Spec.Kspm = &kspm.Spec{}
		volumes := getVolumes(dk)

		require.NotEmpty(t, volumes)
		assert.Len(t, volumes, expectedMountLen+1)

		for _, volume := range volumes {
			assert.NotEmpty(t, volume.Name)
			require.NotEmpty(t, volume.VolumeSource)

			if volume.Name == certVolumeName {
				assert.NotEmpty(t, volume.VolumeSource.Secret.SecretName)
			}
		}
	})
}
