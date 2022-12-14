package daemonset

import (
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/connectioninfo"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/deploymentmetadata"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestEnvironmentVariables(t *testing.T) {
	t.Run("returns default values when members are nil", func(t *testing.T) {
		dsInfo := builderInfo{
			instance: &dynatracev1beta1.DynaKube{},
		}
		envVars := dsInfo.environmentVariables()

		assert.Contains(t, envVars, corev1.EnvVar{Name: dtClusterId, ValueFrom: nil})
		assert.True(t, kubeobjects.EnvVarIsIn(envVars, dtNodeName))
	})
	t.Run("returns all when everything is turned on", func(t *testing.T) {
		clusterID := "test"
		dynakube := &dynatracev1beta1.DynaKube{
			ObjectMeta: v1.ObjectMeta{
				Name: "test",
			},
			Spec: dynatracev1beta1.DynaKubeSpec{
				Proxy: &dynatracev1beta1.DynaKubeProxy{
					Value: "test",
				},
				OneAgent: dynatracev1beta1.OneAgentSpec{
					CloudNativeFullStack: &dynatracev1beta1.CloudNativeFullStackSpec{},
				},
			},
		}
		dsInfo := builderInfo{
			instance:  dynakube,
			clusterId: clusterID,
		}
		envVars := dsInfo.environmentVariables()

		assertClusterIDEnv(t, envVars, clusterID)
		assertNodeNameEnv(t, envVars)
		assertConnectionInfoEnv(t, envVars, dynakube)
		assertDeploymentMetadataEnv(t, envVars, dynakube.Name)
		assertProxyEnv(t, envVars, dynakube)
		assertReadOnlyEnv(t, envVars)
	})
}

func TestAddNodeNameEnv(t *testing.T) {
	t.Run("adds nodeName value from via fieldPath", func(t *testing.T) {
		envVars := addNodeNameEnv(map[string]corev1.EnvVar{})

		assertNodeNameEnv(t, mapToArray(envVars))
	})
}

func assertNodeNameEnv(t *testing.T, envs []corev1.EnvVar) {
	env := kubeobjects.FindEnvVar(envs, dtNodeName)
	assert.Equal(t, dtNodeName, env.Name)
	assert.Equal(t, "spec.nodeName", env.ValueFrom.FieldRef.FieldPath)
}

func TestAddClusterIDEnv(t *testing.T) {
	t.Run("adds clusterID value from struct", func(t *testing.T) {
		clusterID := "test"
		dsInfo := builderInfo{
			instance:  &dynatracev1beta1.DynaKube{},
			clusterId: clusterID,
		}
		envVars := dsInfo.addClusterIDEnv(map[string]corev1.EnvVar{})

		assertClusterIDEnv(t, mapToArray(envVars), clusterID)
	})
}

func assertClusterIDEnv(t *testing.T, envs []corev1.EnvVar, clusterID string) {
	env := kubeobjects.FindEnvVar(envs, dtClusterId)
	assert.Equal(t, dtClusterId, env.Name)
	assert.Equal(t, clusterID, env.Value)
}

func TestAddDeploymentMetadataEnv(t *testing.T) {
	t.Run("adds deployment metadata value via configmap ref", func(t *testing.T) {
		dynakubeName := "test"
		dsInfo := builderInfo{
			instance: &dynatracev1beta1.DynaKube{
				ObjectMeta: v1.ObjectMeta{
					Name: dynakubeName,
				},
			},
		}
		envVars := dsInfo.addDeploymentMetadataEnv(map[string]corev1.EnvVar{})

		assertDeploymentMetadataEnv(t, mapToArray(envVars), dynakubeName)
	})
}

func assertDeploymentMetadataEnv(t *testing.T, envs []corev1.EnvVar, dynakubeName string) {
	env := kubeobjects.FindEnvVar(envs, deploymentmetadata.EnvDtDeploymentMetadata)
	assert.Equal(t, env.Name, deploymentmetadata.EnvDtDeploymentMetadata)
	assert.Equal(t,
		deploymentmetadata.GetDeploymentMetadataConfigMapName(dynakubeName),
		env.ValueFrom.ConfigMapKeyRef.Name,
	)
	assert.Equal(t,
		deploymentmetadata.OneAgentMetadataKey,
		env.ValueFrom.ConfigMapKeyRef.Key,
	)
}

func TestAddConnectionInfoEnvs(t *testing.T) {
	t.Run("adds connection info value via configmap ref", func(t *testing.T) {
		dynakube := &dynatracev1beta1.DynaKube{
			ObjectMeta: v1.ObjectMeta{
				Name: "test",
			},
		}
		dsInfo := builderInfo{
			instance: dynakube,
		}
		envVars := dsInfo.addConnectionInfoEnvs(map[string]corev1.EnvVar{})

		assertConnectionInfoEnv(t, mapToArray(envVars), dynakube)
	})
}

func assertConnectionInfoEnv(t *testing.T, envs []corev1.EnvVar, dynakube *dynatracev1beta1.DynaKube) {
	env := kubeobjects.FindEnvVar(envs, connectioninfo.EnvDtTenant)
	assert.Equal(t, env.Name, connectioninfo.EnvDtTenant)
	assert.Equal(t,
		dynakube.OneAgentConnectionInfoConfigMapName(),
		env.ValueFrom.ConfigMapKeyRef.Name,
	)
	assert.Equal(t,
		connectioninfo.TenantUUIDName,
		env.ValueFrom.ConfigMapKeyRef.Key,
	)

	env = kubeobjects.FindEnvVar(envs, connectioninfo.EnvDtServer)
	assert.Equal(t, env.Name, connectioninfo.EnvDtServer)
	assert.Equal(t,
		dynakube.OneAgentConnectionInfoConfigMapName(),
		env.ValueFrom.ConfigMapKeyRef.Name,
	)
	assert.Equal(t,
		connectioninfo.CommunicationEndpointsName,
		env.ValueFrom.ConfigMapKeyRef.Key,
	)
}

func TestAddProxyEnvs(t *testing.T) {
	t.Run("adds proxy value from dynakube", func(t *testing.T) {
		dynakube := &dynatracev1beta1.DynaKube{
			ObjectMeta: v1.ObjectMeta{
				Name: "test",
			},
			Spec: dynatracev1beta1.DynaKubeSpec{
				Proxy: &dynatracev1beta1.DynaKubeProxy{
					Value: "test",
				},
			},
		}
		dsInfo := builderInfo{
			instance: dynakube,
		}
		envVars := dsInfo.addProxyEnv(map[string]corev1.EnvVar{})

		assertProxyEnv(t, mapToArray(envVars), dynakube)
	})

	t.Run("adds proxy value via secret ref from dynakube", func(t *testing.T) {
		dynakube := &dynatracev1beta1.DynaKube{
			ObjectMeta: v1.ObjectMeta{
				Name: "test",
			},
			Spec: dynatracev1beta1.DynaKubeSpec{
				Proxy: &dynatracev1beta1.DynaKubeProxy{
					ValueFrom: "test",
				},
			},
		}
		dsInfo := builderInfo{
			instance: dynakube,
		}
		envVars := dsInfo.addProxyEnv(map[string]corev1.EnvVar{})

		assertProxyEnv(t, mapToArray(envVars), dynakube)
	})
}

func assertProxyEnv(t *testing.T, envs []corev1.EnvVar, dynakube *dynatracev1beta1.DynaKube) {
	env := kubeobjects.FindEnvVar(envs, proxy)
	assert.Equal(t, env.Name, proxy)
	assert.Equal(t, dynakube.Spec.Proxy.Value, env.Value)
	if dynakube.Spec.Proxy.ValueFrom != "" {
		assert.Equal(t, dynakube.Spec.Proxy.ValueFrom, env.ValueFrom.SecretKeyRef.LocalObjectReference.Name)
		assert.Equal(t, "proxy", env.ValueFrom.SecretKeyRef.Key)
	}
}

func TestAddReadOnlyEnv(t *testing.T) {
	t.Run("adds readonly value for supported oneagent mode", func(t *testing.T) {
		dynakube := &dynatracev1beta1.DynaKube{
			ObjectMeta: v1.ObjectMeta{
				Name: "test",
			},
			Spec: dynatracev1beta1.DynaKubeSpec{
				OneAgent: dynatracev1beta1.OneAgentSpec{
					CloudNativeFullStack: &dynatracev1beta1.CloudNativeFullStackSpec{},
				},
			},
		}
		dsInfo := builderInfo{
			instance: dynakube,
		}
		envVars := dsInfo.addReadOnlyEnv(map[string]corev1.EnvVar{})

		assertReadOnlyEnv(t, mapToArray(envVars))
	})

	t.Run("not adds readonly value for supported oneagent mode", func(t *testing.T) {
		dynakube := &dynatracev1beta1.DynaKube{
			ObjectMeta: v1.ObjectMeta{
				Name: "test",
			},
		}
		dsInfo := builderInfo{
			instance: dynakube,
		}
		envVars := dsInfo.addReadOnlyEnv(map[string]corev1.EnvVar{})

		require.Empty(t, envVars)
	})
}

func assertReadOnlyEnv(t *testing.T, envs []corev1.EnvVar) {
	env := kubeobjects.FindEnvVar(envs, oneagentReadOnlyMode)
	assert.Equal(t, env.Name, oneagentReadOnlyMode)
	assert.Equal(t, "true", env.Value)
}
