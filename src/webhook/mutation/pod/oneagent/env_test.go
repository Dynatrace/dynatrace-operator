package oneagent_mutation

import (
	"testing"

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
		assert.Equal(t, container.Env[0].Name, preloadEnvVarName)
		assert.Contains(t, container.Env[0].Value, installPath)
	})
}

func TestAddNetworkZoneEnv(t *testing.T) {
	t.Run("Add networkzone env", func(t *testing.T) {
		container := &corev1.Container{}
		networkZone := "testZone"

		addNetworkZoneEnv(container, networkZone)

		require.Len(t, container.Env, 1)
		assert.Equal(t, container.Env[0].Name, networkZoneEnvVarName)
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
		addInstallerInitEnvs(container, getTestInstallerInfo(), testVolumeMode)
		require.Len(t, container.Env, 6)
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
	t.Run("Add deployment metadata env", func(t *testing.T) {
		// TODO
	})
}


func TestInitialConnectRetryEnvIf(t *testing.T) {
	t.Run("Add initialConnectRetry env", func(t *testing.T) {
		// TODO
	})
}
