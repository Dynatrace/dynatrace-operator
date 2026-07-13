package daemonset

import (
	"os"
	"strings"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/oneagent"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8senv"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8sresource"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

const (
	testKubernetesClusterName = "cluster-name"
	testKubernetesClusterUID  = "cluster-uid"
	testKubernetesClusterMEID = "cluster-meid"
	testOperatorImageName     = "operator-image-name"
)

func TestInitContainerSpec(t *testing.T) {
	dk := &dynakube.DynaKube{
		Status: dynakube.DynaKubeStatus{},
	}

	dsBuilder := builder{
		dk:             dk,
		hostInjectSpec: &oneagent.HostInjectSpec{},
	}

	require.NoError(t, os.Setenv(k8senv.DTOperatorImageEnvName, testOperatorImageName))
	spec := dsBuilder.initContainerSpec()
	require.NoError(t, os.Unsetenv(k8senv.DTOperatorImageEnvName))

	assert.Equal(t, testOperatorImageName, spec.Image)
	assert.Equal(t, initContainerName, spec.Name)
	assert.Empty(t, spec.ImagePullPolicy)
	assert.NotEmpty(t, spec.Env)
	assert.NotEmpty(t, spec.Args)
	assert.NotEmpty(t, spec.VolumeMounts)
	assert.NotNil(t, spec.SecurityContext)
	assert.NotNil(t, spec.Resources)
}

func TestInitContainerEnvVars(t *testing.T) {
	dsBuilder := builder{}

	envVars := dsBuilder.initContainerEnvVars()

	assert.Len(t, envVars, 1)
	assert.Equal(t, dtNodeName, envVars[0].Name)
	assert.Equal(t, &corev1.EnvVarSource{
		FieldRef: &corev1.ObjectFieldSelector{
			FieldPath: "spec.nodeName",
		},
	}, envVars[0].ValueFrom)
}

func TestInitContainerArguments(t *testing.T) {
	baseStatus := dynakube.DynaKubeStatus{
		KubeSystemUUID:        testKubernetesClusterUID,
		KubernetesClusterMEID: testKubernetesClusterMEID,
		KubernetesClusterName: testKubernetesClusterName,
	}

	t.Run("baseline args structure", func(t *testing.T) {
		dsBuilder := builder{dk: &dynakube.DynaKube{Status: baseStatus}}

		arguments := dsBuilder.initContainerArguments()
		assert.Equal(t, "generate-metadata", arguments[0])
		assert.Equal(t, "--file", arguments[1])
		assert.Equal(t, nodeMetadataFilePath, arguments[2])
		assert.Equal(t, "--attributes", arguments[3])

		attributes := strings.Split(arguments[4], ",")
		assert.Equal(t, "k8s.cluster.name="+testKubernetesClusterName, attributes[0])
		assert.Equal(t, "k8s.cluster.uid="+testKubernetesClusterUID, attributes[1])
		assert.Equal(t, "k8s.node.name=$(DT_K8S_NODE_NAME)", attributes[2])
		assert.Equal(t, "dt.entity.kubernetes_cluster="+testKubernetesClusterMEID, attributes[3])
	})

	t.Run("global resource attributes are sorted and appended", func(t *testing.T) {
		dk := &dynakube.DynaKube{
			Spec: dynakube.DynaKubeSpec{
				ResourceAttributes: map[string]string{"team": "platform", "env": "prod"},
				OneAgent:           oneagent.Spec{HostMonitoring: &oneagent.HostInjectSpec{}},
			},
			Status: baseStatus,
		}

		dsBuilder := builder{dk: dk}
		attributes := strings.Split(dsBuilder.initContainerArguments()[4], ",")

		assert.Equal(t, "k8s.cluster.name="+testKubernetesClusterName, attributes[0])
		assert.Equal(t, "k8s.cluster.uid="+testKubernetesClusterUID, attributes[1])
		assert.Equal(t, "k8s.node.name=$(DT_K8S_NODE_NAME)", attributes[2])
		assert.Equal(t, "dt.entity.kubernetes_cluster="+testKubernetesClusterMEID, attributes[3])
		assert.Equal(t, "env=prod", attributes[4])
		assert.Equal(t, "team=platform", attributes[5])
	})

	t.Run("hostMonitoring additionalResourceAttributes overrides global", func(t *testing.T) {
		dk := &dynakube.DynaKube{
			Spec: dynakube.DynaKubeSpec{
				ResourceAttributes: map[string]string{"shared": "global", "only-global": "g"},
				OneAgent: oneagent.Spec{HostMonitoring: &oneagent.HostInjectSpec{
					AdditionalResourceAttributes: map[string]string{"shared": "host", "only-host": "h"},
				}},
			},
			Status: baseStatus,
		}

		dsBuilder := builder{dk: dk}
		attributes := strings.Split(dsBuilder.initContainerArguments()[4], ",")
		assert.Contains(t, attributes, "shared=host")
		assert.Contains(t, attributes, "only-global=g")
		assert.Contains(t, attributes, "only-host=h")
		assert.NotContains(t, attributes, "shared=global")
	})

	t.Run("classicFullStack additionalResourceAttributes overrides global", func(t *testing.T) {
		dk := &dynakube.DynaKube{
			Spec: dynakube.DynaKubeSpec{
				ResourceAttributes: map[string]string{"env": "global"},
				OneAgent: oneagent.Spec{ClassicFullStack: &oneagent.HostInjectSpec{
					AdditionalResourceAttributes: map[string]string{"env": "classic"},
				}},
			},
			Status: baseStatus,
		}

		dsBuilder := builder{dk: dk}
		attributes := strings.Split(dsBuilder.initContainerArguments()[4], ",")
		assert.Contains(t, attributes, "env=classic")
		assert.NotContains(t, attributes, "env=global")
	})

	t.Run("cloudNativeFullStack additionalResourceAttributes overrides global", func(t *testing.T) {
		dk := &dynakube.DynaKube{
			Spec: dynakube.DynaKubeSpec{
				ResourceAttributes: map[string]string{"shared": "global"},
				OneAgent: oneagent.Spec{CloudNativeFullStack: &oneagent.CloudNativeFullStackSpec{
					HostInjectSpec: oneagent.HostInjectSpec{
						AdditionalResourceAttributes: map[string]string{"shared": "cnf", "extra": "val"},
					},
				}},
			},
			Status: baseStatus,
		}

		dsBuilder := builder{dk: dk}
		attributes := strings.Split(dsBuilder.initContainerArguments()[4], ",")
		assert.Contains(t, attributes, "shared=cnf")
		assert.Contains(t, attributes, "extra=val")
		assert.NotContains(t, attributes, "shared=global")
	})

	t.Run("no resource attributes leaves existing cluster metadata unchanged", func(t *testing.T) {
		dk := &dynakube.DynaKube{
			Spec:   dynakube.DynaKubeSpec{OneAgent: oneagent.Spec{HostMonitoring: &oneagent.HostInjectSpec{}}},
			Status: baseStatus,
		}

		dsBuilder := builder{dk: dk}
		attributes := strings.Split(dsBuilder.initContainerArguments()[4], ",")
		assert.Len(t, attributes, 4)
	})

	t.Run("sanitizes newline in value", func(t *testing.T) {
		dsBuilder := builder{dk: &dynakube.DynaKube{
			Status: dynakube.DynaKubeStatus{KubernetesClusterName: "cluster\nname"},
		}}
		attributes := strings.Split(dsBuilder.initContainerArguments()[4], ",")
		assert.Equal(t, "k8s.cluster.name=clustername", attributes[0])
	})

	t.Run("sanitizes tab in value", func(t *testing.T) {
		dsBuilder := builder{dk: &dynakube.DynaKube{
			Status: dynakube.DynaKubeStatus{KubeSystemUUID: "uid\t123"},
		}}
		attributes := strings.Split(dsBuilder.initContainerArguments()[4], ",")
		assert.Equal(t, "k8s.cluster.uid=uid123", attributes[1])
	})

	t.Run("sanitizes carriage return in value", func(t *testing.T) {
		dsBuilder := builder{dk: &dynakube.DynaKube{
			Status: dynakube.DynaKubeStatus{
				KubernetesClusterName: testKubernetesClusterName,
				KubeSystemUUID:        testKubernetesClusterUID,
				KubernetesClusterMEID: "meid\r123",
			},
		}}
		attributes := strings.Split(dsBuilder.initContainerArguments()[4], ",")
		assert.Equal(t, "dt.entity.kubernetes_cluster=meid123", attributes[3])
	})

	t.Run("sanitizes null byte in value", func(t *testing.T) {
		dk := &dynakube.DynaKube{
			Spec: dynakube.DynaKubeSpec{
				ResourceAttributes: map[string]string{"env": "prod\x00uction"},
				OneAgent:           oneagent.Spec{HostMonitoring: &oneagent.HostInjectSpec{}},
			},
			Status: baseStatus,
		}
		dsBuilder := builder{dk: dk}
		attributes := strings.Split(dsBuilder.initContainerArguments()[4], ",")
		assert.Contains(t, attributes, "env=production")
	})

	t.Run("sanitizes newline in key", func(t *testing.T) {
		dk := &dynakube.DynaKube{
			Spec: dynakube.DynaKubeSpec{
				ResourceAttributes: map[string]string{"ke\ny": "value"},
				OneAgent:           oneagent.Spec{HostMonitoring: &oneagent.HostInjectSpec{}},
			},
			Status: baseStatus,
		}
		dsBuilder := builder{dk: dk}
		attributes := strings.Split(dsBuilder.initContainerArguments()[4], ",")
		assert.Contains(t, attributes, "key=value")
	})
}

func TestInitContainerResources(t *testing.T) {
	t.Run("returns defaults when hostInjectSpec has no initResources, no limits", func(t *testing.T) {
		dsBuilder := builder{
			hostInjectSpec: &oneagent.HostInjectSpec{},
		}
		resources := dsBuilder.initContainerResources()

		assert.Equal(t, k8sresource.NewResourceList("20m", "20Mi"), resources.Requests)
		assert.Empty(t, resources.Limits)
	})

	t.Run("returns custom resources when OneAgentInitResources is set", func(t *testing.T) {
		cpuRequest := resource.MustParse("100m")
		memRequest := resource.MustParse("64Mi")
		cpuLimit := resource.MustParse("200m")
		memLimit := resource.MustParse("128Mi")

		dsBuilder := builder{
			hostInjectSpec: &oneagent.HostInjectSpec{
				OneAgentInitResources: &corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceCPU:    cpuRequest,
						corev1.ResourceMemory: memRequest,
					},
					Limits: corev1.ResourceList{
						corev1.ResourceCPU:    cpuLimit,
						corev1.ResourceMemory: memLimit,
					},
				},
			},
		}
		resources := dsBuilder.initContainerResources()

		assert.True(t, cpuRequest.Equal(*resources.Requests.Cpu()))
		assert.True(t, memRequest.Equal(*resources.Requests.Memory()))
		assert.True(t, cpuLimit.Equal(*resources.Limits.Cpu()))
		assert.True(t, memLimit.Equal(*resources.Limits.Memory()))
	})
}

