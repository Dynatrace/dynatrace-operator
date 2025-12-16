package oneagent

import (
	"testing"

	"github.com/Dynatrace/dynatrace-bootstrapper/cmd/configure"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/exp"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/oneagent"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/installconfig"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8senv"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestIsEnabled(t *testing.T) {
	matchLabels := map[string]string{
		"match": "me",
	}

	type testCase struct {
		title   string
		podMods func(*corev1.Pod)
		nsMods  func(*corev1.Namespace)
		dkMods  func(*dynakube.DynaKube)
		enabled bool
	}

	cases := []testCase{
		{
			title:   "nothing enabled => not enabled",
			podMods: func(p *corev1.Pod) {},
			nsMods:  func(n *corev1.Namespace) {},
			dkMods:  func(dk *dynakube.DynaKube) {},
			enabled: false,
		},

		{
			title:   "only OA enabled => enabled",
			podMods: func(p *corev1.Pod) {},
			nsMods:  func(n *corev1.Namespace) {},
			dkMods: func(dk *dynakube.DynaKube) {
				dk.Spec.OneAgent.ApplicationMonitoring = &oneagent.ApplicationMonitoringSpec{}
			},
			enabled: true,
		},

		{
			title:   "OA + FF enabled => enabled",
			podMods: func(p *corev1.Pod) {},
			nsMods:  func(n *corev1.Namespace) {},
			dkMods: func(dk *dynakube.DynaKube) {
				dk.Spec.OneAgent.ApplicationMonitoring = &oneagent.ApplicationMonitoringSpec{}
				dk.Annotations = map[string]string{exp.OANodeImagePullKey: "true"}
			},
			enabled: true,
		},
		{
			title:   "OA enabled + auto-inject false => disabled",
			podMods: func(p *corev1.Pod) {},
			nsMods:  func(n *corev1.Namespace) {},
			dkMods: func(dk *dynakube.DynaKube) {
				dk.Spec.OneAgent.ApplicationMonitoring = &oneagent.ApplicationMonitoringSpec{}
				dk.Annotations = map[string]string{
					exp.InjectionAutomaticKey: "false",
				}
			},
			enabled: false,
		},
		{
			title:   "OA enabled + auto-inject false + no pod annotation => disabled",
			podMods: func(p *corev1.Pod) {},
			nsMods:  func(n *corev1.Namespace) {},
			dkMods: func(dk *dynakube.DynaKube) {
				dk.Spec.OneAgent.ApplicationMonitoring = &oneagent.ApplicationMonitoringSpec{}
				dk.Annotations = map[string]string{
					exp.InjectionAutomaticKey: "false",
				}
			},
			enabled: false,
		},
		{
			title: "OA enabled + auto-inject false + pod annotation => enabled",
			podMods: func(p *corev1.Pod) {
				p.Annotations = map[string]string{
					AnnotationInject: "true",
				}
			},
			nsMods: func(n *corev1.Namespace) {},
			dkMods: func(dk *dynakube.DynaKube) {
				dk.Spec.OneAgent.ApplicationMonitoring = &oneagent.ApplicationMonitoringSpec{}
				dk.Annotations = map[string]string{
					exp.InjectionAutomaticKey: "false",
				}
			},
			enabled: true,
		},
		{
			title:   "OA enabled + labels not match => disabled",
			podMods: func(p *corev1.Pod) {},
			nsMods:  func(n *corev1.Namespace) {},
			dkMods: func(dk *dynakube.DynaKube) {
				dk.Spec.OneAgent.ApplicationMonitoring = &oneagent.ApplicationMonitoringSpec{}
				dk.Spec.OneAgent.ApplicationMonitoring.NamespaceSelector = metav1.LabelSelector{
					MatchLabels: matchLabels,
				}
			},
			enabled: false,
		},
		{
			title:   "OA enabled + labels match => enabled",
			podMods: func(p *corev1.Pod) {},
			nsMods: func(n *corev1.Namespace) {
				n.Labels = matchLabels
			},
			dkMods: func(dk *dynakube.DynaKube) {
				dk.Spec.OneAgent.ApplicationMonitoring = &oneagent.ApplicationMonitoringSpec{}
				dk.Spec.OneAgent.ApplicationMonitoring.NamespaceSelector = metav1.LabelSelector{
					MatchLabels: matchLabels,
				}
			},
			enabled: true,
		},
	}
	for _, test := range cases {
		t.Run(test.title, func(t *testing.T) {
			pod := &corev1.Pod{}
			test.podMods(pod)

			ns := &corev1.Namespace{}
			test.nsMods(ns)

			dk := &dynakube.DynaKube{}
			test.dkMods(dk)

			req := &dtwebhook.MutationRequest{BaseRequest: &dtwebhook.BaseRequest{Pod: pod, DynaKube: *dk, Namespace: *ns}}

			assert.Equal(t, test.enabled, IsEnabled(req.BaseRequest))
		})
	}
}

func TestIsSelfExtractingImage(t *testing.T) {
	type testCase struct {
		title        string
		podMods      func(*corev1.Pod)
		dkMods       func(*dynakube.DynaKube)
		isCSIPresent bool
		enabled      bool
	}

	cases := []testCase{
		{
			title:        "nothing enabled => not enabled",
			podMods:      func(p *corev1.Pod) {},
			dkMods:       func(dk *dynakube.DynaKube) {},
			enabled:      false,
			isCSIPresent: false,
		},

		{
			title:   "only OA enabled => not enabled",
			podMods: func(p *corev1.Pod) {},
			dkMods: func(dk *dynakube.DynaKube) {
				dk.Spec.OneAgent.ApplicationMonitoring = &oneagent.ApplicationMonitoringSpec{}
			},
			enabled:      false,
			isCSIPresent: false,
		},

		{
			title:   "OA + FF enabled + no-csi => enabled",
			podMods: func(p *corev1.Pod) {},
			dkMods: func(dk *dynakube.DynaKube) {
				dk.Spec.OneAgent.ApplicationMonitoring = &oneagent.ApplicationMonitoringSpec{}
				dk.Annotations = map[string]string{exp.OANodeImagePullKey: "true"}
			},
			enabled:      true,
			isCSIPresent: false,
		},

		{
			title:   "OA + FF enabled + csi => not enabled",
			podMods: func(p *corev1.Pod) {},
			dkMods: func(dk *dynakube.DynaKube) {
				dk.Spec.OneAgent.ApplicationMonitoring = &oneagent.ApplicationMonitoringSpec{}
				dk.Annotations = map[string]string{exp.OANodeImagePullKey: "true"}
			},
			enabled:      false,
			isCSIPresent: true,
		},

		{
			title: "OA + FF enabled + csi + pod annotation => enabled",
			podMods: func(p *corev1.Pod) {
				p.Annotations = map[string]string{
					AnnotationVolumeType: EphemeralVolumeType,
				}
			},
			dkMods: func(dk *dynakube.DynaKube) {
				dk.Spec.OneAgent.ApplicationMonitoring = &oneagent.ApplicationMonitoringSpec{}
				dk.Annotations = map[string]string{exp.OANodeImagePullKey: "true"}
			},
			enabled:      true,
			isCSIPresent: true,
		},
	}
	for _, test := range cases {
		t.Run(test.title, func(t *testing.T) {
			ns := &corev1.Namespace{}
			pod := &corev1.Pod{}
			dk := &dynakube.DynaKube{}

			test.dkMods(dk)
			test.podMods(pod)

			req := &dtwebhook.MutationRequest{BaseRequest: &dtwebhook.BaseRequest{Pod: pod, DynaKube: *dk, Namespace: *ns}}

			installconfig.SetModulesOverride(t, installconfig.Modules{CSIDriver: test.isCSIPresent})

			assert.Equal(t, test.enabled, IsSelfExtractingImage(req.BaseRequest))
		})
	}
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

			assert.True(t, containerIsInjected(request.Pod.Spec.Containers[i], nil))
		}

		assert.True(t, mut.IsInjected(request.BaseRequest))
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
			preload := k8senv.Find(c.Env, PreloadEnv)
			require.NotNil(t, preload)
			assert.Contains(t, preload.Value, expectedInstallPath)
		}

		assert.True(t, mut.IsInjected(request.BaseRequest))
	})
	t.Run("no change => no update", func(t *testing.T) {
		request := createTestMutationRequestWithoutInjectedContainers()
		updateContainer := []corev1.Container{}

		for _, c := range request.Pod.Spec.Containers {
			container := &c
			addVolumeMounts(container, "test")
			updateContainer = append(updateContainer, *container)
		}

		request.Pod.Spec.Containers = updateContainer

		err := mut.Mutate(request)
		require.NoError(t, err)

		assert.True(t, mut.IsInjected(request.BaseRequest))
	})

	t.Run("no tenantUUID + cloudnative => error", func(t *testing.T) {
		request := createTestMutationRequestWithoutInjectedContainers()
		request.DynaKube.Spec.OneAgent.CloudNativeFullStack = &oneagent.CloudNativeFullStackSpec{}

		err := mut.Mutate(request)
		require.Error(t, err)

		assert.False(t, mut.IsInjected(request.BaseRequest))
	})

	t.Run("tenantUUID + cloudnative => update", func(t *testing.T) {
		request := createTestMutationRequestWithoutInjectedContainers()
		request.DynaKube.Spec.OneAgent.CloudNativeFullStack = &oneagent.CloudNativeFullStackSpec{}
		request.DynaKube.Status.OneAgent.ConnectionInfoStatus.TenantUUID = "example"
		request.DynaKube.Status.CodeModules.Version = "1.2.3"

		err := mut.Mutate(request)
		require.NoError(t, err)

		assert.True(t, mut.IsInjected(request.BaseRequest))
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
			if containerIsInjected(original.Pod.Spec.Containers[i], nil) {
				assert.Equal(t, original.Pod.Spec.Containers[i], request.Pod.Spec.Containers[i])
			} else {
				assert.NotEqual(t, original.Pod.Spec.Containers[i], request.Pod.Spec.Containers[i])
			}

			assert.True(t, containerIsInjected(request.Pod.Spec.Containers[i], nil))
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
			preload := k8senv.Find(c.Env, PreloadEnv)
			require.NotNil(t, preload)
			assert.Contains(t, preload.Value, expectedInstallPath)
		}
	})

	t.Run("no change => no update", func(t *testing.T) {
		request := createTestMutationRequestWithoutInjectedContainers()
		updateContainer := []corev1.Container{}

		for _, c := range request.Pod.Spec.Containers {
			container := &c
			addVolumeMounts(container, "test")
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

		assert.Len(t, container.VolumeMounts, 2) // preload,bin

		dtMetaEnv := k8senv.Find(container.Env, DynatraceMetadataEnv)
		require.NotNil(t, dtMetaEnv)
		assert.Contains(t, dtMetaEnv.Value, kubeSystemUUID)

		dtZoneEnv := k8senv.Find(container.Env, NetworkZoneEnv)
		require.NotNil(t, dtZoneEnv)
		assert.Equal(t, networkZone, dtZoneEnv.Value)

		preload := k8senv.Find(container.Env, PreloadEnv)
		require.NotNil(t, preload)
		assert.Contains(t, preload.Value, installPath)

		storageEnv := k8senv.Find(container.Env, DtStorageEnv)
		require.NotNil(t, storageEnv)
		assert.Contains(t, storageEnv.Value, DtStoragePath)

		assert.True(t, containerIsInjected(container, nil))
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
			DynaKube: dynakube.DynaKube{
				Spec: dynakube.DynaKubeSpec{OneAgent: oneagent.Spec{
					ApplicationMonitoring: &oneagent.ApplicationMonitoringSpec{
						AppInjectionSpec: oneagent.AppInjectionSpec{
							CodeModulesImage: "testimage",
						},
					},
				}},
				Status: dynakube.DynaKubeStatus{
					CodeModules: oneagent.CodeModulesStatus{
						VersionStatus: status.VersionStatus{
							ImageID: "testimage",
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
	addVolumeMounts(&request.Pod.Spec.Containers[i], "test")

	return request
}

func Test_setInjectedAnnotation(t *testing.T) {
	t.Run("should add annotation to nil map", func(t *testing.T) {
		mut := NewMutator()
		request := createTestMutationRequestWithInjectedContainers()

		require.False(t, mut.IsInjected(request.BaseRequest))
		setInjectedAnnotation(request.Pod)
		require.Len(t, request.Pod.Annotations, 1)
		require.True(t, mut.IsInjected(request.BaseRequest))
	})

	t.Run("should remove reason from map", func(t *testing.T) {
		mut := NewMutator()
		request := createTestMutationRequestWithInjectedContainers()
		setNotInjectedAnnotationFunc("test")(request.Pod)

		require.False(t, mut.IsInjected(request.BaseRequest))
		setInjectedAnnotation(request.Pod)
		require.Len(t, request.Pod.Annotations, 1)
		require.True(t, mut.IsInjected(request.BaseRequest))
	})
}

func Test_setNotInjectedAnnotationFunc(t *testing.T) {
	t.Run("should add annotations to nil map", func(t *testing.T) {
		mut := NewMutator()
		request := createTestMutationRequestWithoutInjectedContainers()

		require.False(t, mut.IsInjected(request.BaseRequest))
		setNotInjectedAnnotationFunc("test")(request.Pod)
		require.Len(t, request.Pod.Annotations, 2)
		require.False(t, mut.IsInjected(request.BaseRequest))
	})
}
