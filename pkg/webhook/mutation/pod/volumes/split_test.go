package volumes

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/oneagent"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"
)

func TestAddSplitMounts(t *testing.T) {
	t.Run("should add both oneagent and enrichment mounts if enabled", func(t *testing.T) {
		container := &corev1.Container{Name: "test-container"}
		dk := dynakube.DynaKube{
			Spec: dynakube.DynaKubeSpec{
				OneAgent: oneagent.Spec{
					ApplicationMonitoring: &oneagent.ApplicationMonitoringSpec{},
				},
				MetadataEnrichment: dynakube.MetadataEnrichment{
					Enabled: ptr.To(true),
				},
			},
		}
		request := &dtwebhook.BaseRequest{
			DynaKube: dk,
		}

		addSplitMounts(container, request)

		assert.True(t, HasSplitOneAgentMounts(container))
		assert.True(t, HasSplitEnrichmentMounts(container))
		assert.Len(t, container.VolumeMounts, 4) // 1 for OneAgent + 3 for Enrichment
	})

	t.Run("should add only enrichment mounts if oneagent is disabled", func(t *testing.T) {
		container := &corev1.Container{Name: "test-container"}
		dk := dynakube.DynaKube{
			Spec: dynakube.DynaKubeSpec{
				MetadataEnrichment: dynakube.MetadataEnrichment{
					Enabled: ptr.To(true),
				},
			},
		}
		request := &dtwebhook.BaseRequest{
			DynaKube: dk,
		}

		addSplitMounts(container, request)

		assert.False(t, HasSplitOneAgentMounts(container))
		assert.True(t, HasSplitEnrichmentMounts(container))
		assert.Len(t, container.VolumeMounts, 3)
	})

	t.Run("should add only oneagent mounts if enrichment is disabled", func(t *testing.T) {
		container := &corev1.Container{Name: "test-container"}
		dk := dynakube.DynaKube{
			Spec: dynakube.DynaKubeSpec{
				OneAgent: oneagent.Spec{
					ApplicationMonitoring: &oneagent.ApplicationMonitoringSpec{},
				},
				MetadataEnrichment: dynakube.MetadataEnrichment{
					Enabled: ptr.To(false),
				},
			},
		}
		request := &dtwebhook.BaseRequest{
			DynaKube: dk,
		}

		addSplitMounts(container, request)

		assert.True(t, HasSplitOneAgentMounts(container))
		assert.False(t, HasSplitEnrichmentMounts(container))
		assert.Len(t, container.VolumeMounts, 1)
	})

	t.Run("should add nothing if both are disabled", func(t *testing.T) {
		container := &corev1.Container{Name: "test-container"}
		dk := dynakube.DynaKube{
			Spec: dynakube.DynaKubeSpec{
				MetadataEnrichment: dynakube.MetadataEnrichment{
					Enabled: ptr.To(false),
				},
			},
		}
		request := &dtwebhook.BaseRequest{
			DynaKube: dk,
		}

		addSplitMounts(container, request)

		assert.False(t, HasSplitOneAgentMounts(container))
		assert.False(t, HasSplitEnrichmentMounts(container))
		assert.Empty(t, container.VolumeMounts)
	})
}

func TestHasSplitEnrichmentMounts(t *testing.T) {
	t.Run("should return true if json path is present", func(t *testing.T) {
		container := &corev1.Container{
			VolumeMounts: []corev1.VolumeMount{
				{MountPath: configEnrichmentJSONMountPath},
			},
		}
		assert.True(t, HasSplitEnrichmentMounts(container))
	})

	t.Run("should return true if properties path is present", func(t *testing.T) {
		container := &corev1.Container{
			VolumeMounts: []corev1.VolumeMount{
				{MountPath: configEnrichmentPropertiesMountPath},
			},
		}
		assert.True(t, HasSplitEnrichmentMounts(container))
	})

	t.Run("should return true if endpoints path is present", func(t *testing.T) {
		container := &corev1.Container{
			VolumeMounts: []corev1.VolumeMount{
				{MountPath: configEnrichmentEndpointsMountPath},
			},
		}
		assert.True(t, HasSplitEnrichmentMounts(container))
	})

	t.Run("should return false if no enrichment paths are present", func(t *testing.T) {
		container := &corev1.Container{
			VolumeMounts: []corev1.VolumeMount{
				{MountPath: "/other/path"},
			},
		}
		assert.False(t, HasSplitEnrichmentMounts(container))
	})
}

func TestHasSplitOneAgentMounts(t *testing.T) {
	t.Run("should return true if oneagent path is present", func(t *testing.T) {
		container := &corev1.Container{
			VolumeMounts: []corev1.VolumeMount{
				{MountPath: configOneAgentMountPath},
			},
		}
		assert.True(t, HasSplitOneAgentMounts(container))
	})

	t.Run("should return false if oneagent path is not present", func(t *testing.T) {
		container := &corev1.Container{
			VolumeMounts: []corev1.VolumeMount{
				{MountPath: "/other/path"},
			},
		}
		assert.False(t, HasSplitOneAgentMounts(container))
	})
}

func TestAddSplitOneAgentConfigVolumeMount(t *testing.T) {
	container := &corev1.Container{Name: "test-container"}
	addSplitOneAgentConfigVolumeMount(container)

	require.Len(t, container.VolumeMounts, 1)
	vm := container.VolumeMounts[0]
	assert.Equal(t, ConfigVolumeName, vm.Name)
	assert.Equal(t, configOneAgentMountPath, vm.MountPath)
	assert.Equal(t, configOneAgentSubPath(container.Name), vm.SubPath)
}

func TestAddSplitEnrichmentConfigVolumeMount(t *testing.T) {
	container := &corev1.Container{Name: "test-container"}
	addSplitEnrichmentConfigVolumeMount(container)

	require.Len(t, container.VolumeMounts, 3)

	// Check JSON mount
	jsonMount := container.VolumeMounts[0]
	assert.Equal(t, ConfigVolumeName, jsonMount.Name)
	assert.Equal(t, configEnrichmentJSONMountPath, jsonMount.MountPath) //nolint:testifylint
	assert.Equal(t, configEnrichmentJSONSubPath(container.Name), jsonMount.SubPath)

	// Check Properties mount
	propsMount := container.VolumeMounts[1]
	assert.Equal(t, ConfigVolumeName, propsMount.Name)
	assert.Equal(t, configEnrichmentPropertiesMountPath, propsMount.MountPath)
	assert.Equal(t, configEnrichmentPropertiesSubPath(container.Name), propsMount.SubPath)

	// Check Endpoints mount
	endpointsMount := container.VolumeMounts[2]
	assert.Equal(t, ConfigVolumeName, endpointsMount.Name)
	assert.Equal(t, configEnrichmentEndpointsMountPath, endpointsMount.MountPath)
	assert.Equal(t, configEnrichmentEndpointsSubPath(container.Name), endpointsMount.SubPath)
}
