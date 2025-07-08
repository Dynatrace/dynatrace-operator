package oneagent

import (
	"testing"

	"github.com/Dynatrace/dynatrace-bootstrapper/cmd/configure"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/exp"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/oneagent"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/installconfig"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/env"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
)

func TestIsEnabled(t *testing.T) {
	type testCase struct {
		title   string
		podMods func(*corev1.Pod)
		dkMods  func(*dynakube.DynaKube)
		enabled bool
	}

	cases := []testCase{
		{
			title:   "nothing enabled => not enabled",
			podMods: func(p *corev1.Pod) {},
			dkMods:  func(dk *dynakube.DynaKube) {},
			enabled: false,
		},

		{
			title:   "only OA enabled, without FF => enabled",
			podMods: func(p *corev1.Pod) {},
			dkMods: func(dk *dynakube.DynaKube) {
				dk.Spec.OneAgent.ApplicationMonitoring = &oneagent.ApplicationMonitoringSpec{}
			},
			enabled: true,
		},

		{
			title:   "OA + FF enabled => enabled",
			podMods: func(p *corev1.Pod) {},
			dkMods: func(dk *dynakube.DynaKube) {
				dk.Spec.OneAgent.ApplicationMonitoring = &oneagent.ApplicationMonitoringSpec{}
				dk.Annotations = map[string]string{exp.OANodeImagePullKey: "true"}
			},
			enabled: false,
		},
		{
			title: "OA + FF enabled + correct Volume-Type => enabled",
			podMods: func(p *corev1.Pod) {
				p.Annotations = map[string]string{AnnotationVolumeType: EphemeralVolumeType}
			},
			dkMods: func(dk *dynakube.DynaKube) {
				dk.Spec.OneAgent.ApplicationMonitoring = &oneagent.ApplicationMonitoringSpec{}
				dk.Annotations = map[string]string{exp.OANodeImagePullKey: "true"}
			},
			enabled: false,
		},
		{
			title: "OA + FF enabled + incorrect Volume-Type => disabled",
			podMods: func(p *corev1.Pod) {
				p.Annotations = map[string]string{AnnotationVolumeType: CSIVolumeType}
			},
			dkMods: func(dk *dynakube.DynaKube) {
				dk.Spec.OneAgent.ApplicationMonitoring = &oneagent.ApplicationMonitoringSpec{}
				dk.Annotations = map[string]string{exp.OANodeImagePullKey: "true"}
			},
			enabled: false,
		},
	}
	for _, test := range cases {
		t.Run(test.title, func(t *testing.T) {
			pod := &corev1.Pod{}
			test.podMods(pod)

			dk := &dynakube.DynaKube{}
			test.dkMods(dk)

			req := &dtwebhook.MutationRequest{BaseRequest: &dtwebhook.BaseRequest{Pod: pod, DynaKube: *dk}}

			assert.Equal(t, test.enabled, IsEnabled(req.BaseRequest))

			installconfig.SetModulesOverride(t, installconfig.Modules{CSIDriver: false})

			assert.Equal(t, test.enabled, IsEnabled(req.BaseRequest))
		})
	}
}

func TestContainerIsInjected(t *testing.T) {
	t.Run("is injected", func(t *testing.T) {
		request := createTestMutationRequestWithoutInjectedContainers()

		for _, c := range request.Pod.Spec.Containers {
			container := &c
			assert.False(t, containerIsInjected(*container))
			setIsInjectedEnv(container)
			assert.True(t, containerIsInjected(*container))
		}
	})
}

func TestMutate(t *testing.T) {
	mut := NewMutator()

	t.Run("success", func(t *testing.T) {
		request := createTestMutationRequestWithoutInjectedContainers()

		original := createTestMutationRequestWithoutInjectedContainers()
		err := mut.Mutate(request)
		require.NoError(t, err)
		// update install container
		assert.NotEqual(t, original.InstallContainer, request.InstallContainer)

		for i := range request.Pod.Spec.Containers {
			// update each container
			assert.NotEqual(t, original.Pod.Spec.Containers[i], request.Pod.Spec.Containers[i])

			assert.True(t, containerIsInjected(request.Pod.Spec.Containers[i]))
		}
	})
	t.Run("install-path respected", func(t *testing.T) {
		expectedInstallPath := "my-install"
		request := createTestMutationRequestWithoutInjectedContainers()
		request.Pod.Annotations = map[string]string{
			AnnotationInstallPath: expectedInstallPath,
		}

		err := mut.Mutate(request)
		require.NoError(t, err)

		assert.Contains(t, request.InstallContainer.Args, "--"+configure.InstallPathFlag+"="+expectedInstallPath)

		for _, c := range request.Pod.Spec.Containers {
			preload := env.FindEnvVar(c.Env, PreloadEnv)
			require.NotNil(t, preload)
			assert.Contains(t, preload.Value, expectedInstallPath)
		}
	})
	t.Run("no change => no update", func(t *testing.T) {
		request := createTestMutationRequestWithoutInjectedContainers()
		updateContainer := []corev1.Container{}

		for _, c := range request.Pod.Spec.Containers {
			container := &c
			setIsInjectedEnv(container)
			updateContainer = append(updateContainer, *container)
		}

		request.Pod.Spec.Containers = updateContainer

		err := mut.Mutate(request)
		require.NoError(t, err)
	})

	t.Run("no tenantUUID + cloudnative => no update", func(t *testing.T) {
		request := createTestMutationRequestWithoutInjectedContainers()
		request.DynaKube.Spec.OneAgent.CloudNativeFullStack = &oneagent.CloudNativeFullStackSpec{}

		err := mut.Mutate(request)
		require.NoError(t, err)
	})

	t.Run("tenantUUID + cloudnative => update", func(t *testing.T) {
		request := createTestMutationRequestWithoutInjectedContainers()
		request.DynaKube.Spec.OneAgent.CloudNativeFullStack = &oneagent.CloudNativeFullStackSpec{}
		request.DynaKube.Status.OneAgent.ConnectionInfoStatus.TenantUUID = "example"

		err := mut.Mutate(request)
		require.NoError(t, err)
	})
}

