package daemonset

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/value"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
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
		dsBuilder := builder{
			dk: &dynakube.DynaKube{},
		}
		envVars, _ := dsBuilder.environmentVariables()

		assert.Contains(t, envVars, corev1.EnvVar{Name: dtClusterId, ValueFrom: nil})
		assert.True(t, k8senv.IsIn(envVars, dtNodeName))
	})
	t.Run("returns all when everything is turned on", func(t *testing.T) {
		clusterID := "test"
		dk := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test",
			},
			Spec: dynakube.DynaKubeSpec{
				Proxy: &value.Source{
					Value: "test",
				},
				OneAgent: dynakube.OneAgentSpec{
					CloudNativeFullStack: &dynakube.CloudNativeFullStackSpec{},
				},
			},
		}
		dsBuilder := builder{
			dk:        dk,
			clusterID: clusterID,
		}
		envVars, _ := dsBuilder.environmentVariables()

		assertClusterIDEnv(t, envVars, clusterID)
		assertNodeNameEnv(t, envVars)
		assertConnectionInfoEnv(t, envVars, dk)
		assertDeploymentMetadataEnv(t, envVars, dk.Name)

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
		builder := builder{
			dk:             &dynakube.DynaKube{},
			hostInjectSpec: &dynakube.HostInjectSpec{Env: potentiallyOverriddenEnvVars},
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
		dsBuilder := builder{
			dk:        &dynakube.DynaKube{},
			clusterID: clusterID,
		}
		envVars := prioritymap.New()
		dsBuilder.addClusterIDEnv(envVars)

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
		dsBuilder := builder{
			dk: &dynakube.DynaKube{
				ObjectMeta: metav1.ObjectMeta{
					Name: dynakubeName,
				},
			},
		}
		envVars := prioritymap.New()
		dsBuilder.addDeploymentMetadataEnv(envVars)

		assertDeploymentMetadataEnv(t, envVars.AsEnvVars(), dynakubeName)
	})
}

func assertDeploymentMetadataEnv(t *testing.T, envs []corev1.EnvVar, dynakubeName string) {
	env := k8senv.FindEnvVar(envs, deploymentmetadata.EnvDtDeploymentMetadata)
	assert.Equal(t, deploymentmetadata.EnvDtDeploymentMetadata, env.Name)
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
		dk := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test",
			},
		}
		dsBuilder := builder{
			dk: dk,
		}
		envVars := prioritymap.New()
		dsBuilder.addConnectionInfoEnvs(envVars)

		assertConnectionInfoEnv(t, envVars.AsEnvVars(), dk)
	})
}

func assertConnectionInfoEnv(t *testing.T, envs []corev1.EnvVar, dk *dynakube.DynaKube) {
	env := k8senv.FindEnvVar(envs, connectioninfo.EnvDtTenant)
	assert.Equal(t, connectioninfo.EnvDtTenant, env.Name)
	assert.Equal(t,
		dk.OneAgentConnectionInfoConfigMapName(),
		env.ValueFrom.ConfigMapKeyRef.Name,
	)
	assert.Equal(t,
		connectioninfo.TenantUUIDKey,
		env.ValueFrom.ConfigMapKeyRef.Key,
	)

	env = k8senv.FindEnvVar(envs, connectioninfo.EnvDtServer)
	assert.Equal(t, connectioninfo.EnvDtServer, env.Name)
	assert.Equal(t,
		dk.OneAgentConnectionInfoConfigMapName(),
		env.ValueFrom.ConfigMapKeyRef.Name,
	)
	assert.Equal(t,
		connectioninfo.CommunicationEndpointsKey,
		env.ValueFrom.ConfigMapKeyRef.Key,
	)
}

