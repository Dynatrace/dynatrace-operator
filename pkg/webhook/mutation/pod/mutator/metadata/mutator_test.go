package metadata

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	containerattr "github.com/Dynatrace/dynatrace-bootstrapper/cmd/k8sinit/configure/attributes/container"
	podattr "github.com/Dynatrace/dynatrace-bootstrapper/cmd/k8sinit/configure/attributes/pod"
	"github.com/Dynatrace/dynatrace-operator/cmd/bootstrapper"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/exp"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/metadataenrichment"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/oneagent"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/installconfig"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/attributes"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator"
	oacommon "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator/oneagent"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/volumes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

func TestIsEnabled(t *testing.T) {
	matchLabels := map[string]string{
		"match": "me",
	}

	type testCase struct {
		title      string
		podMods    func(*corev1.Pod)
		nsMods     func(*corev1.Namespace)
		dkMods     func(*dynakube.DynaKube)
		withCSI    bool
		withoutCSI bool
	}

	cases := []testCase{
		{
			title:      "nothing enabled => not enabled",
			podMods:    func(p *corev1.Pod) {},
			nsMods:     func(n *corev1.Namespace) {},
			dkMods:     func(dk *dynakube.DynaKube) {},
			withCSI:    false,
			withoutCSI: false,
		},
		{
			title:   "meta enabled => enabled",
			podMods: func(p *corev1.Pod) {},
			nsMods:  func(n *corev1.Namespace) {},
			dkMods: func(dk *dynakube.DynaKube) {
				dk.Spec.MetadataEnrichment.Enabled = ptr.To(true)
			},
			withCSI:    true,
			withoutCSI: true,
		},
		{
			title:   "meta enabled + auto-inject false => disabled",
			podMods: func(p *corev1.Pod) {},
			nsMods:  func(n *corev1.Namespace) {},
			dkMods: func(dk *dynakube.DynaKube) {
				dk.Spec.MetadataEnrichment.Enabled = ptr.To(true)
				dk.Annotations = map[string]string{
					exp.InjectionAutomaticKey: "false",
				}
			},
			withCSI:    false,
			withoutCSI: false,
		},
		{
			title:   "meta enabled + auto-inject false + no pod annotation => disabled",
			podMods: func(p *corev1.Pod) {},
			nsMods:  func(n *corev1.Namespace) {},
			dkMods: func(dk *dynakube.DynaKube) {
				dk.Spec.MetadataEnrichment.Enabled = ptr.To(true)
				dk.Annotations = map[string]string{
					exp.InjectionAutomaticKey: "false",
				}
			},
			withCSI:    false,
			withoutCSI: false,
		},
		{
			title: "meta enabled + auto-inject false + pod annotation => enabled",
			podMods: func(p *corev1.Pod) {
				p.Annotations = map[string]string{
					AnnotationInject: "true",
				}
			},
			nsMods: func(n *corev1.Namespace) {},
			dkMods: func(dk *dynakube.DynaKube) {
				dk.Spec.MetadataEnrichment.Enabled = ptr.To(true)
				dk.Annotations = map[string]string{
					exp.InjectionAutomaticKey: "false",
				}
			},
			withCSI:    true,
			withoutCSI: true,
		},
		{
			title:   "meta enabled + labels not match => disabled",
			podMods: func(p *corev1.Pod) {},
			nsMods:  func(n *corev1.Namespace) {},
			dkMods: func(dk *dynakube.DynaKube) {
				dk.Spec.MetadataEnrichment.Enabled = ptr.To(true)
				dk.Spec.MetadataEnrichment.NamespaceSelector = metav1.LabelSelector{
					MatchLabels: matchLabels,
				}
			},
			withCSI:    false,
			withoutCSI: false,
		},
		{
			title:   "meta enabled + labels match => enabled",
			podMods: func(p *corev1.Pod) {},
			nsMods: func(n *corev1.Namespace) {
				n.Labels = matchLabels
			},
			dkMods: func(dk *dynakube.DynaKube) {
				dk.Spec.MetadataEnrichment.Enabled = ptr.To(true)
				dk.Spec.MetadataEnrichment.NamespaceSelector = metav1.LabelSelector{
					MatchLabels: matchLabels,
				}
			},
			withCSI:    true,
			withoutCSI: true,
		},
		{
			title:   "OA => disabled",
			podMods: func(p *corev1.Pod) {},
			nsMods:  func(n *corev1.Namespace) {},
			dkMods: func(dk *dynakube.DynaKube) {
				dk.Spec.OneAgent.ApplicationMonitoring = &oneagent.ApplicationMonitoringSpec{}
				dk.Spec.MetadataEnrichment.Enabled = ptr.To(false)
			},
			withCSI:    false,
			withoutCSI: false,
		},
		{
			title: "OA + ephemeral Volume-Type => disabled",
			podMods: func(p *corev1.Pod) {
				p.Annotations = map[string]string{oacommon.AnnotationVolumeType: oacommon.EphemeralVolumeType}
			},
			nsMods: func(n *corev1.Namespace) {},
			dkMods: func(dk *dynakube.DynaKube) {
				dk.Spec.OneAgent.ApplicationMonitoring = &oneagent.ApplicationMonitoringSpec{}
			},
			withCSI:    false,
			withoutCSI: false,
		},
		{
			title: "OA + csi Volume-Type => disabled",
			podMods: func(p *corev1.Pod) {
				p.Annotations = map[string]string{oacommon.AnnotationVolumeType: oacommon.CSIVolumeType}
			},
			nsMods: func(n *corev1.Namespace) {},
			dkMods: func(dk *dynakube.DynaKube) {
				dk.Spec.OneAgent.ApplicationMonitoring = &oneagent.ApplicationMonitoringSpec{}
				dk.Spec.MetadataEnrichment.Enabled = ptr.To(false)
			},
			withCSI:    false,
			withoutCSI: false,
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

			mut := NewMutator(fake.NewClient())

			req := &dtwebhook.MutationRequest{BaseRequest: &dtwebhook.BaseRequest{Pod: pod, DynaKube: *dk, Namespace: *ns}}

			assert.Equal(t, test.withCSI, mut.IsEnabled(t.Context(), req.BaseRequest))

			installconfig.SetModulesOverride(t, installconfig.Modules{CSIDriver: false})

			assert.Equal(t, test.withoutCSI, mut.IsEnabled(t.Context(), req.BaseRequest))
		})
	}
}

