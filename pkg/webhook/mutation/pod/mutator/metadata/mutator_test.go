package metadata

import (
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
	maputils "github.com/Dynatrace/dynatrace-operator/pkg/util/map"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator"
	oacommon "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator/oneagent"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/workload"
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

func TestWorkloadAnnotations(t *testing.T) {
	workloadInfoName := "workload-name"
	workloadInfoKind := "workload-kind"

	t.Run("should add annotation to nil map", func(t *testing.T) {
		request := createTestMutationRequest(nil, nil)

		require.Equal(t, "not-found", maputils.GetField(request.Pod.Annotations, AnnotationWorkloadName, "not-found"))
		SetWorkloadAnnotations(request.Pod, &workload.Info{Name: workloadInfoName, Kind: workloadInfoKind})
		require.Len(t, request.Pod.Annotations, 2)
		assert.Equal(t, workloadInfoName, maputils.GetField(request.Pod.Annotations, AnnotationWorkloadName, "not-found"))
		assert.Equal(t, workloadInfoKind, maputils.GetField(request.Pod.Annotations, AnnotationWorkloadKind, "not-found"))
	})
	t.Run("should lower case kind annotation", func(t *testing.T) {
		request := createTestMutationRequest(nil, nil)
		objectMeta := &metav1.PartialObjectMetadata{
			ObjectMeta: metav1.ObjectMeta{Name: workloadInfoName},
			TypeMeta:   metav1.TypeMeta{Kind: "SuperWorkload"},
		}

		SetWorkloadAnnotations(request.Pod, workload.NewInfo(objectMeta))
		assert.Contains(t, request.Pod.Annotations, AnnotationWorkloadKind)
		assert.Equal(t, "superworkload", request.Pod.Annotations[AnnotationWorkloadKind])
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
		assert.Contains(t, request.InstallContainer.Args, buildArgument(DeprecatedWorkloadKindKey, request.Pod.OwnerReferences[0].Kind))
		assert.Contains(t, request.InstallContainer.Args, buildArgument(DeprecatedWorkloadNameKey, request.Pod.OwnerReferences[0].Name))
		assert.Contains(t, request.InstallContainer.Args, buildArgument(nsMetaAnnotationKey, nsMetaAnnotationValue))

		assert.Contains(t, request.InstallContainer.Args, buildArgument("dt.security_context", "high"))
		assert.Contains(t, request.InstallContainer.Args, buildArgument("dt.cost.costcenter", "cost-center"))
		assert.Contains(t, request.InstallContainer.Args, buildArgument("k8s.namespace.label."+testCustomMetadataLabel, "custom-meta-label"))
		assert.Contains(t, request.InstallContainer.Args, buildArgument("k8s.namespace.annotation."+testCustomMetadataAnnotation, "custom-meta-annotation"))
		assert.Contains(t, request.InstallContainer.Args, "--"+bootstrapper.MetadataEnrichmentFlag)

		require.Len(t, request.Pod.Annotations, 7) // workload.kind + workload.name + dt.security_context + dt.cost.costcenter + injected + propagated ns annotations
		assert.Equal(t, strings.ToLower(request.Pod.OwnerReferences[0].Kind), request.Pod.Annotations[AnnotationWorkloadKind])
		assert.Equal(t, request.Pod.OwnerReferences[0].Name, request.Pod.Annotations[AnnotationWorkloadName])
		assert.Equal(t, "true", request.Pod.Annotations[AnnotationInjected])
		assert.Equal(t, nsMetaAnnotationValue, request.Pod.Annotations[metadataenrichment.Prefix+nsMetaAnnotationKey])
		assert.NotEmpty(t, request.Pod.Annotations[metadataenrichment.Annotation])
	})
}

func buildArgument(attr string, value string) string {
	return fmt.Sprintf("--%s=%s=%s", podattr.Flag, attr, strings.ToLower(value))
}