// deprecated
func TestAddProxyEnvs(t *testing.T) {
	t.Run("adds proxy value from dynakube", func(t *testing.T) {
		dk := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test",
			},
			Spec: dynakube.DynaKubeSpec{
				Proxy: &value.Source{
					Value: "test",
				},
			},
		}
		dsBuilder := builder{
			dk: dk,
		}
		envVars := prioritymap.New()
		dsBuilder.addProxyEnv(envVars)

		assertProxyEnv(t, envVars.AsEnvVars(), dk)
	})

	t.Run("adds proxy value via secret ref from dynakube", func(t *testing.T) {
		dk := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test",
			},
			Spec: dynakube.DynaKubeSpec{
				Proxy: &value.Source{
					ValueFrom: "test",
				},
			},
		}
		dsBuilder := builder{
			dk: dk,
		}
		envVars := prioritymap.New()
		dsBuilder.addProxyEnv(envVars)

		assertProxyEnv(t, envVars.AsEnvVars(), dk)
	})
}

// deprecated
func assertProxyEnv(t *testing.T, envs []corev1.EnvVar, dk *dynakube.DynaKube) {
	env := k8senv.FindEnvVar(envs, proxyEnv)
	assert.Equal(t, proxyEnv, env.Name)
	assert.Equal(t, dk.Spec.Proxy.Value, env.Value)

	if dk.Spec.Proxy.ValueFrom != "" {
		assert.Equal(t, dk.Spec.Proxy.ValueFrom, env.ValueFrom.SecretKeyRef.LocalObjectReference.Name)
		assert.Equal(t, "proxy", env.ValueFrom.SecretKeyRef.Key)
	}
}

func TestAddReadOnlyEnv(t *testing.T) {
	t.Run("adds readonly value for supported oneagent mode", func(t *testing.T) {
		dk := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test",
			},
			Spec: dynakube.DynaKubeSpec{
				OneAgent: dynakube.OneAgentSpec{
					CloudNativeFullStack: &dynakube.CloudNativeFullStackSpec{},
				},
			},
		}
		dsBuilder := builder{
			dk: dk,
		}
		envVars := prioritymap.New()
		dsBuilder.addReadOnlyEnv(envVars)

		assertReadOnlyEnv(t, envVars.AsEnvVars())
	})

	t.Run("not adds readonly value for supported oneagent mode", func(t *testing.T) {
		dk := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test",
			},
		}
		dsBuilder := builder{
			dk: dk,
		}
		envVars := prioritymap.New()
		dsBuilder.addReadOnlyEnv(envVars)

		require.Empty(t, envVars.AsEnvVars())
	})
}

func assertReadOnlyEnv(t *testing.T, envs []corev1.EnvVar) {
	env := k8senv.FindEnvVar(envs, oneagentReadOnlyMode)
	assert.Equal(t, oneagentReadOnlyMode, env.Name)
	assert.Equal(t, "true", env.Value)
}

func TestIsProxyAsEnvVarDeprecated(t *testing.T) {
	tests := []struct {
		name            string
		oneAgentVersion string
		want            bool
		wantErr         bool
	}{
		{
			name:            "empty version",
			oneAgentVersion: "",
			want:            true,
			wantErr:         false,
		},
		{
			name:            "wrong version format",
			oneAgentVersion: "1.2",
			want:            false,
			wantErr:         true,
		},
		{
			name:            "older version",
			oneAgentVersion: "1.261.2.20220212-223432",
			want:            false,
			wantErr:         false,
		},
		{
			name:            "newer version",
			oneAgentVersion: "1.285.0.20240122-141707",
			want:            true,
			wantErr:         false,
		},
		{
			name:            "custom-image -> hard-coded version placeholder",
			oneAgentVersion: string(status.CustomImageVersionSource),
			want:            true,
			wantErr:         false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := isProxyAsEnvVarDeprecated(tt.oneAgentVersion)
			if (err != nil) != tt.wantErr {
				t.Errorf("isProxyAsEnvVarDeprecated() error = %v, wantErr %v", err, tt.wantErr)

				return
			}

			if got != tt.want {
				t.Errorf("isProxyAsEnvVarDeprecated() = %v, want %v", got, tt.want)
			}
		})
	}
}
