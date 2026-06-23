package oneagent

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	dtcsi "github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi"
	csivolumes "github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi/server/volumes"
	appvolumes "github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi/server/volumes/app"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/volumes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestAddVolumeMounts(t *testing.T) {
	t.Run("should add oneagent volume mounts", func(t *testing.T) {
		container := &corev1.Container{
			Name: "test-container",
		}
		installPath := "test/path"

		addVolumeMounts(container, installPath)
		require.Len(t, container.VolumeMounts, 2)
		assert.Equal(t, BinVolumeName, container.VolumeMounts[0].Name)
		assert.Equal(t, installPath, container.VolumeMounts[0].MountPath)
		assert.True(t, container.VolumeMounts[0].ReadOnly)

		assert.Equal(t, volumes.ConfigVolumeName, container.VolumeMounts[1].Name)
		assert.Equal(t, ldPreloadPath, container.VolumeMounts[1].MountPath)
		assert.Equal(t, ldPreloadSubPath, container.VolumeMounts[1].SubPath)
	})
}

func TestAddInitVolumeMounts(t *testing.T) {
	t.Run("should add init volume mounts", func(t *testing.T) {
		container := &corev1.Container{}
		readonly := false

		addInitBinMount(container, readonly)
		require.Len(t, container.VolumeMounts, 1)
		assert.Equal(t, BinVolumeName, container.VolumeMounts[0].Name)
		assert.Equal(t, consts.AgentInitBinDirMount, container.VolumeMounts[0].MountPath)
		assert.Equal(t, readonly, container.VolumeMounts[0].ReadOnly)
	})

	t.Run("should add readonly init volume mounts", func(t *testing.T) {
		container := &corev1.Container{}
		readonly := true

		addInitBinMount(container, readonly)
		require.Len(t, container.VolumeMounts, 1)
		assert.Equal(t, BinVolumeName, container.VolumeMounts[0].Name)
		assert.Equal(t, consts.AgentInitBinDirMount, container.VolumeMounts[0].MountPath)
		assert.Equal(t, readonly, container.VolumeMounts[0].ReadOnly)
	})
}

func Test_addEmptyDirBinVolume(t *testing.T) {
	t.Run("should add empty dir bin volume", func(t *testing.T) {
		pod := &corev1.Pod{
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{Name: "test-container", Image: "test-image"},
				},
			},
		}
		addEmptyDirBinVolume(pod, logd.Get())

		assert.Len(t, pod.Spec.Volumes, 1)

		assert.Equal(t, corev1.Volume{
			Name: BinVolumeName,
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		}, pod.Spec.Volumes[0])
	})

	t.Run("should add empty dir bin volume with sizeLimit using pod annotation", func(t *testing.T) {
		pod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					"volume.dynatrace.com/oneagent-bin": "500Mi",
				},
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{Name: "test-container", Image: "test-image"},
				},
			},
		}
		addEmptyDirBinVolume(pod, logd.Get())

		assert.Len(t, pod.Spec.Volumes, 1)

		assert.Equal(t, corev1.Volume{
			Name: BinVolumeName,
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{
					SizeLimit: new(resource.MustParse("500Mi")),
				},
			},
		}, pod.Spec.Volumes[0])
	})

	t.Run("existing volume", func(t *testing.T) {
		pod := &corev1.Pod{
			Spec: corev1.PodSpec{
				Volumes: []corev1.Volume{
					{Name: BinVolumeName, VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{Medium: corev1.StorageMediumHugePages}}},
				},
			},
		}
		expectedPod := pod.DeepCopy()

		require.NoError(t, addEmptyDirBinVolume(pod, logd.Get()))
		assert.Equal(t, expectedPod, pod)
	})

	t.Run("conflicting volume", func(t *testing.T) {
		pod := &corev1.Pod{
			Spec: corev1.PodSpec{
				Volumes: []corev1.Volume{
					{Name: BinVolumeName, VolumeSource: corev1.VolumeSource{HostPath: &corev1.HostPathVolumeSource{Path: "/"}}},
				},
			},
		}

		require.Error(t, addEmptyDirBinVolume(pod, logd.Get()))
	})
}

func Test_addCSIBinVolume(t *testing.T) {
	t.Run("should add CSI bin volume", func(t *testing.T) {
		pod := &corev1.Pod{
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{Name: "test-container", Image: "test-image"},
				},
			},
		}

		require.NoError(t, addCSIBinVolume(pod, "test-dk", "10m"))

		assert.Len(t, pod.Spec.Volumes, 1)

		assert.Equal(t, corev1.Volume{
			Name: BinVolumeName,
			VolumeSource: corev1.VolumeSource{
				CSI: &corev1.CSIVolumeSource{
					Driver:   dtcsi.DriverName,
					ReadOnly: new(true),
					VolumeAttributes: map[string]string{
						csivolumes.CSIVolumeAttributeModeField:     appvolumes.Mode,
						csivolumes.CSIVolumeAttributeDynakubeField: "test-dk",
						csivolumes.CSIVolumeAttributeRetryTimeout:  "10m",
					},
				},
			},
		}, pod.Spec.Volumes[0])
	})

	t.Run("existing volume", func(t *testing.T) {
		pod := &corev1.Pod{
			Spec: corev1.PodSpec{
				Volumes: []corev1.Volume{
					{Name: BinVolumeName, VolumeSource: corev1.VolumeSource{CSI: &corev1.CSIVolumeSource{Driver: dtcsi.DriverName}}},
				},
			},
		}
		expectedPod := pod.DeepCopy()

		require.NoError(t, addCSIBinVolume(pod, "test-dk", "10m"))
		assert.Equal(t, expectedPod, pod)
	})

	t.Run("conflicting volume", func(t *testing.T) {
		pod := &corev1.Pod{
			Spec: corev1.PodSpec{
				Volumes: []corev1.Volume{
					{Name: BinVolumeName, VolumeSource: corev1.VolumeSource{HostPath: &corev1.HostPathVolumeSource{Path: "/"}}},
				},
			},
		}

		require.Error(t, addCSIBinVolume(pod, "test-dk", "10m"))
	})
}
