package daemonset

import (
	"strings"
	"testing"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/exp"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/oneagent"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/deploymentmetadata"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/labels"
	"github.com/Dynatrace/dynatrace-operator/pkg/version"
	webhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/common"
	containerv1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

const (
	testImageTag  = "1.203.0.0-0"
	testTokenHash = "test-token-hash"
)

func TestUseImmutableImage(t *testing.T) {
	t.Run(`use image from status`, func(t *testing.T) {
		imageID := "my.repo.com/image:my-tag"
		dk := dynakube.DynaKube{
			Spec: dynakube.DynaKubeSpec{
				APIURL: testURL,
				OneAgent: oneagent.Spec{
					ClassicFullStack: &oneagent.HostInjectSpec{},
				},
			},
			Status: dynakube.DynaKubeStatus{
				OneAgent: oneagent.Status{
					VersionStatus: status.VersionStatus{
						ImageID: imageID,
					},
				},
			},
		}
		dsBuilder := NewClassicFullStack(&dk, testClusterID)
		ds, err := dsBuilder.BuildDaemonSet()
		require.NoError(t, err)

		podSpecs := ds.Spec.Template.Spec
		assert.NotNil(t, podSpecs)
		assert.Equal(t, imageID, podSpecs.Containers[0].Image)
	})
}

func TestLabels(t *testing.T) {
	feature := strings.ReplaceAll(deploymentmetadata.ClassicFullStackDeploymentType, "_", "")

	t.Run("use version when set", func(t *testing.T) {
		dk := dynakube.DynaKube{
			Spec: dynakube.DynaKubeSpec{
				APIURL: testURL,
				OneAgent: oneagent.Spec{
					ClassicFullStack: &oneagent.HostInjectSpec{},
				},
			},
			Status: dynakube.DynaKubeStatus{
				OneAgent: oneagent.Status{
					VersionStatus: status.VersionStatus{
						Version: testImageTag,
					},
				},
			},
		}
		expectedLabels := map[string]string{
			labels.AppNameLabel:      labels.OneAgentComponentLabel,
			labels.AppCreatedByLabel: dk.Name,
			labels.AppComponentLabel: feature,
			labels.AppVersionLabel:   testImageTag,
			labels.AppManagedByLabel: version.AppName,
		}
		expectedMatchLabels := map[string]string{
			labels.AppNameLabel:      labels.OneAgentComponentLabel,
			labels.AppCreatedByLabel: dk.Name,
			labels.AppManagedByLabel: version.AppName,
		}
		dsBuilder := NewClassicFullStack(&dk, testClusterID)
		ds, err := dsBuilder.BuildDaemonSet()
		require.NoError(t, err)

		podSpecs := ds.Spec.Template.Spec
		assert.NotNil(t, podSpecs)
		assert.Equal(t, expectedLabels, ds.Labels)
		assert.Equal(t, expectedMatchLabels, ds.Spec.Selector.MatchLabels)
		assert.Equal(t, expectedLabels, ds.Spec.Template.Labels)
	})
	t.Run("if no version is set, no version label", func(t *testing.T) {
		dk := dynakube.DynaKube{
			Spec: dynakube.DynaKubeSpec{
				APIURL: testURL,
				OneAgent: oneagent.Spec{
					ClassicFullStack: &oneagent.HostInjectSpec{},
				},
			},
		}

		expectedLabels := map[string]string{
			labels.AppNameLabel:      labels.OneAgentComponentLabel,
			labels.AppCreatedByLabel: dk.Name,
			labels.AppVersionLabel:   "",
			labels.AppComponentLabel: feature,
			labels.AppManagedByLabel: version.AppName,
		}
		expectedMatchLabels := map[string]string{
			labels.AppNameLabel:      labels.OneAgentComponentLabel,
			labels.AppCreatedByLabel: dk.Name,
			labels.AppManagedByLabel: version.AppName,
		}

		dsBuilder := NewClassicFullStack(&dk, testClusterID)
		ds, err := dsBuilder.BuildDaemonSet()
		require.NoError(t, err)

		podSpecs := ds.Spec.Template.Spec
		assert.NotNil(t, podSpecs)
		assert.Equal(t, expectedLabels, ds.Labels)
		assert.Equal(t, expectedMatchLabels, ds.Spec.Selector.MatchLabels)
		assert.Equal(t, expectedLabels, ds.Spec.Template.Labels)
	})
}

func TestCustomPullSecret(t *testing.T) {
	dk := dynakube.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name: testDynakubeName,
		},
		Spec: dynakube.DynaKubeSpec{
			APIURL: testURL,
			OneAgent: oneagent.Spec{
				ClassicFullStack: &oneagent.HostInjectSpec{},
			},
			CustomPullSecret: testName,
		},
	}
	dsBuilder := NewClassicFullStack(&dk, testClusterID)
	ds, err := dsBuilder.BuildDaemonSet()
	require.NoError(t, err)

	podSpecs := ds.Spec.Template.Spec
	assert.NotNil(t, podSpecs)
	assert.Len(t, podSpecs.ImagePullSecrets, 2)
	assert.Equal(t, testDynakubeName+dynakube.PullSecretSuffix, podSpecs.ImagePullSecrets[0].Name)
	assert.Equal(t, testName, podSpecs.ImagePullSecrets[1].Name)
}

