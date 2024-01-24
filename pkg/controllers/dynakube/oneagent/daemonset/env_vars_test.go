package daemonset

import (
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/connectioninfo"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/deploymentmetadata"
	k8senv "github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/env"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/prioritymap"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestEnvironmentVariables(t *testing.T) {
	t.Run("returns default values when members are nil", func(t *testing.T) {
		dsInfo := builderInfo{
			dynakube: &dynatracev1beta1.DynaKube{},
		}
		envVars, _ := dsInfo.environmentVariables()

		assert.Contains(t, envVars, corev1.EnvVar{Name: dtClusterId, ValueFrom: nil})
		assert.True(t, k8senv.IsIn(envVars, dtNodeName))
	})
	t.Run("returns all when everything is turned on", func(t *testing.T) {
		clusterID := "test"
		dynakube := &dynatracev1beta1.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
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
			dynakube:  dynakube,
			clusterID: clusterID,
		}
		envVars, _ := dsInfo.environmentVariables()

		assertClusterIDEnv(t, envVars, clusterID)
		assertNodeNameEnv(t, envVars)
		assertConnectionInfoEnv(t, envVars, dynakube)
		assertDeploymentMetadataEnv(t, envVars, dynakube.Name)
		// deprecated
		assertProxyEnv(t, envVars, dynakube)
		assertReadOnlyEnv(t, envVars)
	})
	t.Run("when injected envvars are provided then they will not be overridden", func(t *testing.T) {
		potentiallyOverriddenEnvVars := []corev1.EnvVar{
			{Name: dtNodeName, Value: testValue},
			{Name: dtClusterId, Value: testValue},
			{Name: deploymentmetadata.EnvDtDeploymentMetadata, Value: testValue},
			{Name: deploymentmetadata.EnvDtOperatorVersion, Value: testValue},
			{Name: connectioninfo.EnvDtTenant, Value: testValue},
			{Name: proxyEnv, Value: testValue},
			{Name: oneagentReadOnlyMode, Value: testValue},
		}
		builder := builderInfo{
			dynakube:       &dynatracev1beta1.DynaKube{},
			hostInjectSpec: &dynatracev1beta1.HostInjectSpec{Env: potentiallyOverriddenEnvVars},
		}
		envVars, _ := builder.environmentVariables()

		assertEnvVarNameAndValue(t, envVars, dtNodeName, testValue)
		assertEnvVarNameAndValue(t, envVars, dtClusterId, testValue)
		assertEnvVarNameAndValue(t, envVars, deploymentmetadata.EnvDtDeploymentMetadata, testValue)
		assertEnvVarNameAndValue(t, envVars, deploymentmetadata.EnvDtOperatorVersion, testValue)
		assertEnvVarNameAndValue(t, envVars, connectioninfo.EnvDtTenant, testValue)
		assertEnvVarNameAndValue(t, envVars, proxyEnv, testValue)
		assertEnvVarNameAndValue(t, envVars, oneagentReadOnlyMode, testValue)
	})
}

func assertEnvVarNameAndValue(t *testing.T, envVars []corev1.EnvVar, name, value string) {
	env := k8senv.FindEnvVar(envVars, name)
	assert.Equal(t, name, env.Name)
	assert.Equal(t, value, env.Value)
}

func TestAddNodeNameEnv(t *testing.T) {
	t.Run("adds nodeName value from via fieldPath", func(t *testing.T) {
		envVars := prioritymap.New()
		addNodeNameEnv(envVars)

		assertNodeNameEnv(t, envVars.AsEnvVars())
	})
}

func assertNodeNameEnv(t *testing.T, envs []corev1.EnvVar) {
	env := k8senv.FindEnvVar(envs, dtNodeName)
	assert.Equal(t, dtNodeName, env.Name)
	assert.Equal(t, "spec.nodeName", env.ValueFrom.FieldRef.FieldPath)
}

func TestAddClusterIDEnv(t *testing.T) {
	t.Run("adds clusterID value from struct", func(t *testing.T) {
		clusterID := "test"
		dsInfo := builderInfo{
			dynakube:  &dynatracev1beta1.DynaKube{},
			clusterID: clusterID,
		}
		envVars := prioritymap.New()
		dsInfo.addClusterIDEnv(envVars)

		assertClusterIDEnv(t, envVars.AsEnvVars(), clusterID)
	})
}

func assertClusterIDEnv(t *testing.T, envs []corev1.EnvVar, clusterID string) {
	env := k8senv.FindEnvVar(envs, dtClusterId)
	assert.Equal(t, dtClusterId, env.Name)
	assert.Equal(t, clusterID, env.Value)
}

