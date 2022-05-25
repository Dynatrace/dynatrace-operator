package oneagent_mutation

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestMutateConfigureInitContainer(t *testing.T) {
	t.Run("add envs and volume mounts (no-csi)", func(t *testing.T) {
		mutator := createTestPodMutator([]client.Object{getTestInitSecret()})
		request := createTestMutationRequest(getTestDynakube(), nil)
		installerInfo := getTestInstallerInfo()

		mutator.configureInitContainer(request, installerInfo)

		require.Equal(t, 6, len(request.InstallContainer.Env))
		assert.Equal(t, installerVolumeMode, request.InstallContainer.Env[4].Value)
		assert.Equal(t, 2, len(request.InstallContainer.VolumeMounts))
	})

	t.Run("add envs and volume mounts (csi)", func(t *testing.T) {
		mutator := createTestPodMutator([]client.Object{getTestInitSecret()})
		request := createTestMutationRequest(getTestCSIDynakube(), nil)
		installerInfo := getTestInstallerInfo()

		mutator.configureInitContainer(request, installerInfo)

		require.Equal(t, 6, len(request.InstallContainer.Env))
		assert.Equal(t, provisionedVolumeMode, request.InstallContainer.Env[4].Value)
		assert.Equal(t, 2, len(request.InstallContainer.VolumeMounts))
	})
}
