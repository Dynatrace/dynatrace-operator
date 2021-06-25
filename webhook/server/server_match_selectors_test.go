package server

import (
	"context"
	"encoding/json"
	"testing"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	"github.com/Dynatrace/dynatrace-operator/scheme"
	"github.com/Dynatrace/dynatrace-operator/scheme/fake"
	jsonpatch "github.com/evanphx/json-patch"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

func TestMatchLabels(t *testing.T) {
	decoder, err := admission.NewDecoder(scheme.Scheme)
	require.NoError(t, err)

	inj := &podInjector{
		client: fake.NewClient(
			&dynatracev1alpha1.DynaKube{
				ObjectMeta: metav1.ObjectMeta{Name: "dynakube", Namespace: "dynatrace"},
				Spec: dynatracev1alpha1.DynaKubeSpec{
					APIURL: "https://test-api-url.com/api",
					InfraMonitoring: dynatracev1alpha1.FullStackSpec{
						Enabled:           true,
						UseImmutableImage: true,
					},
					CodeModules: dynatracev1alpha1.CodeModulesSpec{
						Enabled: true,
						Selector: metav1.LabelSelector{
							MatchLabels: map[string]string{
								"inject":      "true",
								"also-inject": "true",
							},
						},
					},
				},
				Status: dynatracev1alpha1.DynaKubeStatus{
					OneAgent: dynatracev1alpha1.OneAgentStatus{
						UseImmutableImage: true,
					},
				},
			},
			&corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "dynatrace",
					Labels: map[string]string{
						"also-inject": "true",
					},
				},
			},
			&corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-namespace",
					Labels: map[string]string{
						"inject": "true",
					},
				},
			},
			&corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "another-namespace",
					Labels: map[string]string{
						"inject":      "true",
						"also-inject": "true",
					},
				},
			},
		),
		decoder:   decoder,
		image:     "test-api-url.com/linux/codemodule",
		namespace: "dynatrace",
	}

	pods := []corev1.Pod{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-pod-123456",
				Namespace: "test-namespace",
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{{
					Name:  "test-container",
					Image: "alpine",
				}},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-pod-123456",
				Namespace: "another-namespace",
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{{
					Name:  "test-container",
					Image: "alpine",
				}},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-pod-123456",
				Namespace: "dynatrace",
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{{
					Name:  "test-container",
					Image: "alpine",
				}},
			},
		},
	}

	for _, pod := range pods {
		basePodBytes, err := json.Marshal(&pod)
		require.NoError(t, err)

		req := admission.Request{
			AdmissionRequest: admissionv1.AdmissionRequest{
				Object: runtime.RawExtension{
					Raw: basePodBytes,
				},
				Namespace: pod.Namespace,
			},
		}

		// Only "another-namespace" should be injected, as it is the only one with a full match
		if pod.Namespace == "another-namespace" {
			updatedPod := podFromResponse(t, handleRequest(t, inj, req), basePodBytes)
			assert.Contains(t, updatedPod.Annotations, "oneagent.dynatrace.com/injected")
			assert.Equal(t, "true", updatedPod.Annotations["oneagent.dynatrace.com/injected"])
		} else {
			resp := inj.Handle(context.TODO(), req)
			require.NoError(t, resp.Complete(req))
			assert.True(t, resp.Allowed)

			patchType := admissionv1.PatchTypeJSONPatch
			assert.NotEqual(t, &patchType, resp.PatchType)
		}
	}
}

func TestMatchExpressions(t *testing.T) {
	decoder, err := admission.NewDecoder(scheme.Scheme)
	require.NoError(t, err)

	inj := &podInjector{
		client: fake.NewClient(
			&dynatracev1alpha1.DynaKube{
				ObjectMeta: metav1.ObjectMeta{Name: "dynakube-1", Namespace: "dynatrace"},
				Spec: dynatracev1alpha1.DynaKubeSpec{
					APIURL: "https://test-api-url.com/api",
					InfraMonitoring: dynatracev1alpha1.FullStackSpec{
						Enabled:           true,
						UseImmutableImage: true,
					},
					CodeModules: dynatracev1alpha1.CodeModulesSpec{
						Enabled: true,
						Selector: metav1.LabelSelector{
							MatchExpressions: []metav1.LabelSelectorRequirement{
								{Key: "app", Operator: metav1.LabelSelectorOpIn, Values: []string{"ui", "server"}},
							},
						},
					},
				},
				Status: dynatracev1alpha1.DynaKubeStatus{
					OneAgent: dynatracev1alpha1.OneAgentStatus{
						UseImmutableImage: true,
					},
				},
			},
			&dynatracev1alpha1.DynaKube{
				ObjectMeta: metav1.ObjectMeta{Name: "dynakube-2", Namespace: "dynatrace"},
				Spec: dynatracev1alpha1.DynaKubeSpec{
					APIURL: "https://test-api-url.com/api",
					InfraMonitoring: dynatracev1alpha1.FullStackSpec{
						Enabled:           true,
						UseImmutableImage: true,
					},
					CodeModules: dynatracev1alpha1.CodeModulesSpec{
						Enabled: true,
						Selector: metav1.LabelSelector{
							MatchLabels: map[string]string{
								"inject": "true",
							},
						},
					},
				},
				Status: dynatracev1alpha1.DynaKubeStatus{
					OneAgent: dynatracev1alpha1.OneAgentStatus{
						UseImmutableImage: true,
					},
				},
			},
			&corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "dynatrace",
					// Should be injected by dynatrace-2
					Labels: map[string]string{
						"inject": "true",
					},
				},
			},
			&corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-namespace",
					// Should be injected by dynatrace-1
					Labels: map[string]string{
						"some": "label",
						"app":  "ui",
					},
				},
			},
			&corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "another-namespace",
					// Should be injected by dynatrace-1
					Labels: map[string]string{
						"some": "label",
						"app":  "server",
					},
				},
			},
		),
		decoder:   decoder,
		image:     "test-api-url.com/linux/codemodule",
		namespace: "dynatrace",
	}

	pods := []corev1.Pod{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-pod-123456",
				Namespace: "test-namespace",
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{{
					Name:  "test-container",
					Image: "alpine",
				}},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-pod-123456",
				Namespace: "another-namespace",
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{{
					Name:  "test-container",
					Image: "alpine",
				}},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-pod-123456",
				Namespace: "dynatrace",
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{{
					Name:  "test-container",
					Image: "alpine",
				}},
			},
		},
	}

	for _, pod := range pods {
		basePodBytes, err := json.Marshal(&pod)
		require.NoError(t, err)

		req := admission.Request{
			AdmissionRequest: admissionv1.AdmissionRequest{
				Namespace: pod.Namespace,
				Object: runtime.RawExtension{
					Raw: basePodBytes,
				},
			},
		}

		updatedPod := podFromResponse(t, handleRequest(t, inj, req), basePodBytes)
		assert.Contains(t, updatedPod.Annotations, "oneagent.dynatrace.com/injected")
		assert.Equal(t, "true", updatedPod.Annotations["oneagent.dynatrace.com/injected"])
	}
}