func TestReinvoke(t *testing.T) {
	mut := NewMutator()

	t.Run("success", func(t *testing.T) {
		request := createTestMutationRequestWithInjectedContainers()

		original := createTestMutationRequestWithInjectedContainers()
		updated := mut.Reinvoke(request.ToReinvocationRequest())
		require.True(t, updated)

		// no update to install container
		assert.Equal(t, original.InstallContainer, request.InstallContainer)

		for i := range request.Pod.Spec.Containers {
			// only update not-injected
			if containerIsInjected(original.Pod.Spec.Containers[i]) {
				assert.Equal(t, original.Pod.Spec.Containers[i], request.Pod.Spec.Containers[i])
			} else {
				assert.NotEqual(t, original.Pod.Spec.Containers[i], request.Pod.Spec.Containers[i])
			}

			assert.True(t, containerIsInjected(request.Pod.Spec.Containers[i]))
		}
	})

	t.Run("install-path respected", func(t *testing.T) {
		expectedInstallPath := "my-install"
		request := createTestMutationRequestWithoutInjectedContainers()
		request.Pod.Annotations = map[string]string{
			AnnotationInstallPath: expectedInstallPath,
		}

		updated := mut.Reinvoke(request.ToReinvocationRequest())
		require.True(t, updated)

		for _, c := range request.Pod.Spec.Containers {
			preload := env.FindEnvVar(c.Env, PreloadEnv)
			require.NotNil(t, preload)
			assert.Contains(t, preload.Value, expectedInstallPath)
		}
	})

	t.Run("no change => no update", func(t *testing.T) {
		request := createTestMutationRequestWithoutInjectedContainers()
		updateContainer := []corev1.Container{}

		for _, c := range request.Pod.Spec.Containers {
			container := &c
			setIsInjectedEnv(container)
			updateContainer = append(updateContainer, *container)
		}

		request.Pod.Spec.Containers = updateContainer

		updated := mut.Reinvoke(request.ToReinvocationRequest())
		require.False(t, updated)
	})
}

func TestAddOneAgentToContainer(t *testing.T) {
	kubeSystemUUID := "my uuid"
	networkZone := "my zone"
	installPath := "install/path"

	t.Run("add everything", func(t *testing.T) {
		container := corev1.Container{}
		dk := dynakube.DynaKube{
			Spec: dynakube.DynaKubeSpec{
				OneAgent:    oneagent.Spec{ApplicationMonitoring: &oneagent.ApplicationMonitoringSpec{}},
				NetworkZone: networkZone,
			},
			Status: dynakube.DynaKubeStatus{
				KubeSystemUUID: kubeSystemUUID,
			},
		}

		addOneAgentToContainer(dk, &container, corev1.Namespace{}, installPath)

		assert.Len(t, container.VolumeMounts, 3) // preload,bin,config

		dtMetaEnv := env.FindEnvVar(container.Env, DynatraceMetadataEnv)
		require.NotNil(t, dtMetaEnv)
		assert.Contains(t, dtMetaEnv.Value, kubeSystemUUID)

		dtZoneEnv := env.FindEnvVar(container.Env, NetworkZoneEnv)
		require.NotNil(t, dtZoneEnv)
		assert.Equal(t, networkZone, dtZoneEnv.Value)

		preload := env.FindEnvVar(container.Env, PreloadEnv)
		require.NotNil(t, preload)
		assert.Contains(t, preload.Value, installPath)

		assert.True(t, containerIsInjected(container))
	})
}

func createTestMutationRequestWithoutInjectedContainers() *dtwebhook.MutationRequest {
	return &dtwebhook.MutationRequest{
		InstallContainer: &corev1.Container{
			Name: dtwebhook.InstallContainerName,
		},
		BaseRequest: &dtwebhook.BaseRequest{
			Pod: &corev1.Pod{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "sample-container-1",
							Image: "sample-image-1",
						},
						{
							Name:  "sample-container-2",
							Image: "sample-image-2",
						},
					},
				},
			},
		},
	}
}

func createTestMutationRequestWithInjectedContainers() *dtwebhook.MutationRequest {
	request := createTestMutationRequestWithoutInjectedContainers()

	i := 0
	request.Pod.Spec.Containers[i].Env = append(request.Pod.Spec.Containers[i].Env, corev1.EnvVar{
		Name:  isInjectedEnv,
		Value: "true",
	})

	return request
}
