package oneagent_mutation

import (
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/deploymentmetadata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
)

func TestAddPreloadEnv(t *testing.T) {
	t.Run("Add preload env", func(t *testing.T) {
		container := &corev1.Container{}
		installPath := "path/test"

		addPreloadEnv(container, installPath)

		require.Len(t, container.Env, 1)
		assert.Equal(t, container.Env[0].Name, preloadEnv)
		assert.Contains(t, container.Env[0].Value, installPath)
	})
}

func TestAddNetworkZoneEnv(t *testing.T) {
	t.Run("Add networkZone env", func(t *testing.T) {
		container := &corev1.Container{}
		networkZone := "testZone"

		addNetworkZoneEnv(container, networkZone)

		require.Len(t, container.Env, 1)
		assert.Equal(t, container.Env[0].Name, networkZoneEnv)
		assert.Equal(t, container.Env[0].Value, networkZone)
	})
}

func TestAddProxyEnv(t *testing.T) {
	t.Run("Add proxy env", func(t *testing.T) {
		container := &corev1.Container{}

		addProxyEnv(container)

		require.Len(t, container.Env, 1)
		assert.IsType(t, container.Env[0].ValueFrom, &corev1.EnvVarSource{})
	})
}

func TestAddInstallerInitEnvs(t *testing.T) {
	t.Run("Add installer init env", func(t *testing.T) {
		container := &corev1.Container{}
		testVolumeMode := "testMode"
		installerInfo := getTestInstallerInfo()
		addInstallerInitEnvs(container, installerInfo, testVolumeMode)
		require.Len(t, container.Env, expectedBaseInitContainerEnvCount)
		assert.Equal(t, installerInfo.flavor, container.Env[0].Value)
		assert.Equal(t, installerInfo.technologies, container.Env[1].Value)
		assert.Equal(t, installerInfo.installPath, container.Env[2].Value)
		assert.Equal(t, installerInfo.installerURL, container.Env[3].Value)
		assert.Equal(t, installerInfo.version, container.Env[4].Value)
		assert.Equal(t, testVolumeMode, container.Env[5].Value)
		assert.Equal(t, "true", container.Env[6].Value)
	})
}

func TestAddContainerInfoInitEnv(t *testing.T) {
	t.Run("Add container info init env", func(t *testing.T) {
		container := &corev1.Container{}
		addContainerInfoInitEnv(container, 1, "test-pod", "test-namespace")
		require.Len(t, container.Env, 2)
	})
}

func TestAddDeploymentMetadataEnv(t *testing.T) {
	t.Run("Add cloudNative deployment metadata env", func(t *testing.T) {
		container := &corev1.Container{}
		dynakube := dynatracev1beta1.DynaKube{
			Spec: dynatracev1beta1.DynaKubeSpec{
				OneAgent: dynatracev1beta1.OneAgentSpec{
					CloudNativeFullStack: &dynatracev1beta1.CloudNativeFullStackSpec{},
				},
			},
		}
		addDeploymentMetadataEnv(container, dynakube, testClusterID)
		require.Len(t, container.Env, 1)
		assert.Contains(t, container.Env[0].Value, testClusterID)
		assert.Contains(t, container.Env[0].Value, deploymentmetadata.DeploymentTypeCloudNative)
	})

	t.Run("Add appMonitoring deployment metadata env", func(t *testing.T) {
		container := &corev1.Container{}
		dynakube := dynatracev1beta1.DynaKube{
			Spec: dynatracev1beta1.DynaKubeSpec{
				OneAgent: dynatracev1beta1.OneAgentSpec{
					ApplicationMonitoring: &dynatracev1beta1.ApplicationMonitoringSpec{},
				},
			},
		}
		addDeploymentMetadataEnv(container, dynakube, testClusterID)
		require.Len(t, container.Env, 1)
		assert.Contains(t, container.Env[0].Value, testClusterID)
		assert.Contains(t, container.Env[0].Value, deploymentmetadata.DeploymentTypeApplicationMonitoring)
	})
}

func TestAddVersionDetectionEnvs(t *testing.T) {
	t.Run("adds defaults", func(t *testing.T) {
		container := &corev1.Container{}

		addVersionDetectionEnvs(container, defaultVersionLabelMapping)

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
				{Name: releaseVersionEnv, Value: testVersion},
				{Name: releaseProductEnv, Value: testProduct},
			},
		}

		addVersionDetectionEnvs(container, defaultVersionLabelMapping)

		require.Len(t, container.Env, 2)
		assert.Equal(t, testVersion, container.Env[0].Value)
		assert.Equal(t, testProduct, container.Env[1].Value)
	})

	t.Run("partial addition", func(t *testing.T) {
		testVersion := "1.2.3"
		container := &corev1.Container{
			Env: []corev1.EnvVar{
				{Name: releaseVersionEnv, Value: testVersion},
			},
		}

		addVersionDetectionEnvs(container, defaultVersionLabelMapping)

		require.Len(t, container.Env, 2)
		assert.Equal(t, testVersion, container.Env[0].Value)
		assert.Equal(t, defaultVersionLabelMapping[releaseProductEnv], container.Env[1].ValueFrom.FieldRef.FieldPath)
	})
}