func Test_setInjectedAnnotation(t *testing.T) {
	t.Run("should add annotation to nil map", func(t *testing.T) {
		mut := NewMutator(nil)
		request := createTestMutationRequest(t, nil, nil)

		require.False(t, mut.IsInjected(t.Context(), request.BaseRequest))
		setInjectedAnnotation(request.Pod)
		require.Len(t, request.Pod.Annotations, 1)
		require.True(t, mut.IsInjected(t.Context(), request.BaseRequest))
	})

	t.Run("should remove reason from map", func(t *testing.T) {
		mut := NewMutator(nil)
		request := createTestMutationRequest(t, nil, nil)
		setNotInjectedAnnotationFunc("test")(request.Pod)

		require.False(t, mut.IsInjected(t.Context(), request.BaseRequest))
		setInjectedAnnotation(request.Pod)
		require.Len(t, request.Pod.Annotations, 1)
		require.True(t, mut.IsInjected(t.Context(), request.BaseRequest))
	})
}

func Test_setNotInjectedAnnotationFunc(t *testing.T) {
	t.Run("should add annotations to nil map", func(t *testing.T) {
		mut := NewMutator(nil)
		request := createTestMutationRequest(t, nil, nil)

		require.False(t, mut.IsInjected(t.Context(), request.BaseRequest))
		setNotInjectedAnnotationFunc("test")(request.Pod)
		require.Len(t, request.Pod.Annotations, 2)
		require.False(t, mut.IsInjected(t.Context(), request.BaseRequest))
	})
}