func TestErrorOnMultipleMatchingCodeModules(t *testing.T) {
	decoder, err := admission.NewDecoder(scheme.Scheme)
	require.NoError(t, err)

	inj := &podInjector{
		client: fake.NewClient(
			&dynatracev1alpha1.DynaKube{
				ObjectMeta: metav1.ObjectMeta{Name: "dynakube-1", Namespace: "dynatrace"},
				Spec: dynatracev1alpha1.DynaKubeSpec{
					APIURL: "https://test-api-url.com/api",
					InfraMonitoring: dynatracev1alpha1.FullStackSpec{
						Enabled:           true,
						UseImmutableImage: true,
					},
					CodeModules: dynatracev1alpha1.CodeModulesSpec{
						Enabled: true,
						Selector: metav1.LabelSelector{
							MatchExpressions: []metav1.LabelSelectorRequirement{
								{Key: "app", Operator: metav1.LabelSelectorOpIn, Values: []string{"ui", "server"}},
							},
						},
					},
				},
				Status: dynatracev1alpha1.DynaKubeStatus{
					OneAgent: dynatracev1alpha1.OneAgentStatus{
						UseImmutableImage: true,
					},
				},
			},
			&dynatracev1alpha1.DynaKube{
				ObjectMeta: metav1.ObjectMeta{Name: "dynakube-2", Namespace: "dynatrace"},
				Spec: dynatracev1alpha1.DynaKubeSpec{
					APIURL: "https://test-api-url.com/api",
					InfraMonitoring: dynatracev1alpha1.FullStackSpec{
						Enabled:           true,
						UseImmutableImage: true,
					},
					CodeModules: dynatracev1alpha1.CodeModulesSpec{
						Enabled: true,
						Selector: metav1.LabelSelector{
							MatchLabels: map[string]string{
								"inject": "true",
							},
						},
					},
				},
				Status: dynatracev1alpha1.DynaKubeStatus{
					OneAgent: dynatracev1alpha1.OneAgentStatus{
						UseImmutableImage: true,
					},
				},
			},
			&corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "dynatrace",
				},
			},
			&corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-namespace",
					Labels: map[string]string{
						"some":   "label",
						"app":    "ui",
						"inject": "true",
					},
				},
			},
			&corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "another-namespace",
				},
			},
		),
		decoder:   decoder,
		image:     "test-api-url.com/linux/codemodule",
		namespace: "dynatrace",
	}

	pods := []corev1.Pod{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-pod-123456",
				Namespace: "test-namespace",
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{{
					Name:  "test-container",
					Image: "alpine",
				}},
			},
		},
	}

	for _, pod := range pods {
		basePodBytes, err := json.Marshal(&pod)
		require.NoError(t, err)

		req := admission.Request{
			AdmissionRequest: admissionv1.AdmissionRequest{
				Namespace: pod.Namespace,
				Object: runtime.RawExtension{
					Raw: basePodBytes,
				},
			},
		}

		resp := inj.Handle(context.TODO(), req)
		require.NoError(t, resp.Complete(req))
		assert.False(t, resp.Allowed)
	}
}

func podFromResponse(t *testing.T, resp admission.Response, basePodBytes []byte) corev1.Pod {
	patchType := admissionv1.PatchTypeJSONPatch
	assert.Equal(t, resp.PatchType, &patchType)

	patch, err := jsonpatch.DecodePatch(resp.Patch)
	require.NoError(t, err)

	updPodBytes, err := patch.Apply(basePodBytes)
	require.NoError(t, err)

	var updPod corev1.Pod
	require.NoError(t, json.Unmarshal(updPodBytes, &updPod))
	return updPod
}

func handleRequest(t *testing.T, injector *podInjector, req admission.Request) admission.Response {
	resp := injector.Handle(context.TODO(), req)
	require.NoError(t, resp.Complete(req))
	assert.True(t, resp.Allowed)
	return resp
}
