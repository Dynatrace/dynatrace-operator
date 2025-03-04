package oneagent

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
)

func TestAddInstallerInitEnvs(t *testing.T) {
	t.Run("Add installer init env", func(t *testing.T) {
		container := &corev1.Container{}
		installerInfo := getTestInstallerInfo()
		addInstallerInitEnvs(container, installerInfo)
		require.Len(t, container.Env, expectedBaseInitContainerEnvCount)
		assert.Equal(t, installerInfo.flavor, container.Env[0].Value)
		assert.Equal(t, installerInfo.technologies, container.Env[1].Value)
		assert.Equal(t, installerInfo.installPath, container.Env[2].Value)
		assert.Equal(t, installerInfo.installerURL, container.Env[3].Value)
		assert.Equal(t, installerInfo.version, container.Env[4].Value)
		assert.Equal(t, "true", container.Env[5].Value)
	})
}