func TestResources(t *testing.T) {
	t.Run(`minimal cpu request of 100mC is set if no resources specified`, func(t *testing.T) {
		dk := dynakube.DynaKube{
			Spec: dynakube.DynaKubeSpec{
				APIURL: testURL,
				OneAgent: oneagent.Spec{
					ClassicFullStack: &oneagent.HostInjectSpec{},
				},
			},
		}
		dsBuilder := NewClassicFullStack(&dk, testClusterID)
		ds, err := dsBuilder.BuildDaemonSet()
		require.NoError(t, err)

		podSpecs := ds.Spec.Template.Spec
		assert.NotNil(t, podSpecs)
		assert.NotEmpty(t, podSpecs.Containers)

		hasMinimumCPURequest := resource.NewScaledQuantity(1, -1).Equal(*podSpecs.Containers[0].Resources.Requests.Cpu())
		assert.True(t, hasMinimumCPURequest)
	})
	t.Run(`resource requests and limits set`, func(t *testing.T) {
		cpuRequest := resource.NewScaledQuantity(2, -1)
		cpuLimit := resource.NewScaledQuantity(3, -1)
		memoryRequest := resource.NewScaledQuantity(1, 3)
		memoryLimit := resource.NewScaledQuantity(2, 3)

		dk := dynakube.DynaKube{
			Spec: dynakube.DynaKubeSpec{
				APIURL: testURL,
				OneAgent: oneagent.Spec{
					ClassicFullStack: &oneagent.HostInjectSpec{
						OneAgentResources: corev1.ResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceCPU:    *cpuRequest,
								corev1.ResourceMemory: *memoryRequest,
							},
							Limits: corev1.ResourceList{
								corev1.ResourceCPU:    *cpuLimit,
								corev1.ResourceMemory: *memoryLimit,
							},
						},
					},
				},
			},
		}

		dsBuilder := NewClassicFullStack(&dk, testClusterID)
		ds, err := dsBuilder.BuildDaemonSet()
		require.NoError(t, err)

		podSpecs := ds.Spec.Template.Spec
		assert.NotNil(t, podSpecs)
		assert.NotEmpty(t, podSpecs.Containers)

		hasCPURequest := cpuRequest.Equal(*podSpecs.Containers[0].Resources.Requests.Cpu())
		hasCPULimit := cpuLimit.Equal(*podSpecs.Containers[0].Resources.Limits.Cpu())
		hasMemoryRequest := memoryRequest.Equal(*podSpecs.Containers[0].Resources.Requests.Memory())
		hasMemoryLimit := memoryLimit.Equal(*podSpecs.Containers[0].Resources.Limits.Memory())

		assert.True(t, hasCPURequest)
		assert.True(t, hasCPULimit)
		assert.True(t, hasMemoryRequest)
		assert.True(t, hasMemoryLimit)
	})
}

func TestSecurityContextCapabilities(t *testing.T) {
	securityContextCapabilities := defaultSecurityContextCapabilities()

	assert.NotNil(t, securityContextCapabilities)

	dropCapabilities := securityContextCapabilities.Drop
	assert.Contains(t, dropCapabilities, corev1.Capability("ALL"))

	addCapabilities := securityContextCapabilities.Add
	assert.Contains(t, addCapabilities, corev1.Capability("CHOWN"))
	assert.Contains(t, addCapabilities, corev1.Capability("DAC_OVERRIDE"))
	assert.Contains(t, addCapabilities, corev1.Capability("DAC_READ_SEARCH"))
	assert.Contains(t, addCapabilities, corev1.Capability("FOWNER"))
	assert.Contains(t, addCapabilities, corev1.Capability("FSETID"))
	assert.Contains(t, addCapabilities, corev1.Capability("KILL"))
	assert.Contains(t, addCapabilities, corev1.Capability("NET_ADMIN"))
	assert.Contains(t, addCapabilities, corev1.Capability("NET_RAW"))
	assert.Contains(t, addCapabilities, corev1.Capability("SETFCAP"))
	assert.Contains(t, addCapabilities, corev1.Capability("SETGID"))
	assert.Contains(t, addCapabilities, corev1.Capability("SETUID"))
	assert.Contains(t, addCapabilities, corev1.Capability("SYS_ADMIN"))
	assert.Contains(t, addCapabilities, corev1.Capability("SYS_CHROOT"))
	assert.Contains(t, addCapabilities, corev1.Capability("SYS_PTRACE"))
	assert.Contains(t, addCapabilities, corev1.Capability("SYS_RESOURCE"))
}