func TestAddDeploymentMetadataEnv(t *testing.T) {
	t.Run("adds deployment metadata value via configmap ref", func(t *testing.T) {
		dynakubeName := "test"
		dsInfo := builderInfo{
			dynakube: &dynatracev1beta1.DynaKube{
				ObjectMeta: metav1.ObjectMeta{
					Name: dynakubeName,
				},
			},
		}
		envVars := prioritymap.New()
		dsInfo.addDeploymentMetadataEnv(envVars)

		assertDeploymentMetadataEnv(t, envVars.AsEnvVars(), dynakubeName)
	})
}

func assertDeploymentMetadataEnv(t *testing.T, envs []corev1.EnvVar, dynakubeName string) {
	env := k8senv.FindEnvVar(envs, deploymentmetadata.EnvDtDeploymentMetadata)
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
			ObjectMeta: metav1.ObjectMeta{
				Name: "test",
			},
		}
		dsInfo := builderInfo{
			dynakube: dynakube,
		}
		envVars := prioritymap.New()
		dsInfo.addConnectionInfoEnvs(envVars)

		assertConnectionInfoEnv(t, envVars.AsEnvVars(), dynakube)
	})
}

func assertConnectionInfoEnv(t *testing.T, envs []corev1.EnvVar, dynakube *dynatracev1beta1.DynaKube) {
	env := k8senv.FindEnvVar(envs, connectioninfo.EnvDtTenant)
	assert.Equal(t, env.Name, connectioninfo.EnvDtTenant)
	assert.Equal(t,
		dynakube.OneAgentConnectionInfoConfigMapName(),
		env.ValueFrom.ConfigMapKeyRef.Name,
	)
	assert.Equal(t,
		connectioninfo.TenantUUIDName,
		env.ValueFrom.ConfigMapKeyRef.Key,
	)

	env = k8senv.FindEnvVar(envs, connectioninfo.EnvDtServer)
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

// deprecated
func TestAddProxyEnvs(t *testing.T) {
	t.Run("adds proxy value from dynakube", func(t *testing.T) {
		dynakube := &dynatracev1beta1.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test",
			},
			Spec: dynatracev1beta1.DynaKubeSpec{
				Proxy: &dynatracev1beta1.DynaKubeProxy{
					Value: "test",
				},
			},
		}
		dsInfo := builderInfo{
			dynakube: dynakube,
		}
		envVars := prioritymap.New()
		dsInfo.addProxyEnv(envVars)

		assertProxyEnv(t, envVars.AsEnvVars(), dynakube)
	})

	t.Run("adds proxy value via secret ref from dynakube", func(t *testing.T) {
		dynakube := &dynatracev1beta1.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test",
			},
			Spec: dynatracev1beta1.DynaKubeSpec{
				Proxy: &dynatracev1beta1.DynaKubeProxy{
					ValueFrom: "test",
				},
			},
		}
		dsInfo := builderInfo{
			dynakube: dynakube,
		}
		envVars := prioritymap.New()
		dsInfo.addProxyEnv(envVars)

		assertProxyEnv(t, envVars.AsEnvVars(), dynakube)
	})
}

// deprecated
func assertProxyEnv(t *testing.T, envs []corev1.EnvVar, dynakube *dynatracev1beta1.DynaKube) {
	env := k8senv.FindEnvVar(envs, proxyEnv)
	assert.Equal(t, env.Name, proxyEnv)
	assert.Equal(t, dynakube.Spec.Proxy.Value, env.Value)
	if dynakube.Spec.Proxy.ValueFrom != "" {
		assert.Equal(t, dynakube.Spec.Proxy.ValueFrom, env.ValueFrom.SecretKeyRef.LocalObjectReference.Name)
		assert.Equal(t, "proxy", env.ValueFrom.SecretKeyRef.Key)
	}
}

func TestAddReadOnlyEnv(t *testing.T) {
	t.Run("adds readonly value for supported oneagent mode", func(t *testing.T) {
		dynakube := &dynatracev1beta1.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test",
			},
			Spec: dynatracev1beta1.DynaKubeSpec{
				OneAgent: dynatracev1beta1.OneAgentSpec{
					CloudNativeFullStack: &dynatracev1beta1.CloudNativeFullStackSpec{},
				},
			},
		}
		dsInfo := builderInfo{
			dynakube: dynakube,
		}
		envVars := prioritymap.New()
		dsInfo.addReadOnlyEnv(envVars)

		assertReadOnlyEnv(t, envVars.AsEnvVars())
	})

	t.Run("not adds readonly value for supported oneagent mode", func(t *testing.T) {
		dynakube := &dynatracev1beta1.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test",
			},
		}
		dsInfo := builderInfo{
			dynakube: dynakube,
		}
		envVars := prioritymap.New()
		dsInfo.addReadOnlyEnv(envVars)

		require.Empty(t, envVars.AsEnvVars())
	})
}

func assertReadOnlyEnv(t *testing.T, envs []corev1.EnvVar) {
	env := k8senv.FindEnvVar(envs, oneagentReadOnlyMode)
	assert.Equal(t, env.Name, oneagentReadOnlyMode)
	assert.Equal(t, "true", env.Value)
}
