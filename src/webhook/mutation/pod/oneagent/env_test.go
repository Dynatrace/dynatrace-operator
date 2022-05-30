package oneagent_mutation

import (
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/oneagent/daemonset"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	t.Run("Add networkZone env", func(t *testing.T) {
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
		installerInfo := getTestInstallerInfo()
		addInstallerInitEnvs(container, installerInfo, testVolumeMode)
		require.Len(t, container.Env, 6)
		assert.Equal(t, installerInfo.flavor, container.Env[0].Value)
		assert.Equal(t, installerInfo.technologies, container.Env[1].Value)
		assert.Equal(t, installerInfo.installPath, container.Env[2].Value)
		assert.Equal(t, installerInfo.installerURL, container.Env[3].Value)
		assert.Equal(t, testVolumeMode, container.Env[4].Value)
		assert.Equal(t, "true", container.Env[5].Value)
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
		addDeploymentMetadataEnv(container, &dynakube, testClusterID)
		require.Len(t, container.Env, 1)
		assert.Contains(t, container.Env[0].Value, testClusterID)
		assert.Contains(t, container.Env[0].Value, daemonset.DeploymentTypeCloudNative)
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
		addDeploymentMetadataEnv(container, &dynakube, testClusterID)
		require.Len(t, container.Env, 1)
		assert.Contains(t, container.Env[0].Value, testClusterID)
		assert.Contains(t, container.Env[0].Value, daemonset.DeploymentTypeApplicationMonitoring)
	})
}

func TestInitialConnectRetryEnvIf(t *testing.T) {
	t.Run("Add initialConnectRetry env", func(t *testing.T) {
		container := &corev1.Container{}
		testValue := "42"
		dynakube := dynatracev1beta1.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					dynatracev1beta1.AnnotationFeatureOneAgentInitialConnectRetry: testValue,
				},
			},
		}
		addInitialConnectRetryEnv(container, &dynakube)
		require.Len(t, container.Env, 1)
		assert.Equal(t, container.Env[0].Value, testValue)
	})
}
