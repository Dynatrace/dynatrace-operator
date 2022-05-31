package oneagent_mutation

import (
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
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

func TestAddCertVolumeMounts(t *testing.T) {
	t.Run("should add cert volume mounts", func(t *testing.T) {
		container := &corev1.Container{}

		addCertVolumeMounts(container)
		require.Len(t, container.VolumeMounts, 1)
		assert.Equal(t, customCertFileName, container.VolumeMounts[0].SubPath)
	})
}

func TestAddInitVolumeMounts(t *testing.T) {
	t.Run("should add init volume mounts", func(t *testing.T) {
		container := &corev1.Container{}

		addInitVolumeMounts(container)
		require.Len(t, container.VolumeMounts, 2)
	})
}

func TestAddOneAgentVolumes(t *testing.T) {
	t.Run("should add oneagent volumes, with csi", func(t *testing.T) {
		pod := &corev1.Pod{}
		dynakube := dynatracev1beta1.DynaKube{
			Spec: dynatracev1beta1.DynaKubeSpec{
				OneAgent: dynatracev1beta1.OneAgentSpec{
					CloudNativeFullStack: &dynatracev1beta1.CloudNativeFullStackSpec{},
				},
			},
		}

		addOneAgentVolumes(pod, dynakube)
		require.Len(t, pod.Spec.Volumes, 2)
		assert.NotNil(t, pod.Spec.Volumes[0].VolumeSource.CSI)
	})

	t.Run("should add oneagent volumes, without csi", func(t *testing.T) {
		pod := &corev1.Pod{}
		dynakube := dynatracev1beta1.DynaKube{
			Spec: dynatracev1beta1.DynaKubeSpec{
				OneAgent: dynatracev1beta1.OneAgentSpec{
					ApplicationMonitoring: &dynatracev1beta1.ApplicationMonitoringSpec{},
				},
			},
		}

		addOneAgentVolumes(pod, dynakube)
		require.Len(t, pod.Spec.Volumes, 2)
		assert.NotNil(t, pod.Spec.Volumes[0].VolumeSource.EmptyDir)
	})
}
