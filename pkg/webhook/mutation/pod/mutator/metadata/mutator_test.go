package metadata

import (
	"context"
	"fmt"
	"strings"
	"testing"

	podattr "github.com/Dynatrace/dynatrace-bootstrapper/cmd/configure/attributes/pod"
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
			title:   "only OA enabled, without FF => not enabled",
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
			title:   "OA + FF enabled => enabled with no CSI",
			podMods: func(p *corev1.Pod) {},
			nsMods:  func(n *corev1.Namespace) {},
			dkMods: func(dk *dynakube.DynaKube) {
				dk.Spec.OneAgent.ApplicationMonitoring = &oneagent.ApplicationMonitoringSpec{}
				dk.Annotations = map[string]string{exp.OANodeImagePullKey: "true"}
				dk.Spec.MetadataEnrichment.Enabled = ptr.To(false)
			},
			withCSI:    false,
			withoutCSI: true,
		},
		{
			title: "OA + FF enabled + ephemeral Volume-Type => enabled",
			podMods: func(p *corev1.Pod) {
				p.Annotations = map[string]string{oacommon.AnnotationVolumeType: oacommon.EphemeralVolumeType}
			},
			nsMods: func(n *corev1.Namespace) {},
			dkMods: func(dk *dynakube.DynaKube) {
				dk.Spec.OneAgent.ApplicationMonitoring = &oneagent.ApplicationMonitoringSpec{}
				dk.Annotations = map[string]string{exp.OANodeImagePullKey: "true"}
			},
			withCSI:    true,
			withoutCSI: true,
		},
		{
			title: "OA + FF enabled + csi Volume-Type => disabled",
			podMods: func(p *corev1.Pod) {
				p.Annotations = map[string]string{oacommon.AnnotationVolumeType: oacommon.CSIVolumeType}
			},
			nsMods: func(n *corev1.Namespace) {},
			dkMods: func(dk *dynakube.DynaKube) {
				dk.Spec.OneAgent.ApplicationMonitoring = &oneagent.ApplicationMonitoringSpec{}
				dk.Annotations = map[string]string{exp.OANodeImagePullKey: "true"}
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

			assert.Equal(t, test.withCSI, mut.IsEnabled(req.BaseRequest))

			installconfig.SetModulesOverride(t, installconfig.Modules{CSIDriver: false})

			assert.Equal(t, test.withoutCSI, mut.IsEnabled(req.BaseRequest))
		})
	}
}

func Test_setInjectedAnnotation(t *testing.T) {
	t.Run("should add annotation to nil map", func(t *testing.T) {
		mut := NewMutator(nil)
		request := createTestMutationRequest(nil, nil)

		require.False(t, mut.IsInjected(request.BaseRequest))
		setInjectedAnnotation(request.Pod)
		require.Len(t, request.Pod.Annotations, 1)
		require.True(t, mut.IsInjected(request.BaseRequest))
	})

	t.Run("should remove reason from map", func(t *testing.T) {
		mut := NewMutator(nil)
		request := createTestMutationRequest(nil, nil)
		setNotInjectedAnnotationFunc("test")(request.Pod)

		require.False(t, mut.IsInjected(request.BaseRequest))
		setInjectedAnnotation(request.Pod)
		require.Len(t, request.Pod.Annotations, 1)
		require.True(t, mut.IsInjected(request.BaseRequest))
	})
}

func Test_setNotInjectedAnnotationFunc(t *testing.T) {
	t.Run("should add annotations to nil map", func(t *testing.T) {
		mut := NewMutator(nil)
		request := createTestMutationRequest(nil, nil)

		require.False(t, mut.IsInjected(request.BaseRequest))
		setNotInjectedAnnotationFunc("test")(request.Pod)
		require.Len(t, request.Pod.Annotations, 2)
		require.False(t, mut.IsInjected(request.BaseRequest))
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
			BaseRequest: &dtwebhook.BaseRequest{
				Pod: pod,
				DynaKube: dynakube.DynaKube{
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
		assert.Contains(t, request.InstallContainer.Args, buildArgument(attributes.DeprecatedWorkloadKindKey, request.Pod.OwnerReferences[0].Kind))
		assert.Contains(t, request.InstallContainer.Args, buildArgument(attributes.DeprecatedWorkloadNameKey, request.Pod.OwnerReferences[0].Name))
		assert.Contains(t, request.InstallContainer.Args, buildArgument(nsMetaAnnotationKey, nsMetaAnnotationValue))
		assert.Contains(t, request.InstallContainer.Args, buildArgument("dt.security_context", "high"))
		assert.Contains(t, request.InstallContainer.Args, buildArgument("dt.cost.costcenter", "cost-center"))
		assert.Contains(t, request.InstallContainer.Args, buildArgument("k8s.namespace.label."+testCustomMetadataLabel, "custom-meta-label"))
		assert.Contains(t, request.InstallContainer.Args, buildArgument("k8s.namespace.annotation."+testCustomMetadataAnnotation, "custom-meta-annotation"))
		assert.Contains(t, request.InstallContainer.Args, "--"+bootstrapper.MetadataEnrichmentFlag)

		/* TODO: these are not added by the mutator but by the metadata injection handler, should this be unified ot one place
		   - --attribute-container={"container_image.registry":"docker.io","container_image.repository":"nginx","container_image.tags":"latest","k8s.container.name":"app"}
		   - --attribute=k8s.pod.uid=$(K8S_PODUID)
		   - --attribute=k8s.pod.name=$(K8S_PODNAME)
		   - --attribute=k8s.node.name=$(K8S_NODE_NAME)
		   - --attribute=k8s.namespace.name=default
		   - --attribute=k8s.cluster.uid=01f0f32f-fd74-443d-975f-e46a7635db27
		   - --attribute=k8s.cluster.name=dynakube
		   - --attribute=dt.entity.kubernetes_cluster=KUBERNETES_CLUSTER-DE4AF78E24729521
		   - --attribute=dt.kubernetes.cluster.id=01f0f32f-fd74-443d-975f-e46a7635db27
		*/

		require.Len(t, request.Pod.Annotations, 7) // workload.kind + workload.name + dt.security_context + dt.cost.costcenter + injected + propagated ns annotations
		assert.Equal(t, strings.ToLower(request.Pod.OwnerReferences[0].Kind), request.Pod.Annotations[attributes.AnnotationWorkloadKind])
		assert.Equal(t, request.Pod.OwnerReferences[0].Name, request.Pod.Annotations[attributes.AnnotationWorkloadName])
		assert.Equal(t, "true", request.Pod.Annotations[AnnotationInjected])
		assert.Equal(t, nsMetaAnnotationValue, request.Pod.Annotations[metadataenrichment.Prefix+nsMetaAnnotationKey])
		assert.NotEmpty(t, request.Pod.Annotations[metadataenrichment.Annotation])
	})
}

func buildArgument(attr string, value string) string {
	return fmt.Sprintf("--%s=%s=%s", podattr.Flag, attr, strings.ToLower(value))
}

func createTestMutationRequest(dk *dynakube.DynaKube, annotations map[string]string) *dtwebhook.MutationRequest {
	if dk == nil {
		dk = &dynakube.DynaKube{}
	}

	return dtwebhook.NewMutationRequest(
		context.Background(),
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