func TestMutate(t *testing.T) {
	pod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test",
			OwnerReferences: []metav1.OwnerReference{
				{
					Name:       "owner",
					APIVersion: "v1",
					Kind:       "ReplicationController",
					Controller: ptr.To(true),
				},
			},
		},
	}

	t.Run("metadata enrichment fails => error", func(t *testing.T) {
		request := dtwebhook.MutationRequest{
			Context: t.Context(),
			BaseRequest: &dtwebhook.BaseRequest{
				Pod: pod.DeepCopy(),
				DynaKube: dynakube.DynaKube{
					Spec: dynakube.DynaKubeSpec{
						MetadataEnrichment: metadataenrichment.Spec{
							Enabled: ptr.To(true),
						},
					},
				},
			},
		}
		mut := NewMutator(fake.NewClient())

		err := mut.Mutate(&request)
		require.Error(t, err)
	})
	t.Run("metadata enrichment passes => additional args and annotations", func(t *testing.T) {
		const (
			nsMetaAnnotationKey          = "meta-annotation-key"
			nsMetaAnnotationValue        = "meta-annotation-value"
			testSecContextLabel          = "test-security-context-label"
			testCostCenterAnnotation     = "test-cost-center-annotation"
			testCustomMetadataLabel      = "test-custom-metadata-label"
			testCustomMetadataAnnotation = "test-custom-metadata-annotation"
			testKubeSystemID             = "01234567-abcd-efgh-ijkl-987654321zyx"
			testClusterName              = "dynakube"
			testClusterMEID              = "KUBERNETES_CLUSTER-DE4AF78E24729521"
		)

		type testCase struct {
			name                      string
			annotations               map[string]string
			withDeprecatedAnnotations bool
		}

		testCases := []testCase{
			{
				name:                      "without deprecated annotations",
				annotations:               map[string]string{exp.EnrichmentEnableAttributesDTKubernetes: "false"},
				withDeprecatedAnnotations: false,
			},
			{
				name:                      "with deprecated annotations",
				annotations:               map[string]string{},
				withDeprecatedAnnotations: true,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				initContainer := corev1.Container{
					Args: []string{},
				}
				pod := pod.DeepCopy()

				owner := &corev1.ReplicationController{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "v1",
						Kind:       "ReplicationController",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "owner",
						Namespace: pod.Namespace,
					},
				}

				expectedPod := pod.DeepCopy()

				request := dtwebhook.MutationRequest{
					Context: t.Context(),
					BaseRequest: &dtwebhook.BaseRequest{
						Pod: pod,
						DynaKube: dynakube.DynaKube{
							ObjectMeta: metav1.ObjectMeta{
								Annotations: tc.annotations,
							},
							Spec: dynakube.DynaKubeSpec{
								MetadataEnrichment: metadataenrichment.Spec{
									Enabled: ptr.To(true),
								},
							},
							Status: dynakube.DynaKubeStatus{
								KubeSystemUUID:        testKubeSystemID,
								KubernetesClusterMEID: testClusterMEID,
								KubernetesClusterName: testClusterName,
								MetadataEnrichment: metadataenrichment.Status{
									Rules: []metadataenrichment.Rule{
										{
											Type:   "LABEL",
											Source: testSecContextLabel,
											Target: "dt.security_context",
										},
										{
											Type:   "LABEL",
											Source: testCustomMetadataLabel,
										},
										{
											Type:   "ANNOTATION",
											Source: testCostCenterAnnotation,
											Target: "dt.cost.costcenter",
										},
										{
											Type:   "ANNOTATION",
											Source: testCustomMetadataAnnotation,
										},
									},
								},
							},
						},
						Namespace: corev1.Namespace{
							ObjectMeta: metav1.ObjectMeta{
								Name: pod.Namespace,
								Annotations: map[string]string{
									metadataenrichment.Prefix + nsMetaAnnotationKey: nsMetaAnnotationValue,
									testCostCenterAnnotation:                        "cost-center",
									testCustomMetadataAnnotation:                    "custom-meta-annotation",
								},
								Labels: map[string]string{
									testSecContextLabel:     "high",
									testCustomMetadataLabel: "custom-meta-label",
								},
							},
						},
					},
					InstallContainer: &initContainer,
				}

				mut := NewMutator(fake.NewClient(owner, pod))

				err := mut.Mutate(&request)
				require.NoError(t, err)
				require.NotEqual(t, *expectedPod, *request.Pod)
				require.NotEmpty(t, request.Pod.OwnerReferences)

				assert.Contains(t, request.InstallContainer.Args, buildArgument("k8s.workload.kind", request.Pod.OwnerReferences[0].Kind))
				assert.Contains(t, request.InstallContainer.Args, buildArgument("k8s.workload.name", request.Pod.OwnerReferences[0].Name))
				assert.Contains(t, request.InstallContainer.Args, buildArgument(nsMetaAnnotationKey, nsMetaAnnotationValue))

				if tc.withDeprecatedAnnotations {
					assert.Contains(t, request.InstallContainer.Args, buildArgument(attributes.DeprecatedWorkloadKindKey, request.Pod.OwnerReferences[0].Kind))
					assert.Contains(t, request.InstallContainer.Args, buildArgument(attributes.DeprecatedWorkloadNameKey, request.Pod.OwnerReferences[0].Name))
				} else {
					assert.NotContains(t, request.InstallContainer.Args, buildArgument(attributes.DeprecatedWorkloadKindKey, request.Pod.OwnerReferences[0].Kind))
					assert.NotContains(t, request.InstallContainer.Args, buildArgument(attributes.DeprecatedWorkloadNameKey, request.Pod.OwnerReferences[0].Name))
				}

				assert.Contains(t, request.InstallContainer.Args, buildArgument("dt.security_context", "high"))
				assert.Contains(t, request.InstallContainer.Args, buildArgument("dt.cost.costcenter", "cost-center"))
				assert.Contains(t, request.InstallContainer.Args, buildArgument("k8s.namespace.label."+testCustomMetadataLabel, "custom-meta-label"))
				assert.Contains(t, request.InstallContainer.Args, buildArgument("k8s.namespace.annotation."+testCustomMetadataAnnotation, "custom-meta-annotation"))
				assert.Contains(t, request.InstallContainer.Args, "--"+bootstrapper.MetadataEnrichmentFlag)

				require.Len(t, request.Pod.Annotations, 7) // workload.kind + workload.name + dt.security_context + dt.cost.costcenter + injected + propagated ns annotations
				assert.Equal(t, strings.ToLower(request.Pod.OwnerReferences[0].Kind), request.Pod.Annotations[metadataenrichment.Prefix+attributes.K8sWorkloadKindAttr])
				assert.Equal(t, request.Pod.OwnerReferences[0].Name, request.Pod.Annotations[metadataenrichment.Prefix+attributes.K8sWorkloadNameAttr])
				assert.Equal(t, "true", request.Pod.Annotations[AnnotationInjected])
				assert.Equal(t, nsMetaAnnotationValue, request.Pod.Annotations[metadataenrichment.Prefix+nsMetaAnnotationKey])
				assert.NotEmpty(t, request.Pod.Annotations[metadataenrichment.Annotation])
			})
		}
	})
}