func TestHostMonitoring_SecurityContext(t *testing.T) {
	t.Run("returns default context if instance is nil", func(t *testing.T) {
		dsBuilder := builder{}
		securityContext := dsBuilder.securityContext()

		assert.Equal(t, defaultSecurityContextCapabilities(), securityContext.Capabilities)
	})
	t.Run(`User and group id set when read only mode is enabled`, func(t *testing.T) {
		dk := dynakube.DynaKube{
			Spec: dynakube.DynaKubeSpec{
				APIURL: testURL,
				OneAgent: oneagent.Spec{
					HostMonitoring: &oneagent.HostInjectSpec{},
				},
			},
		}
		dsBuilder := NewHostMonitoring(&dk, testClusterID)
		ds, err := dsBuilder.BuildDaemonSet()
		require.NoError(t, err)

		assert.GreaterOrEqual(t, 1, len(ds.Spec.Template.Spec.Containers))

		securityContext := ds.Spec.Template.Spec.Containers[0].SecurityContext

		assert.NotNil(t, securityContext)
		assert.Equal(t, ptr.To(int64(1000)), securityContext.RunAsUser)
		assert.Equal(t, ptr.To(int64(1000)), securityContext.RunAsGroup)
		assert.Equal(t, ptr.To(true), securityContext.RunAsNonRoot)
		assert.NotEmpty(t, securityContext.Capabilities)
		assert.Nil(t, securityContext.SeccompProfile)
		require.NotNil(t, securityContext.ReadOnlyRootFilesystem)
		assert.True(t, *securityContext.ReadOnlyRootFilesystem)
	})

	t.Run("old version does not have ReadOnlyRootFilesystem", func(t *testing.T) {
		dk := dynakube.DynaKube{
			Spec: dynakube.DynaKubeSpec{
				APIURL: testURL,
				OneAgent: oneagent.Spec{
					HostMonitoring: &oneagent.HostInjectSpec{},
				},
			},
			Status: dynakube.DynaKubeStatus{
				OneAgent: oneagent.Status{
					VersionStatus: status.VersionStatus{
						Version: "1.290.18.20240520-124108",
					},
				},
			},
		}
		dsBuilder := NewHostMonitoring(&dk, testClusterID)
		ds, err := dsBuilder.BuildDaemonSet()
		require.NoError(t, err)

		assert.GreaterOrEqual(t, 1, len(ds.Spec.Template.Spec.Containers))

		securityContext := ds.Spec.Template.Spec.Containers[0].SecurityContext

		assert.NotNil(t, securityContext)
		require.NotNil(t, securityContext.ReadOnlyRootFilesystem)
		assert.False(t, *securityContext.ReadOnlyRootFilesystem)
	})

	t.Run("newer version has ReadOnlyRootFilesystem", func(t *testing.T) {
		dk := dynakube.DynaKube{
			Spec: dynakube.DynaKubeSpec{
				APIURL: testURL,
				OneAgent: oneagent.Spec{
					HostMonitoring: &oneagent.HostInjectSpec{},
				},
			},
			Status: dynakube.DynaKubeStatus{
				OneAgent: oneagent.Status{
					VersionStatus: status.VersionStatus{
						Version: "1.291.18.20240520-124108",
					},
				},
			},
		}
		dsBuilder := NewHostMonitoring(&dk, testClusterID)
		ds, err := dsBuilder.BuildDaemonSet()
		require.NoError(t, err)

		assert.GreaterOrEqual(t, 1, len(ds.Spec.Template.Spec.Containers))

		securityContext := ds.Spec.Template.Spec.Containers[0].SecurityContext

		assert.NotNil(t, securityContext)
		require.NotNil(t, securityContext.ReadOnlyRootFilesystem)
		assert.True(t, *securityContext.ReadOnlyRootFilesystem)
	})

	t.Run(`privileged security context when feature flag is enabled`, func(t *testing.T) {
		dk := dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					exp.OAPrivilegedKey: "true",
				},
			},
			Spec: dynakube.DynaKubeSpec{
				APIURL: testURL,
				OneAgent: oneagent.Spec{
					HostMonitoring: &oneagent.HostInjectSpec{},
				},
			},
		}
		dsBuilder := NewHostMonitoring(&dk, testClusterID)
		ds, err := dsBuilder.BuildDaemonSet()
		require.NoError(t, err)

		assert.GreaterOrEqual(t, 1, len(ds.Spec.Template.Spec.Containers))

		securityContext := ds.Spec.Template.Spec.Containers[0].SecurityContext

		assert.NotNil(t, securityContext)
		assert.Equal(t, ptr.To(int64(1000)), securityContext.RunAsUser)
		assert.Equal(t, ptr.To(int64(1000)), securityContext.RunAsGroup)
		assert.Equal(t, ptr.To(true), securityContext.RunAsNonRoot)
		assert.Equal(t, ptr.To(true), securityContext.Privileged)
		assert.Empty(t, securityContext.Capabilities)
		assert.Nil(t, securityContext.SeccompProfile)
	})

	t.Run(`privileged security context when feature flag is enabled for classic fullstack`, func(t *testing.T) {
		dk := dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					exp.OAPrivilegedKey: "true",
				},
			},
			Spec: dynakube.DynaKubeSpec{
				APIURL: testURL,
				OneAgent: oneagent.Spec{
					ClassicFullStack: &oneagent.HostInjectSpec{},
				},
			},
		}
		dsBuilder := NewClassicFullStack(&dk, testClusterID)
		ds, err := dsBuilder.BuildDaemonSet()
		require.NoError(t, err)

		assert.GreaterOrEqual(t, 1, len(ds.Spec.Template.Spec.Containers))

		securityContext := ds.Spec.Template.Spec.Containers[0].SecurityContext

		assert.NotNil(t, securityContext)
		assert.Nil(t, securityContext.RunAsUser)
		assert.Nil(t, securityContext.RunAsGroup)
		assert.Nil(t, securityContext.RunAsNonRoot)
		assert.Equal(t, ptr.To(true), securityContext.Privileged)
		assert.Empty(t, securityContext.Capabilities)
		assert.Nil(t, securityContext.SeccompProfile)
	})

	t.Run(`localhost seccomp profile when feature flag is enabled`, func(t *testing.T) {
		customSecCompProfile := "seccomp.json"
		dk := dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{},
			Spec: dynakube.DynaKubeSpec{
				APIURL: testURL,
				OneAgent: oneagent.Spec{
					ClassicFullStack: &oneagent.HostInjectSpec{
						SecCompProfile: customSecCompProfile,
					},
				},
			},
		}
		dsBuilder := NewClassicFullStack(&dk, testClusterID)
		ds, err := dsBuilder.BuildDaemonSet()
		require.NoError(t, err)

		assert.GreaterOrEqual(t, 1, len(ds.Spec.Template.Spec.Containers))

		securityContext := ds.Spec.Template.Spec.Containers[0].SecurityContext

		assert.NotNil(t, securityContext)
		assert.Nil(t, securityContext.RunAsUser)
		assert.Nil(t, securityContext.RunAsGroup)
		assert.Nil(t, securityContext.RunAsNonRoot)
		assert.Nil(t, securityContext.Privileged)
		assert.NotEmpty(t, securityContext.Capabilities)
		assert.Equal(t, corev1.SeccompProfileTypeLocalhost, securityContext.SeccompProfile.Type)
		assert.Equal(t, customSecCompProfile, *securityContext.SeccompProfile.LocalhostProfile)
	})

	t.Run(`localhost seccomp profile disabled if privileged security context enabled`, func(t *testing.T) {
		customSecCompProfile := "seccomp.json"
		dk := dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					exp.OAPrivilegedKey: "true",
				},
			},
			Spec: dynakube.DynaKubeSpec{
				APIURL: testURL,
				OneAgent: oneagent.Spec{
					ClassicFullStack: &oneagent.HostInjectSpec{
						SecCompProfile: customSecCompProfile,
					},
				},
			},
		}
		dsBuilder := NewClassicFullStack(&dk, testClusterID)
		ds, err := dsBuilder.BuildDaemonSet()
		require.NoError(t, err)

		assert.GreaterOrEqual(t, 1, len(ds.Spec.Template.Spec.Containers))

		securityContext := ds.Spec.Template.Spec.Containers[0].SecurityContext

		assert.NotNil(t, securityContext)
		assert.Nil(t, securityContext.RunAsUser)
		assert.Nil(t, securityContext.RunAsGroup)
		assert.Nil(t, securityContext.RunAsNonRoot)
		assert.Equal(t, ptr.To(true), securityContext.Privileged)
		assert.Empty(t, securityContext.Capabilities)
		assert.Nil(t, securityContext.SeccompProfile)
	})
}

