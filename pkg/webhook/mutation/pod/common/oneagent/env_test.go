package oneagent

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/oneagent"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/deploymentmetadata"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/env"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
)

func TestAddPreloadEnv(t *testing.T) {
	installPath := "path/test"

	t.Run("Add preload env", func(t *testing.T) {
		container := createContainerWithPreloadEnv("")

		AddPreloadEnv(container, installPath)

		verifyContainerWithPreloadEnv(t, container, installPath)
	})
	t.Run("Concat preload env, default delimiter", func(t *testing.T) {
		existingPath := "path/user"
		container := createContainerWithPreloadEnv(existingPath)

		AddPreloadEnv(container, installPath)

		verifyContainerWithPreloadEnv(t, container, existingPath+":"+installPath)
	})
	t.Run("Concat preload env, respect delimiter", func(t *testing.T) {
		existingPath := "path1/user path2/user"
		container := createContainerWithPreloadEnv(existingPath)

		AddPreloadEnv(container, installPath)

		verifyContainerWithPreloadEnv(t, container, existingPath+" "+installPath)
	})
	t.Run("Ignore preload env, if value already present", func(t *testing.T) {
		existingPath := "path1/user path2/user"
		existingPath += " " + installPath
		container := createContainerWithPreloadEnv(existingPath)

		AddPreloadEnv(container, installPath)

		verifyContainerWithPreloadEnv(t, container, existingPath)
	})
}

func createContainerWithPreloadEnv(existingPath string) *corev1.Container {
	container := &corev1.Container{
		Env: []corev1.EnvVar{
			{
				Name:  "some-other-env",
				Value: "some-value",
			},
		},
	}
	if existingPath != "" {
		container.Env = append(container.Env, corev1.EnvVar{
			Name:  PreloadEnv,
			Value: existingPath,
		})
	}

	return container
}

func verifyContainerWithPreloadEnv(t *testing.T, container *corev1.Container, expectedPath string) {
	require.NotEmpty(t, container.Env)
	containerEnv := env.FindEnvVar(container.Env, PreloadEnv)
	require.NotNil(t, containerEnv)
	assert.Contains(t, containerEnv.Value, expectedPath)
}

func TestAddNetworkZoneEnv(t *testing.T) {
	t.Run("Add networkZone env", func(t *testing.T) {
		container := &corev1.Container{}
		networkZone := "testZone"

		AddNetworkZoneEnv(container, networkZone)

		require.Len(t, container.Env, 1)
		assert.Equal(t, NetworkZoneEnv, container.Env[0].Name)
		assert.Equal(t, networkZone, container.Env[0].Value)
	})
}

func TestAddDeploymentMetadataEnv(t *testing.T) {
	clusterID := "cluster-id"

	t.Run("Add cloudNative deployment metadata env", func(t *testing.T) {
		container := &corev1.Container{}
		dk := dynakube.DynaKube{
			Spec: dynakube.DynaKubeSpec{
				OneAgent: oneagent.Spec{
					CloudNativeFullStack: &oneagent.CloudNativeFullStackSpec{},
				},
			},
			Status: dynakube.DynaKubeStatus{
				KubeSystemUUID: clusterID,
			},
		}
		AddDeploymentMetadataEnv(container, dk)
		require.Len(t, container.Env, 1)
		assert.Contains(t, container.Env[0].Value, clusterID)
		assert.Contains(t, container.Env[0].Value, deploymentmetadata.CloudNativeDeploymentType)
	})

	t.Run("Add appMonitoring deployment metadata env", func(t *testing.T) {
		container := &corev1.Container{}
		dk := dynakube.DynaKube{
			Spec: dynakube.DynaKubeSpec{
				OneAgent: oneagent.Spec{
					ApplicationMonitoring: &oneagent.ApplicationMonitoringSpec{},
				},
			},
			Status: dynakube.DynaKubeStatus{
				KubeSystemUUID: clusterID,
			},
		}
		AddDeploymentMetadataEnv(container, dk)
		require.Len(t, container.Env, 1)
		assert.Contains(t, container.Env[0].Value, clusterID)
		assert.Contains(t, container.Env[0].Value, deploymentmetadata.ApplicationMonitoringDeploymentType)
	})
}

func TestAddVersionDetection(t *testing.T) {
	t.Run("adds defaults", func(t *testing.T) {
		container := &corev1.Container{}

		AddVersionDetectionEnvs(container, getTestNamespace(defaultVersionLabelMapping))

		require.Len(t, container.Env, len(defaultVersionLabelMapping))

		for _, envvar := range container.Env {
			assert.Equal(t, defaultVersionLabelMapping[envvar.Name], envvar.ValueFrom.FieldRef.FieldPath)
		}
	})

	t.Run("not overwrite present envs", func(t *testing.T) {
		testVersion := "1.2.3"
		testProduct := "testy"
		container := &corev1.Container{
			Env: []corev1.EnvVar{
				{Name: ReleaseVersionEnv, Value: testVersion},
				{Name: ReleaseProductEnv, Value: testProduct},
			},
		}

		AddVersionDetectionEnvs(container, getTestNamespace(defaultVersionLabelMapping))

		require.Len(t, container.Env, 2)
		assert.Equal(t, testVersion, container.Env[0].Value)
		assert.Equal(t, testProduct, container.Env[1].Value)
	})

	t.Run("partial addition", func(t *testing.T) {
		testVersion := "1.2.3"
		container := &corev1.Container{
			Env: []corev1.EnvVar{
				{Name: ReleaseVersionEnv, Value: testVersion},
			},
		}

		AddVersionDetectionEnvs(container, getTestNamespace(defaultVersionLabelMapping))

		require.Len(t, container.Env, 2)
		assert.Equal(t, testVersion, container.Env[0].Value)
		assert.Equal(t, defaultVersionLabelMapping[ReleaseProductEnv], container.Env[1].ValueFrom.FieldRef.FieldPath)
	})
}