func buildArgument(attr string, value string) string {
	return fmt.Sprintf("--%s=%s=%s", podattr.Flag, attr, strings.ToLower(value))
}

func createTestMutationRequest(t *testing.T, dk *dynakube.DynaKube, annotations map[string]string) *dtwebhook.MutationRequest {
	if dk == nil {
		dk = &dynakube.DynaKube{}
	}

	return dtwebhook.NewMutationRequest(
		t.Context(),
		*getTestNamespace(dk),
		&corev1.Container{
			Name: dtwebhook.InstallContainerName,
		},
		getTestPod(annotations),
		*dk,
	)
}

func getTestNamespace(dk *dynakube.DynaKube) *corev1.Namespace {
	return &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-ns",
			Labels: map[string]string{
				dtwebhook.InjectionInstanceLabel: dk.Name,
			},
		},
	}
}

func getTestPod(annotations map[string]string) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "test-pod",
			Namespace:   "test-ns",
			Annotations: annotations,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "container-1",
					Image: "alpine-1",
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      "volume",
							MountPath: "/volume",
						},
					},
				},
				{
					Name:  "container-2",
					Image: "alpine-2",
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      "volume",
							MountPath: "/volume",
						},
					},
				},
			},
			InitContainers: []corev1.Container{
				{
					Name:  "init-container",
					Image: "alpine",
				},
			},
			Volumes: []corev1.Volume{
				{
					Name: "volume",
					VolumeSource: corev1.VolumeSource{
						EmptyDir: &corev1.EmptyDirVolumeSource{},
					},
				},
			},
		},
	}
}