func TestPodSpecServiceAccountName(t *testing.T) {
	t.Run("service account name is unprivileged + readonly by default", func(t *testing.T) {
		builder := builder{
			dk: &dynakube.DynaKube{},
		}
		podSpec, _ := builder.podSpec()

		assert.Equal(t, serviceAccountName, podSpec.ServiceAccountName)
	})
	t.Run("privileged and readonly is recognized", func(t *testing.T) {
		builder := builder{
			dk: &dynakube.DynaKube{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						exp.OAPrivilegedKey: "true",
					},
				},
			},
		}
		podSpec, _ := builder.podSpec()

		assert.Equal(t, serviceAccountName, podSpec.ServiceAccountName)
	})
	t.Run("service account name is unprivileged if run as unprivileged", func(t *testing.T) {
		dk := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					exp.OAPrivilegedKey: "false",
				},
			},
		}
		builder := builder{
			dk: dk,
		}
		podSpec, _ := builder.podSpec()

		assert.Equal(t, serviceAccountName, podSpec.ServiceAccountName)
	})
}

func TestPodSpecProbes(t *testing.T) {
	expectedHealthcheck := containerv1.HealthConfig{
		Test:        []string{"echo", "super pod"},
		Interval:    10 * time.Second,
		Timeout:     30 * time.Second,
		StartPeriod: 60 * time.Second,
		Retries:     3,
	}

	t.Run("set probes when dynakube oneagent status has healthcheck", func(t *testing.T) {
		builder := builder{
			dk: &dynakube.DynaKube{
				Status: dynakube.DynaKubeStatus{
					OneAgent: oneagent.Status{
						Healthcheck: &expectedHealthcheck,
					},
				},
			},
		}
		podSpec, _ := builder.podSpec()

		actualReadinessProbe := podSpec.Containers[0].ReadinessProbe
		require.NotNil(t, actualReadinessProbe)
		assert.Equal(t, expectedHealthcheck.Test, actualReadinessProbe.Exec.Command)
		assert.Equal(t, int32(expectedHealthcheck.Interval.Seconds()), actualReadinessProbe.PeriodSeconds)
		assert.Equal(t, int32(expectedHealthcheck.Timeout.Seconds()), actualReadinessProbe.TimeoutSeconds)
		assert.Equal(t, int32(expectedHealthcheck.StartPeriod.Seconds()), actualReadinessProbe.InitialDelaySeconds)
		assert.Equal(t, int32(expectedHealthcheck.Retries), actualReadinessProbe.FailureThreshold) //nolint:gosec
		assert.Equal(t, probeDefaultSuccessThreshold, actualReadinessProbe.SuccessThreshold)

		actualLivenessProbe := podSpec.Containers[0].LivenessProbe
		require.NotNil(t, actualLivenessProbe)
		assert.Equal(t, expectedHealthcheck.Test, actualLivenessProbe.Exec.Command)
		assert.Equal(t, int32(expectedHealthcheck.Interval.Seconds()), actualLivenessProbe.PeriodSeconds)
		assert.Equal(t, int32(expectedHealthcheck.Timeout.Seconds()), actualLivenessProbe.TimeoutSeconds)
		assert.Equal(t, int32(expectedHealthcheck.StartPeriod.Seconds()), actualLivenessProbe.InitialDelaySeconds)
		assert.Equal(t, int32(expectedHealthcheck.Retries), actualLivenessProbe.FailureThreshold) //nolint:gosec
		assert.Equal(t, probeDefaultSuccessThreshold, actualLivenessProbe.SuccessThreshold)
	})
	t.Run("check probes with 1200s start period", func(t *testing.T) {
		updatedHealthCheck := expectedHealthcheck.DeepCopy()
		updatedHealthCheck.StartPeriod = 1200 * time.Second

		builder := builder{
			dk: &dynakube.DynaKube{
				Status: dynakube.DynaKubeStatus{
					OneAgent: oneagent.Status{
						Healthcheck: updatedHealthCheck,
					},
				},
			},
		}
		podSpec, _ := builder.podSpec()

		actualReadinessProbe := podSpec.Containers[0].ReadinessProbe
		require.NotNil(t, actualReadinessProbe)
		assert.Equal(t, probeMaxInitialDelay, actualReadinessProbe.InitialDelaySeconds)

		actualLivenessProbe := podSpec.Containers[0].LivenessProbe
		require.NotNil(t, actualLivenessProbe)
		assert.Equal(t, int32(updatedHealthCheck.StartPeriod.Seconds()), actualLivenessProbe.InitialDelaySeconds)
	})
	t.Run("nil probes when dynakube oneagent status has no healthcheck", func(t *testing.T) {
		builder := builder{
			dk: &dynakube.DynaKube{},
		}
		podSpec, _ := builder.podSpec()

		assert.Nil(t, podSpec.Containers[0].ReadinessProbe)
		assert.Nil(t, podSpec.Containers[0].LivenessProbe)
	})
	t.Run("no livenessProbe when skip featureFlag is set", func(t *testing.T) {
		builder := builder{
			dk: &dynakube.DynaKube{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						exp.OASkipLivenessProbeKey: "true",
					},
				},
				Status: dynakube.DynaKubeStatus{
					OneAgent: oneagent.Status{
						Healthcheck: &expectedHealthcheck,
					},
				},
			},
		}
		podSpec, _ := builder.podSpec()

		assert.NotNil(t, podSpec.Containers[0].ReadinessProbe)
		assert.Nil(t, podSpec.Containers[0].LivenessProbe)
	})
}