func TestInitContainerVolumeMounts(t *testing.T) {
	dsBuilder := builder{}

	volumeMounts := dsBuilder.initContainerVolumeMounts()

	assert.Len(t, volumeMounts, 1)
	assert.Contains(t, volumeMounts, corev1.VolumeMount{
		Name:      nodeMetadataVolumeName,
		MountPath: nodeMetadataFolderPath,
		ReadOnly:  false,
	})
}

func TestInitContainerSecurityContext(t *testing.T) {
	dsBuilder := builder{}

	securityContext := dsBuilder.initContainerSecurityContext()

	assert.False(t, *securityContext.Privileged)
	assert.False(t, *securityContext.AllowPrivilegeEscalation)
	assert.True(t, *securityContext.RunAsNonRoot)
	assert.Equal(t, userGroupID, *securityContext.RunAsUser)
	assert.Equal(t, userGroupID, *securityContext.RunAsGroup)
	assert.Empty(t, securityContext.Capabilities.Add)
	assert.Len(t, securityContext.Capabilities.Drop, 1)
	assert.Contains(t, securityContext.Capabilities.Drop, corev1.Capability("ALL"))
	assert.Equal(t, corev1.SeccompProfileTypeRuntimeDefault, securityContext.SeccompProfile.Type)
	assert.True(t, *securityContext.ReadOnlyRootFilesystem)
}