func TestAddContainerAttributes(t *testing.T) {
	// request to pre-mount required volumes: OneAgent or Enrichment or both
	vmBaseRequest := &dtwebhook.BaseRequest{
		Pod: &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					dtwebhook.AnnotationInjectionSplitMounts: "false",
				},
			},
		},
		DynaKube: dynakube.DynaKube{
			Spec: dynakube.DynaKubeSpec{
				MetadataEnrichment: metadataenrichment.Spec{
					Enabled: ptr.To(true),
				},
				OneAgent: oneagent.Spec{
					ApplicationMonitoring: &oneagent.ApplicationMonitoringSpec{},
				},
			},
		},
	}

	validateContainerAttributes := func(t *testing.T, pod corev1.Pod, args []string) {
		t.Helper()

		require.NotEmpty(t, args)

		for _, arg := range args {
			splitArg := strings.Split(arg, "=")
			require.Len(t, splitArg, 2)

			var attr containerattr.Attributes

			require.NoError(t, json.Unmarshal([]byte(splitArg[1]), &attr))
			assert.Contains(t, pod.Spec.Containers, corev1.Container{
				Name:  attr.ContainerName,
				Image: attr.ToURI(),
				VolumeMounts: []corev1.VolumeMount{
					{
						Name:      volumes.ConfigVolumeName,
						MountPath: volumes.ConfigMountPath,
						SubPath:   attr.ContainerName,
					},
				},
			})
		}
	}

	t.Run("add container-attributes + mount", func(t *testing.T) {
		app1Container := corev1.Container{
			Name:  "app-1-name",
			Image: "registry1.example.com/repository/image:tag",
		}
		app2Container := corev1.Container{
			Name:  "app-2-name",
			Image: "registry2.example.com/repository/image:tag",
		}
		initContainer := corev1.Container{
			Args: []string{},
		}
		pod := corev1.Pod{
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					app1Container,
					app2Container,
				},
			},
		}

		request := dtwebhook.MutationRequest{
			BaseRequest: &dtwebhook.BaseRequest{
				Pod: &pod,
			},
			InstallContainer: &initContainer,
		}

		mutated, err := AddContainerAttributes(request.BaseRequest, &initContainer)
		require.NoError(t, err)
		assert.True(t, mutated)

		validateContainerAttributes(t, pod, initContainer.Args)
	})

	t.Run("no new container ==> no new arg", func(t *testing.T) {
		app1Container := corev1.Container{
			Name:  "app-1-name",
			Image: "registry1.example.com/repository/image:tag",
		}
		volumes.AddConfigVolumeMount(&app1Container, vmBaseRequest)

		app2Container := corev1.Container{
			Name:  "app-2-name",
			Image: "registry2.example.com/repository/image:tag",
		}
		volumes.AddConfigVolumeMount(&app2Container, vmBaseRequest)

		initContainer := corev1.Container{
			Args: []string{},
		}
		pod := corev1.Pod{
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					app1Container,
					app2Container,
				},
			},
		}

		request := dtwebhook.MutationRequest{
			BaseRequest: &dtwebhook.BaseRequest{
				Pod: &pod,
			},
			InstallContainer: &initContainer,
		}

		mutated, err := AddContainerAttributes(request.BaseRequest, &initContainer)

		require.NoError(t, err)
		assert.False(t, mutated)

		require.Empty(t, initContainer.Args)
	})

	t.Run("partially new => only add new", func(t *testing.T) {
		app1Container := corev1.Container{
			Name:  "app-1-name",
			Image: "registry1.example.com/repository/image:tag",
		}
		volumes.AddConfigVolumeMount(&app1Container, vmBaseRequest)

		app2Container := corev1.Container{
			Name:  "app-2-name",
			Image: "registry2.example.com/repository/image:tag",
		}

		initContainer := corev1.Container{
			Args: []string{},
		}
		pod := corev1.Pod{
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					app1Container,
					app2Container,
				},
			},
		}

		request := dtwebhook.MutationRequest{
			BaseRequest: &dtwebhook.BaseRequest{
				Pod: &pod,
			},
			InstallContainer: &initContainer,
		}

		mutated, err := AddContainerAttributes(request.BaseRequest, &initContainer)

		require.NoError(t, err)
		assert.True(t, mutated)

		require.Len(t, initContainer.Args, 1)
		validateContainerAttributes(t, pod, initContainer.Args)
	})
}