func TestOneAgentResources(t *testing.T) {
	t.Run("get empty resources if hostInjection spec is nil", func(t *testing.T) {
		builder := builder{}
		resources := builder.oneAgentResource()

		assert.Equal(t, corev1.ResourceRequirements{}, resources)
	})
	t.Run("get resources if hostInjection spec is set", func(t *testing.T) {
		builder := builder{
			hostInjectSpec: &oneagent.HostInjectSpec{
				OneAgentResources: corev1.ResourceRequirements{
					Requests: map[corev1.ResourceName]resource.Quantity{
						corev1.ResourceCPU: *resource.NewScaledQuantity(2, 1),
					},
				},
			},
		}
		resources := builder.oneAgentResource()

		assert.Equal(t, corev1.ResourceRequirements{
			Requests: map[corev1.ResourceName]resource.Quantity{
				corev1.ResourceCPU: *resource.NewScaledQuantity(2, 1),
			},
		}, resources)
	})
}

func TestDNSPolicy(t *testing.T) {
	t.Run("returns default dns policy if hostInjection is nil", func(t *testing.T) {
		builder := builder{}
		dnsPolicy := builder.dnsPolicy()

		assert.Equal(t, corev1.DNSClusterFirstWithHostNet, dnsPolicy)
	})
}

