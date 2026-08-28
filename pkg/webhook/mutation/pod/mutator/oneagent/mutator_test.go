// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package oneagent

import (
	"testing"

	"github.com/Dynatrace/dynatrace-bootstrapper/cmd/k8sinit/configure"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/exp"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/oneagent"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/installconfig"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8senv"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	nodev1 "k8s.io/api/node/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
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
			title:   "OA enabled + FF enabled => enabled",
			podMods: func(p *corev1.Pod) {},
			nsMods:  func(n *corev1.Namespace) {},
			dkMods: func(dk *dynakube.DynaKube) {
				dk.Spec.OneAgent.ApplicationMonitoring = &oneagent.ApplicationMonitoringSpec{}
				dk.Annotations = map[string]string{
					exp.InjectionAutomaticKey: "true",
				}
			},
			enabled: true,
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
			title: "OA enabled + auto-inject false + pod annotation false => disabled",
			podMods: func(p *corev1.Pod) {
				p.Annotations = map[string]string{
					AnnotationInject: "false",
				}
			},
			nsMods: func(n *corev1.Namespace) {},
			dkMods: func(dk *dynakube.DynaKube) {
				dk.Spec.OneAgent.ApplicationMonitoring = &oneagent.ApplicationMonitoringSpec{}
				dk.Annotations = map[string]string{
					exp.InjectionAutomaticKey: "false",
				}
			},
			enabled: false,
		},
		{
			title: "OA disable + auto-inject true => disabled",
			podMods: func(p *corev1.Pod) {
				p.Annotations = map[string]string{
					AnnotationInject: "true",
				}
			},
			nsMods:  func(n *corev1.Namespace) {},
			dkMods:  func(dk *dynakube.DynaKube) {},
			enabled: false,
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
		{
			title: "OA enabled + labels match + pod annotation => enabled",
			podMods: func(p *corev1.Pod) {
				p.Annotations = map[string]string{
					AnnotationInject: "true",
				}
			},
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
		{
			title: "OA enabled + labels not match => disabled",
			podMods: func(p *corev1.Pod) {
				p.Annotations = map[string]string{
					AnnotationInject: "true",
				}
			},
			nsMods: func(n *corev1.Namespace) {},
			dkMods: func(dk *dynakube.DynaKube) {
				dk.Spec.OneAgent.ApplicationMonitoring = &oneagent.ApplicationMonitoringSpec{}
				dk.Spec.OneAgent.ApplicationMonitoring.NamespaceSelector = metav1.LabelSelector{
					MatchLabels: matchLabels,
				}
			},
			enabled: false,
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
			title:   "OA + image set + no-csi => enabled",
			podMods: func(p *corev1.Pod) {},
			dkMods: func(dk *dynakube.DynaKube) {
				dk.Spec.OneAgent.ApplicationMonitoring = &oneagent.ApplicationMonitoringSpec{}
				dk.Status.CodeModules.ImageID = "testImage"
			},
			enabled:      true,
			isCSIPresent: false,
		},

		{
			title:   "OA + image set + csi => not enabled",
			podMods: func(p *corev1.Pod) {},
			dkMods: func(dk *dynakube.DynaKube) {
				dk.Spec.OneAgent.ApplicationMonitoring = &oneagent.ApplicationMonitoringSpec{}
				dk.Status.CodeModules.ImageID = "testImage"
			},
			enabled:      false,
			isCSIPresent: true,
		},

		{
			title: "OA + image set + csi + pod annotation => enabled",
			podMods: func(p *corev1.Pod) {
				p.Annotations = map[string]string{
					AnnotationVolumeType: EphemeralVolumeType,
				}
			},
			dkMods: func(dk *dynakube.DynaKube) {
				dk.Spec.OneAgent.ApplicationMonitoring = &oneagent.ApplicationMonitoringSpec{}
				dk.Status.CodeModules.ImageID = "testImage"
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

func TestValidateInstallPath(t *testing.T) {
	t.Run("can't be just root", func(t *testing.T) {
		require.Error(t, validateInstallPath("/"))
	})

	t.Run("relative install path is rejected", func(t *testing.T) {
		require.Error(t, validateInstallPath("relative/path"))
	})

	t.Run("install path with separator is rejected", func(t *testing.T) {
		for _, path := range []string{
			"/valid/path,/injected/path",
			"/valid/path:/other",
			"/valid/path\x00/other",
		} {
			require.Error(t, validateInstallPath(path))
		}
	})

	t.Run("install path with whitespace is rejected", func(t *testing.T) {
		for _, path := range []string{
			"/valid/path\n/injected/path",
			"/valid/path\r/other",
			"/valid/path\t/other",
			"/valid/path\x00/other",
		} {
			require.Error(t, validateInstallPath(path))
		}
	})

	t.Run("unclean install path is rejected", func(t *testing.T) {
		require.Error(t, validateInstallPath("/valid/../path"))
	})

	// Valid but rejected: spaces in paths are indistinguishable from
	// whitespace separators in ld.so.preload, so we treat them as invalid.
	t.Run("path with space is rejected despite being a valid linux path", func(t *testing.T) {
		require.Error(t, validateInstallPath("/opt/my agent/dynatrace"))
	})
}

func newTestMutator() *Mutator {
	return &Mutator{
		client: fake.NewClientBuilder().WithScheme(scheme.Scheme).Build(),
	}
}

func TestMutate(t *testing.T) {
	mut := newTestMutator()

	t.Run("success", func(t *testing.T) {
		request := createTestMutationRequestWithoutInjectedContainers(t)

		original := createTestMutationRequestWithoutInjectedContainers(t)
		err := mut.Mutate(request)
		require.NoError(t, err)
		// update install container
		assert.NotEqual(t, original.InstallContainer, request.InstallContainer)

		for i := range request.Pod.Spec.Containers {
			// update each container
			assert.NotEqual(t, original.Pod.Spec.Containers[i], request.Pod.Spec.Containers[i])

			assert.True(t, containerIsInjected(request.Pod.Spec.Containers[i], nil))
		}

		assert.True(t, mut.IsInjected(t.Context(), request.BaseRequest))
	})
	t.Run("install-path respected", func(t *testing.T) {
		expectedInstallPath := "/my-install"
		request := createTestMutationRequestWithoutInjectedContainers(t)
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

		assert.True(t, mut.IsInjected(t.Context(), request.BaseRequest))
	})
	t.Run("no change => no update", func(t *testing.T) {
		request := createTestMutationRequestWithoutInjectedContainers(t)
		for i := range request.Pod.Spec.Containers {
			addVolumeMounts(&request.Pod.Spec.Containers[i], "test")
		}

		err := mut.Mutate(request)
		require.NoError(t, err)

		assert.True(t, mut.IsInjected(t.Context(), request.BaseRequest))
	})

	t.Run("install-path with separator => error", func(t *testing.T) {
		request := createTestMutationRequestWithoutInjectedContainers(t)
		request.Pod.Annotations = map[string]string{
			AnnotationInstallPath: "my:install",
		}

		err := mut.Mutate(request)
		require.ErrorAs(t, err, new(dtwebhook.MutatorError))
		assert.False(t, mut.IsInjected(t.Context(), request.BaseRequest))
	})

	t.Run("install-path with whitespace => error", func(t *testing.T) {
		request := createTestMutationRequestWithoutInjectedContainers(t)
		request.Pod.Annotations = map[string]string{
			AnnotationInstallPath: "my install",
		}

		err := mut.Mutate(request)
		require.ErrorAs(t, err, new(dtwebhook.MutatorError))
		assert.False(t, mut.IsInjected(t.Context(), request.BaseRequest))
	})

	t.Run("no tenantUUID + cloudnative => error", func(t *testing.T) {
		request := createTestMutationRequestWithoutInjectedContainers(t)
		request.DynaKube.Spec.OneAgent.CloudNativeFullStack = &oneagent.CloudNativeFullStackSpec{}

		err := mut.Mutate(request)
		require.Error(t, err)

		assert.False(t, mut.IsInjected(t.Context(), request.BaseRequest))
	})

	t.Run("tenantUUID + cloudnative => update", func(t *testing.T) {
		request := createTestMutationRequestWithoutInjectedContainers(t)
		request.DynaKube.Spec.OneAgent.CloudNativeFullStack = &oneagent.CloudNativeFullStackSpec{}
		request.DynaKube.Status.OneAgent.ConnectionInfo.TenantUUID = "example"
		request.DynaKube.Status.CodeModules.Version = "1.2.3"

		err := mut.Mutate(request)
		require.NoError(t, err)

		assert.True(t, mut.IsInjected(t.Context(), request.BaseRequest))
	})
}

func TestReinvoke(t *testing.T) {
	mut := newTestMutator()

	t.Run("success", func(t *testing.T) {
		request := createTestMutationRequestWithInjectedContainers(t)

		original := createTestMutationRequestWithInjectedContainers(t)
		updated := mut.Reinvoke(t.Context(), request.ToReinvocationRequest())
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
		expectedInstallPath := "/my-install"
		request := createTestMutationRequestWithoutInjectedContainers(t)
		request.Pod.Annotations = map[string]string{
			AnnotationInstallPath: expectedInstallPath,
		}

		updated := mut.Reinvoke(t.Context(), request.ToReinvocationRequest())
		require.True(t, updated)

		for _, c := range request.Pod.Spec.Containers {
			preload := k8senv.Find(c.Env, PreloadEnv)
			require.NotNil(t, preload)
			assert.Contains(t, preload.Value, expectedInstallPath)
		}
	})

	t.Run("no change => no update", func(t *testing.T) {
		request := createTestMutationRequestWithoutInjectedContainers(t)
		for i := range request.Pod.Spec.Containers {
			addVolumeMounts(&request.Pod.Spec.Containers[i], "test")
		}

		updated := mut.Reinvoke(t.Context(), request.ToReinvocationRequest())
		require.False(t, updated)
	})

	t.Run("incorrect install-path => no update", func(t *testing.T) {
		request := createTestMutationRequestWithoutInjectedContainers(t)
		request.Pod.Annotations = map[string]string{
			AnnotationInstallPath: "my install",
		}

		updated := mut.Reinvoke(t.Context(), request.ToReinvocationRequest())
		require.False(t, updated)
	})
}

func TestMutateUserContainers(t *testing.T) {
	kubeSystemUUID := "my uuid"
	networkZone := "my zone"
	installPath := "install/path"

	makeRequest := func(networkZone string) *dtwebhook.BaseRequest {
		return &dtwebhook.BaseRequest{
			Pod: &corev1.Pod{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{Name: "app"}},
				},
			},
			DynaKube: dynakube.DynaKube{
				Spec: dynakube.DynaKubeSpec{
					OneAgent:    oneagent.Spec{ApplicationMonitoring: &oneagent.ApplicationMonitoringSpec{}},
					NetworkZone: networkZone,
				},
				Status: dynakube.DynaKubeStatus{KubeSystemUUID: kubeSystemUUID},
			},
		}
	}

	t.Run("adds all envs and volume mounts", func(t *testing.T) {
		request := makeRequest(networkZone)

		mutateUserContainers(request, installPath, "", logd.Get())

		container := &request.Pod.Spec.Containers[0]
		assert.Len(t, container.VolumeMounts, 2) // bin + preload

		require.NotNil(t, k8senv.Find(container.Env, DynatraceMetadataEnv))
		assert.Contains(t, k8senv.Find(container.Env, DynatraceMetadataEnv).Value, kubeSystemUUID)

		require.NotNil(t, k8senv.Find(container.Env, NetworkZoneEnv))
		assert.Equal(t, networkZone, k8senv.Find(container.Env, NetworkZoneEnv).Value)

		require.NotNil(t, k8senv.Find(container.Env, PreloadEnv))
		assert.Contains(t, k8senv.Find(container.Env, PreloadEnv).Value, installPath)

		require.NotNil(t, k8senv.Find(container.Env, DTStorageEnv))
		assert.Contains(t, k8senv.Find(container.Env, DTStorageEnv).Value, DTStoragePath)

		assert.True(t, containerIsInjected(*container, nil))
	})

	t.Run("injects runtime class handler when resolved", func(t *testing.T) {
		handler := "kata"
		request := makeRequest("")

		mutateUserContainers(request, installPath, handler, logd.Get())

		container := &request.Pod.Spec.Containers[0]
		runtimeEnv := k8senv.Find(container.Env, PodRuntimeClassEnv)
		require.NotNil(t, runtimeEnv)
		assert.Equal(t, handler, runtimeEnv.Value)
	})

	t.Run("skips runtime class env when handler is empty", func(t *testing.T) {
		request := makeRequest("")

		mutateUserContainers(request, installPath, "", logd.Get())

		container := &request.Pod.Spec.Containers[0]
		assert.Nil(t, k8senv.Find(container.Env, PodRuntimeClassEnv))
	})
}

func TestResolveRuntimeClassHandler(t *testing.T) {
	runtimeClassName := "kata-containers"
	handler := "kata"

	runtimeClass := &nodev1.RuntimeClass{
		ObjectMeta: metav1.ObjectMeta{Name: runtimeClassName},
		Handler:    handler,
	}

	_, log := logd.NewFromContext(t.Context(), "test")

	t.Run("resolves handler from RuntimeClass", func(t *testing.T) {
		mut := &Mutator{
			client: fake.NewClientBuilder().WithScheme(scheme.Scheme).WithObjects(runtimeClass).Build(),
		}

		result := mut.resolveRuntimeClassHandler(t.Context(), &runtimeClassName, log)

		assert.Equal(t, handler, result)
	})

	t.Run("returns empty string when RuntimeClass not found", func(t *testing.T) {
		mut := &Mutator{
			client: fake.NewClientBuilder().WithScheme(scheme.Scheme).Build(),
		}

		result := mut.resolveRuntimeClassHandler(t.Context(), &runtimeClassName, log)

		assert.Empty(t, result)
	})

	t.Run("returns empty string when runtimeClassName is nil", func(t *testing.T) {
		mut := &Mutator{
			client: fake.NewClientBuilder().WithScheme(scheme.Scheme).Build(),
		}

		result := mut.resolveRuntimeClassHandler(t.Context(), nil, log)

		assert.Empty(t, result)
	})
}

func createTestMutationRequestWithoutInjectedContainers(t *testing.T) *dtwebhook.MutationRequest {
	t.Helper()

	return &dtwebhook.MutationRequest{
		Context: t.Context(),
		InstallContainer: &corev1.Container{
			Name: dtwebhook.InstallContainerName,
		},
		BaseRequest: &dtwebhook.BaseRequest{
			Pod: &corev1.Pod{
				TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{},
				},
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
				Status: corev1.PodStatus{},
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

func createTestMutationRequestWithInjectedContainers(t *testing.T) *dtwebhook.MutationRequest {
	t.Helper()

	request := createTestMutationRequestWithoutInjectedContainers(t)

	i := 0
	addVolumeMounts(&request.Pod.Spec.Containers[i], "test")

	return request
}

func Test_setInjectedAnnotation(t *testing.T) {
	t.Run("should add annotation to nil map", func(t *testing.T) {
		mut := newTestMutator()
		request := createTestMutationRequestWithInjectedContainers(t)

		require.False(t, mut.IsInjected(t.Context(), request.BaseRequest))
		setInjectedAnnotation(request.Pod)
		require.Len(t, request.Pod.Annotations, 1)
		require.True(t, mut.IsInjected(t.Context(), request.BaseRequest))
	})

	t.Run("should remove reason from map", func(t *testing.T) {
		mut := newTestMutator()
		request := createTestMutationRequestWithInjectedContainers(t)
		setNotInjectedAnnotationFunc("test")(request.Pod)

		require.False(t, mut.IsInjected(t.Context(), request.BaseRequest))
		setInjectedAnnotation(request.Pod)
		require.Len(t, request.Pod.Annotations, 1)
		require.True(t, mut.IsInjected(t.Context(), request.BaseRequest))
	})
}

func Test_setNotInjectedAnnotationFunc(t *testing.T) {
	t.Run("should add annotations to nil map", func(t *testing.T) {
		mut := newTestMutator()
		request := createTestMutationRequestWithoutInjectedContainers(t)

		require.False(t, mut.IsInjected(t.Context(), request.BaseRequest))
		setNotInjectedAnnotationFunc("test")(request.Pod)
		require.Len(t, request.Pod.Annotations, 2)
		require.False(t, mut.IsInjected(t.Context(), request.BaseRequest))
	})
}