func TestAddContainerAttributesWithSplitVolumes(t *testing.T) {
	// request to pre-mount required volumes: OneAgent or Enrichment or both
	vmBaseRequest := func(metadataEnrichment bool, oneAgent bool) *dtwebhook.BaseRequest {
		br := &dtwebhook.BaseRequest{
			Pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						dtwebhook.AnnotationInjectionSplitMounts: "true",
					},
				},
			},
			DynaKube: dynakube.DynaKube{
				Spec: dynakube.DynaKubeSpec{
					MetadataEnrichment: metadataenrichment.Spec{
						Enabled: ptr.To(metadataEnrichment),
					},
				},
			},
		}
		if oneAgent {
			br.DynaKube.Spec.OneAgent = oneagent.Spec{
				ApplicationMonitoring: &oneagent.ApplicationMonitoringSpec{},
			}
		}

		return br
	}

	validateContainerAttributes := func(t *testing.T, pod corev1.Pod, args []string) {
		t.Helper()

		require.NotEmpty(t, args)

		for i := range pod.Spec.Containers {
			slices.SortFunc(pod.Spec.Containers[i].VolumeMounts, func(a, b corev1.VolumeMount) int {
				return strings.Compare(a.MountPath, b.MountPath)
			})
		}

		for _, arg := range args {
			splitArg := strings.Split(arg, "=")
			require.Len(t, splitArg, 2)

			var attr containerattr.Attributes

			require.NoError(t, json.Unmarshal([]byte(splitArg[1]), &attr))

			assert.Contains(t, pod.Spec.Containers, corev1.Container{
				Name:  attr.ContainerName,
				Image: attr.ToURI(),
				VolumeMounts: []corev1.VolumeMount{
					{
						Name:      volumes.ConfigVolumeName,
						MountPath: filepath.Join(volumes.ConfigMountPath, "enrichment", "dt_metadata.json"),
						SubPath:   filepath.Join(attr.ContainerName, "enrichment", "dt_metadata.json"),
					},
					{
						Name:      volumes.ConfigVolumeName,
						MountPath: filepath.Join(volumes.ConfigMountPath, "enrichment", "dt_metadata.properties"),
						SubPath:   filepath.Join(attr.ContainerName, "enrichment", "dt_metadata.properties"),
					},
					{
						Name:      volumes.ConfigVolumeName,
						MountPath: filepath.Join(volumes.ConfigMountPath, "enrichment", "endpoint"),
						SubPath:   filepath.Join(attr.ContainerName, "enrichment", "endpoint"),
					},
					{
						Name:      volumes.ConfigVolumeName,
						MountPath: filepath.Join(volumes.ConfigMountPath, "oneagent"),
						SubPath:   filepath.Join(attr.ContainerName, "oneagent"),
					},
				},
			})
		}
	}

	validateContainerAttributesforMetadataEnrichment := func(t *testing.T, pod corev1.Pod, args []string) {
		t.Helper()

		require.NotEmpty(t, args)

		for _, arg := range args {
			splitArg := strings.Split(arg, "=")
			require.Len(t, splitArg, 2)

			var attr containerattr.Attributes

			require.NoError(t, json.Unmarshal([]byte(splitArg[1]), &attr))

			assert.Contains(t, pod.Spec.Containers, corev1.Container{
				Name:  attr.ContainerName,
				Image: attr.ToURI(),
				VolumeMounts: []corev1.VolumeMount{
					{
						Name:      volumes.ConfigVolumeName,
						MountPath: filepath.Join(volumes.ConfigMountPath, "enrichment", "dt_metadata.json"),
						SubPath:   filepath.Join(attr.ContainerName, "enrichment", "dt_metadata.json"),
					},
					{
						Name:      volumes.ConfigVolumeName,
						MountPath: filepath.Join(volumes.ConfigMountPath, "enrichment", "dt_metadata.properties"),
						SubPath:   filepath.Join(attr.ContainerName, "enrichment", "dt_metadata.properties"),
					},
					{
						Name:      volumes.ConfigVolumeName,
						MountPath: filepath.Join(volumes.ConfigMountPath, "enrichment", "endpoint"),
						SubPath:   filepath.Join(attr.ContainerName, "enrichment", "endpoint"),
					},
				},
			})
		}
	}

	t.Run("add container-attributes + mount", func(t *testing.T) {
		app1Container := corev1.Container{
			Name:  "app-1-name",
			Image: "registry1.example.com/repository/image:tag",
		}
		app2Container := corev1.Container{
			Name:  "app-2-name",
			Image: "registry2.example.com/repository/image:tag",
		}
		initContainer := corev1.Container{
			Args: []string{},
		}
		pod := corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					dtwebhook.AnnotationInjectionSplitMounts: "true",
				},
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					app1Container,
					app2Container,
				},
			},
		}

		request := dtwebhook.MutationRequest{
			BaseRequest: &dtwebhook.BaseRequest{
				Pod: &pod,
				DynaKube: dynakube.DynaKube{
					Spec: dynakube.DynaKubeSpec{
						MetadataEnrichment: metadataenrichment.Spec{
							Enabled: ptr.To(true),
						},
						OneAgent: oneagent.Spec{
							ApplicationMonitoring: &oneagent.ApplicationMonitoringSpec{},
						},
					},
				},
			},
			InstallContainer: &initContainer,
		}

		mutated, err := AddContainerAttributes(request.BaseRequest, &initContainer)

		require.NoError(t, err)
		assert.True(t, mutated)

		validateContainerAttributes(t, pod, initContainer.Args)
	})

	t.Run("no new container ==> no new arg", func(t *testing.T) {
		app1Container := corev1.Container{
			Name:  "app-1-name",
			Image: "registry1.example.com/repository/image:tag",
		}
		volumes.AddConfigVolumeMount(&app1Container, vmBaseRequest(true, true))

		app2Container := corev1.Container{
			Name:  "app-2-name",
			Image: "registry2.example.com/repository/image:tag",
		}
		volumes.AddConfigVolumeMount(&app2Container, vmBaseRequest(true, true))

		initContainer := corev1.Container{
			Args: []string{},
		}
		pod := corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					dtwebhook.AnnotationInjectionSplitMounts: "true",
				},
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					app1Container,
					app2Container,
				},
			},
		}

		request := dtwebhook.MutationRequest{
			BaseRequest: &dtwebhook.BaseRequest{
				Pod: &pod,
				DynaKube: dynakube.DynaKube{
					Spec: dynakube.DynaKubeSpec{
						MetadataEnrichment: metadataenrichment.Spec{
							Enabled: ptr.To(true),
						},
						OneAgent: oneagent.Spec{
							ApplicationMonitoring: &oneagent.ApplicationMonitoringSpec{},
						},
					},
				},
			},
			InstallContainer: &initContainer,
		}

		mutated, err := AddContainerAttributes(request.BaseRequest, &initContainer)

		require.NoError(t, err)
		assert.False(t, mutated)

		require.Empty(t, initContainer.Args)
	})

	t.Run("partially new => only add new", func(t *testing.T) {
		app1Container := corev1.Container{
			Name:  "app-1-name",
			Image: "registry1.example.com/repository/image:tag",
		}
		volumes.AddConfigVolumeMount(&app1Container, vmBaseRequest(true, true))

		app2Container := corev1.Container{
			Name:  "app-2-name",
			Image: "registry2.example.com/repository/image:tag",
		}

		initContainer := corev1.Container{
			Args: []string{},
		}
		pod := corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					dtwebhook.AnnotationInjectionSplitMounts: "true",
				},
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					app1Container,
					app2Container,
				},
			},
		}

		request := dtwebhook.MutationRequest{
			BaseRequest: &dtwebhook.BaseRequest{
				Pod: &pod,
				DynaKube: dynakube.DynaKube{
					Spec: dynakube.DynaKubeSpec{
						MetadataEnrichment: metadataenrichment.Spec{
							Enabled: ptr.To(true),
						},
						OneAgent: oneagent.Spec{
							ApplicationMonitoring: &oneagent.ApplicationMonitoringSpec{},
						},
					},
				},
			},
			InstallContainer: &initContainer,
		}

		mutated, err := AddContainerAttributes(request.BaseRequest, &initContainer)

		require.NoError(t, err)
		assert.True(t, mutated)

		require.Len(t, initContainer.Args, 1)
		validateContainerAttributes(t, pod, initContainer.Args)
	})

	t.Run("partially new => add oneagent or enrichment", func(t *testing.T) {
		app1Container := corev1.Container{
			Name:  "app-1-name",
			Image: "registry1.example.com/repository/image:tag",
			VolumeMounts: []corev1.VolumeMount{
				{
					Name:      volumes.ConfigVolumeName,
					MountPath: filepath.Join(volumes.ConfigMountPath, "oneagent"),
					SubPath:   filepath.Join("app-1-name", "oneagent"),
				},
			},
		}

		app2Container := corev1.Container{
			Name:  "app-2-name",
			Image: "registry2.example.com/repository/image:tag",
		}
		volumes.AddConfigVolumeMount(&app2Container, vmBaseRequest(true, false))

		initContainer := corev1.Container{
			Args: []string{},
		}
		pod := corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					dtwebhook.AnnotationInjectionSplitMounts: "true",
				},
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					app1Container,
					app2Container,
				},
			},
		}

		request := dtwebhook.MutationRequest{
			BaseRequest: &dtwebhook.BaseRequest{
				Pod: &pod,
				DynaKube: dynakube.DynaKube{
					Spec: dynakube.DynaKubeSpec{
						MetadataEnrichment: metadataenrichment.Spec{
							Enabled: ptr.To(true),
						},
						OneAgent: oneagent.Spec{
							ApplicationMonitoring: &oneagent.ApplicationMonitoringSpec{},
						},
					},
				},
			},
			InstallContainer: &initContainer,
		}

		mutated, err := AddContainerAttributes(request.BaseRequest, &initContainer)

		require.NoError(t, err)
		assert.True(t, mutated)

		require.Len(t, initContainer.Args, 2)
		validateContainerAttributes(t, pod, initContainer.Args)
	})

	t.Run("partially new => add oneagent", func(t *testing.T) {
		app1Container := corev1.Container{
			Name:  "app-1-name",
			Image: "registry1.example.com/repository/image:tag",
		}

		app2Container := corev1.Container{
			Name:  "app-2-name",
			Image: "registry2.example.com/repository/image:tag",
		}

		initContainer := corev1.Container{
			Args: []string{},
		}
		pod := corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					dtwebhook.AnnotationInjectionSplitMounts: "true",
				},
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					app1Container,
					app2Container,
				},
			},
		}

		request := dtwebhook.MutationRequest{
			BaseRequest: &dtwebhook.BaseRequest{
				Pod: &pod,
				DynaKube: dynakube.DynaKube{
					Spec: dynakube.DynaKubeSpec{
						MetadataEnrichment: metadataenrichment.Spec{
							Enabled: ptr.To(false),
						},
						OneAgent: oneagent.Spec{
							ApplicationMonitoring: &oneagent.ApplicationMonitoringSpec{},
						},
					},
				},
			},
			InstallContainer: &initContainer,
		}

		mutated, err := AddContainerAttributes(request.BaseRequest, &initContainer)

		require.NoError(t, err)
		assert.True(t, mutated)

		require.Len(t, initContainer.Args, 2)
		validateContainerAttributes(t, pod, initContainer.Args)
	})

	t.Run("partially new => add enrichment", func(t *testing.T) {
		app1Container := corev1.Container{
			Name:  "app-1-name",
			Image: "registry1.example.com/repository/image:tag",
		}

		app2Container := corev1.Container{
			Name:  "app-2-name",
			Image: "registry2.example.com/repository/image:tag",
		}

		initContainer := corev1.Container{
			Args: []string{},
		}
		pod := corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					dtwebhook.AnnotationInjectionSplitMounts: "true",
				},
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					app1Container,
					app2Container,
				},
			},
		}

		request := dtwebhook.MutationRequest{
			BaseRequest: &dtwebhook.BaseRequest{
				Pod: &pod,
				DynaKube: dynakube.DynaKube{
					Spec: dynakube.DynaKubeSpec{
						MetadataEnrichment: metadataenrichment.Spec{
							Enabled: ptr.To(true),
						},
					},
				},
			},
			InstallContainer: &initContainer,
		}

		mutated, err := AddContainerAttributes(request.BaseRequest, &initContainer)

		require.NoError(t, err)
		assert.True(t, mutated)

		require.Len(t, initContainer.Args, 2)
		validateContainerAttributesforMetadataEnrichment(t, pod, initContainer.Args)
	})
}