func TestNodeSelector(t *testing.T) {
	t.Run("returns empty map if hostInjectSpec is nil", func(t *testing.T) {
		dsBuilder := builder{}
		nodeSelector := dsBuilder.nodeSelector()

		assert.Equal(t, map[string]string{}, nodeSelector)
	})
	t.Run("returns nodeselector", func(t *testing.T) {
		dsBuilder := builder{
			hostInjectSpec: &oneagent.HostInjectSpec{
				NodeSelector: map[string]string{testKey: testValue},
			},
		}
		nodeSelector := dsBuilder.nodeSelector()

		assert.Contains(t, nodeSelector, testKey)
	})
}

func TestPriorityClass(t *testing.T) {
	t.Run("returns empty string if hostInjectSpec is nil", func(t *testing.T) {
		dsBuilder := builder{}
		priorityClassName := dsBuilder.priorityClassName()

		assert.Empty(t, priorityClassName)
	})
	t.Run("returns nodeselector", func(t *testing.T) {
		dsBuilder := builder{
			hostInjectSpec: &oneagent.HostInjectSpec{
				PriorityClassName: testName,
			},
		}
		priorityClassName := dsBuilder.priorityClassName()

		assert.Equal(t, testName, priorityClassName)
	})
}

func TestTolerations(t *testing.T) {
	t.Run("returns empty list if hostInjectSpec is nil", func(t *testing.T) {
		dsBuilder := builder{}
		tolerations := dsBuilder.tolerations()

		assert.Empty(t, tolerations)
	})
	t.Run("returns tolerations", func(t *testing.T) {
		dsBuilder := builder{
			hostInjectSpec: &oneagent.HostInjectSpec{
				Tolerations: []corev1.Toleration{
					{
						Key:   testKey,
						Value: testValue,
					},
				},
			},
		}
		tolerations := dsBuilder.tolerations()

		assert.Contains(t, tolerations, corev1.Toleration{
			Key:   testKey,
			Value: testValue,
		})
	})
}

func TestImagePullSecrets(t *testing.T) {
	t.Run("returns empty list if instance is null", func(t *testing.T) {
		dsBuilder := builder{}
		pullSecrets := dsBuilder.imagePullSecrets()

		assert.Empty(t, pullSecrets)
	})
	t.Run("returns default instance pull secret", func(t *testing.T) {
		dsBuilder := builder{
			dk: &dynakube.DynaKube{
				ObjectMeta: metav1.ObjectMeta{
					Name: testName,
				},
			},
		}
		pullSecrets := dsBuilder.imagePullSecrets()

		assert.Contains(t, pullSecrets, corev1.LocalObjectReference{
			Name: testName + dynakube.PullSecretSuffix,
		})
	})
	t.Run("returns custom pull secret", func(t *testing.T) {
		dsBuilder := builder{
			dk: &dynakube.DynaKube{
				ObjectMeta: metav1.ObjectMeta{
					Name: testName,
				},
				Spec: dynakube.DynaKubeSpec{
					CustomPullSecret: testValue,
				},
			},
		}
		pullSecrets := dsBuilder.imagePullSecrets()

		assert.Contains(t, pullSecrets, corev1.LocalObjectReference{
			Name: testValue,
		})
	})
}

func TestImmutableOneAgentImage(t *testing.T) {
	t.Run("returns empty string if instance is nil", func(t *testing.T) {
		dsBuilder := builder{}
		image := dsBuilder.immutableOneAgentImage()

		assert.Empty(t, image)
	})
	t.Run("returns instance image", func(t *testing.T) {
		dsBuilder := builder{
			dk: &dynakube.DynaKube{},
		}
		image := dsBuilder.immutableOneAgentImage()

		assert.Equal(t, dsBuilder.dk.OneAgent().GetImage(), image)
	})
}

