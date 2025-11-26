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
			nsMetaAnnotationKey   = "meta-annotation-key"
			nsMetaAnnotationValue = "meta-annotation-value"
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
				},
				Namespace: corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: pod.Namespace,
						Annotations: map[string]string{
							metadataenrichment.Prefix + nsMetaAnnotationKey: nsMetaAnnotationValue,
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

		kindAttr := fmt.Sprintf("--%s=%s=%s", podattr.Flag, "k8s.workload.kind", strings.ToLower(request.Pod.OwnerReferences[0].Kind))
		nameAttr := fmt.Sprintf("--%s=%s=%s", podattr.Flag, "k8s.workload.name", strings.ToLower(request.Pod.OwnerReferences[0].Name))
		depKindAttr := fmt.Sprintf("--%s=%s=%s", podattr.Flag, deprecatedWorkloadKindKey, strings.ToLower(request.Pod.OwnerReferences[0].Kind))
		depNameAttr := fmt.Sprintf("--%s=%s=%s", podattr.Flag, deprecatedWorkloadNameKey, strings.ToLower(request.Pod.OwnerReferences[0].Name))
		metaFromNsAttr := fmt.Sprintf("--%s=%s=%s", podattr.Flag, nsMetaAnnotationKey, nsMetaAnnotationValue)

		assert.Contains(t, request.InstallContainer.Args, kindAttr)
		assert.Contains(t, request.InstallContainer.Args, nameAttr)
		assert.Contains(t, request.InstallContainer.Args, depKindAttr)
		assert.Contains(t, request.InstallContainer.Args, depNameAttr)
		assert.Contains(t, request.InstallContainer.Args, metaFromNsAttr)
		assert.Contains(t, request.InstallContainer.Args, "--"+bootstrapper.MetadataEnrichmentFlag)

		require.Len(t, request.Pod.Annotations, 5) // workload.kind + workload.name + injected + propagated ns annotations
		assert.Equal(t, strings.ToLower(request.Pod.OwnerReferences[0].Kind), request.Pod.Annotations[workload.AnnotationWorkloadKind])
		assert.Equal(t, request.Pod.OwnerReferences[0].Name, request.Pod.Annotations[workload.AnnotationWorkloadName])
		assert.Equal(t, "true", request.Pod.Annotations[AnnotationInjected])
		assert.Equal(t, nsMetaAnnotationValue, request.Pod.Annotations[metadataenrichment.Prefix+nsMetaAnnotationKey])
		assert.NotEmpty(t, request.Pod.Annotations[metadataenrichment.Annotation])
	})
}