func TestAnnotations(t *testing.T) {
	t.Run("cloud native has apparmor annotation by default", func(t *testing.T) {
		dk := dynakube.DynaKube{
			Spec: dynakube.DynaKubeSpec{
				OneAgent: oneagent.Spec{
					CloudNativeFullStack: &oneagent.CloudNativeFullStackSpec{
						HostInjectSpec: oneagent.HostInjectSpec{},
					},
				},
			},
		}
		dk.Status.OneAgent.ConnectionInfoStatus.TenantTokenHash = testTokenHash

		expectedTemplateAnnotations := map[string]string{
			webhook.AnnotationDynatraceInject: "false",
			annotationUnprivileged:            annotationUnprivilegedValue,
			annotationTenantTokenHash:         testTokenHash,
			annotationEnableDaemonSetEviction: "false",
		}

		builder := NewCloudNativeFullStack(&dk, testClusterID)
		daemonset, err := builder.BuildDaemonSet()

		require.NoError(t, err)
		assert.NotNil(t, daemonset)
		assert.Equal(t, expectedTemplateAnnotations, daemonset.Spec.Template.Annotations)
	})
	t.Run("host monitoring has apparmor annotation by default", func(t *testing.T) {
		dk := dynakube.DynaKube{
			Spec: dynakube.DynaKubeSpec{
				OneAgent: oneagent.Spec{
					HostMonitoring: &oneagent.HostInjectSpec{},
				},
			},
		}
		dk.Status.OneAgent.ConnectionInfoStatus.TenantTokenHash = testTokenHash

		expectedTemplateAnnotations := map[string]string{
			webhook.AnnotationDynatraceInject: "false",
			annotationUnprivileged:            annotationUnprivilegedValue,
			annotationTenantTokenHash:         testTokenHash,
			annotationEnableDaemonSetEviction: "false",
		}

		builder := NewHostMonitoring(&dk, testClusterID)
		daemonset, err := builder.BuildDaemonSet()

		require.NoError(t, err)
		assert.NotNil(t, daemonset)
		assert.Equal(t, expectedTemplateAnnotations, daemonset.Spec.Template.Annotations)
	})
	t.Run("classic fullstack has apparmor annotation by default", func(t *testing.T) {
		dk := dynakube.DynaKube{
			Spec: dynakube.DynaKubeSpec{
				OneAgent: oneagent.Spec{
					ClassicFullStack: &oneagent.HostInjectSpec{},
				},
			},
		}
		dk.Status.OneAgent.ConnectionInfoStatus.TenantTokenHash = testTokenHash

		expectedTemplateAnnotations := map[string]string{
			webhook.AnnotationDynatraceInject: "false",
			annotationUnprivileged:            annotationUnprivilegedValue,
			annotationTenantTokenHash:         testTokenHash,
			annotationEnableDaemonSetEviction: "false",
		}

		builder := NewClassicFullStack(&dk, testClusterID)
		daemonset, err := builder.BuildDaemonSet()

		require.NoError(t, err)
		assert.NotNil(t, daemonset)
		assert.Equal(t, expectedTemplateAnnotations, daemonset.Spec.Template.Annotations)
	})
	t.Run("annotations are added with cloud native", func(t *testing.T) {
		dk := dynakube.DynaKube{
			Spec: dynakube.DynaKubeSpec{
				OneAgent: oneagent.Spec{
					CloudNativeFullStack: &oneagent.CloudNativeFullStackSpec{
						HostInjectSpec: oneagent.HostInjectSpec{
							Annotations: map[string]string{
								testKey: testName,
							},
						},
					},
				},
			},
		}
		dk.Status.OneAgent.ConnectionInfoStatus.TenantTokenHash = testTokenHash

		expectedTemplateAnnotations := map[string]string{
			webhook.AnnotationDynatraceInject: "false",
			annotationUnprivileged:            annotationUnprivilegedValue,
			testKey:                           testName,
			annotationTenantTokenHash:         testTokenHash,
			annotationEnableDaemonSetEviction: "false",
		}

		builder := NewCloudNativeFullStack(&dk, testClusterID)
		daemonset, err := builder.BuildDaemonSet()

		require.NoError(t, err)
		assert.NotNil(t, daemonset)
		assert.Equal(t, expectedTemplateAnnotations, daemonset.Spec.Template.Annotations)
	})
	t.Run("annotations are added with host monitoring", func(t *testing.T) {
		dk := dynakube.DynaKube{
			Spec: dynakube.DynaKubeSpec{
				OneAgent: oneagent.Spec{
					HostMonitoring: &oneagent.HostInjectSpec{
						Annotations: map[string]string{
							testKey: testName,
						},
					},
				},
			},
		}
		dk.Status.OneAgent.ConnectionInfoStatus.TenantTokenHash = testTokenHash

		expectedTemplateAnnotations := map[string]string{
			webhook.AnnotationDynatraceInject: "false",
			annotationUnprivileged:            annotationUnprivilegedValue,
			testKey:                           testName,
			annotationTenantTokenHash:         testTokenHash,
			annotationEnableDaemonSetEviction: "false",
		}

		builder := NewHostMonitoring(&dk, testClusterID)
		daemonset, err := builder.BuildDaemonSet()

		require.NoError(t, err)
		assert.NotNil(t, daemonset)
		assert.Equal(t, expectedTemplateAnnotations, daemonset.Spec.Template.Annotations)
	})
	t.Run("annotations are added with classic fullstack", func(t *testing.T) {
		dk := dynakube.DynaKube{
			Spec: dynakube.DynaKubeSpec{
				OneAgent: oneagent.Spec{
					ClassicFullStack: &oneagent.HostInjectSpec{
						Annotations: map[string]string{
							testKey: testName,
						},
					},
				},
			},
		}
		dk.Status.OneAgent.ConnectionInfoStatus.TenantTokenHash = testTokenHash

		expectedTemplateAnnotations := map[string]string{
			webhook.AnnotationDynatraceInject: "false",
			annotationUnprivileged:            annotationUnprivilegedValue,
			testKey:                           testName,
			annotationTenantTokenHash:         testTokenHash,
			annotationEnableDaemonSetEviction: "false",
		}

		builder := NewClassicFullStack(&dk, testClusterID)
		daemonset, err := builder.BuildDaemonSet()

		require.NoError(t, err)
		assert.NotNil(t, daemonset)
		assert.Equal(t, expectedTemplateAnnotations, daemonset.Spec.Template.Annotations)
	})
}

func TestOneAgentHostGroup(t *testing.T) {
	t.Run("cloud native - host group settings", func(t *testing.T) {
		dk := dynakube.DynaKube{
			Spec: dynakube.DynaKubeSpec{
				OneAgent: oneagent.Spec{
					CloudNativeFullStack: &oneagent.CloudNativeFullStackSpec{
						HostInjectSpec: oneagent.HostInjectSpec{
							Args: []string{
								"--set-host-group=oldgroup",
							},
						},
					},
					HostGroup: "newgroup",
				},
			},
		}

		builder := NewCloudNativeFullStack(&dk, testClusterID)
		daemonset, err := builder.BuildDaemonSet()

		require.NoError(t, err)
		require.NotNil(t, daemonset)
		assert.Equal(t, "--set-host-group=newgroup", daemonset.Spec.Template.Spec.Containers[0].Args[0])
	})
}

func TestDefaultArguments(t *testing.T) {
	const (
		namespace = "dynatrace"
		name      = "dynakube"
	)

	base := dynakube.DynaKube{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
		Spec: dynakube.DynaKubeSpec{
			APIURL: "https://ENVIRONMENTID.live.dynatrace.com/api",
			Tokens: name,
			OneAgent: oneagent.Spec{
				ClassicFullStack: &oneagent.HostInjectSpec{},
			},
		},
	}
	base.Status.OneAgent.ConnectionInfoStatus.CommunicationHosts = []oneagent.CommunicationHostStatus{
		{
			Protocol: "http",
			Host:     "dummyhost",
			Port:     666,
		},
	}

	t.Run("test customized OneAgent arguments", func(t *testing.T) {
		dk := base.DeepCopy()
		args := []string{
			"--set-app-log-content-access=true",
			"--set-host-id-source=fqdn",
			"--set-host-group=APP_LUSTIG_PETER",
			"--set-server=https://hyper.super.com:9999",
		}
		dk.Spec.OneAgent.ClassicFullStack.Args = args
		dsBuilder := NewClassicFullStack(dk, testClusterID)
		ds, err := dsBuilder.BuildDaemonSet()
		require.NoError(t, err)

		expectedDefaultArguments := []string{
			"--set-app-log-content-access=true",
			"--set-host-group=APP_LUSTIG_PETER",
			"--set-host-id-source=fqdn",
			"--set-host-property=OperatorVersion=$(DT_OPERATOR_VERSION)",
			"--set-no-proxy=",
			"--set-proxy=",
			"--set-server=https://hyper.super.com:9999",
			"--set-tenant=$(DT_TENANT)",
		}
		assert.Equal(t, expectedDefaultArguments, ds.Spec.Template.Spec.Containers[0].Args)
	})

	t.Run("test default OneAgent arguments", func(t *testing.T) {
		dk := base.DeepCopy()
		args := []string{
			"--set-app-log-content-access=true",
			"--set-host-group=APP_LUSTIG_PETER",
			"--set-server=https://hyper.super.com:9999",
		}
		dk.Spec.OneAgent.ClassicFullStack.Args = args
		dsBuilder := NewClassicFullStack(dk, testClusterID)
		ds, err := dsBuilder.BuildDaemonSet()
		require.NoError(t, err)

		expectedDefaultArguments := []string{
			"--set-app-log-content-access=true",
			"--set-host-group=APP_LUSTIG_PETER",
			"--set-host-id-source=auto",
			"--set-host-property=OperatorVersion=$(DT_OPERATOR_VERSION)",
			"--set-no-proxy=",
			"--set-proxy=",
			"--set-server=https://hyper.super.com:9999",
			"--set-tenant=$(DT_TENANT)",
		}
		assert.Equal(t, expectedDefaultArguments, ds.Spec.Template.Spec.Containers[0].Args)
	})
}
