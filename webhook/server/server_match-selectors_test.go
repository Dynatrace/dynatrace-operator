package server

import (
	"context"
	"encoding/json"
	"testing"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	"github.com/Dynatrace/dynatrace-operator/scheme"
	"github.com/Dynatrace/dynatrace-operator/scheme/fake"
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
				Labels: map[string]string{
					"inject": "true",
				},
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{{
					Name:  "test-container",
					Image: "alpine",
				}},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Name: "test-pod-123456", Namespace: "another-namespace"},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{{
					Name:  "test-container",
					Image: "alpine",
				}},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Name: "test-pod-123456", Namespace: "dynatrace"},
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
				//Namespace: "test-namespace",
			},
		}

		resp := inj.Handle(context.TODO(), req)
		require.NoError(t, resp.Complete(req))
		assert.True(t, resp.Allowed)
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
				},
			},
			&corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-namespace",
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
				// Should be injected by dynatrace-1
				Labels: map[string]string{
					"some": "label",
					"app":  "ui",
				},
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
				// Should be injected by dynatrace-1
				Labels: map[string]string{
					"some": "label",
					"app":  "server",
				},
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
				// Should be injected by dynatrace-2
				Labels: map[string]string{
					"inject": "true",
				},
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
				//Namespace: "test-namespace",
			},
		}

		resp := inj.Handle(context.TODO(), req)
		require.NoError(t, resp.Complete(req))
		assert.True(t, resp.Allowed)
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
				// Should throw error cause both DynaKubes match
				Labels: map[string]string{
					"some":   "label",
					"app":    "ui",
					"inject": "true",
				},
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
			},
		}

		resp := inj.Handle(context.TODO(), req)
		require.NoError(t, resp.Complete(req))
		assert.False(t, resp.Allowed)
	}
}

func TestFindCodeModules(t *testing.T) {
	decoder, err := admission.NewDecoder(scheme.Scheme)
	require.NoError(t, err)

	instances := []dynatracev1alpha1.DynaKube{
		{
			ObjectMeta: metav1.ObjectMeta{Name: "codeModules-1", Namespace: "dynatrace"},
			Spec: dynatracev1alpha1.DynaKubeSpec{
				CodeModules: dynatracev1alpha1.CodeModulesSpec{
					Enabled: true,
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Name: "codeModules-2", Namespace: "dynatrace"},
			Spec: dynatracev1alpha1.DynaKubeSpec{
				CodeModules: dynatracev1alpha1.CodeModulesSpec{
					Enabled: true,
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Name: "other", Namespace: "dynatrace"},
			Spec: dynatracev1alpha1.DynaKubeSpec{
				CodeModules: dynatracev1alpha1.CodeModulesSpec{
					Enabled: false,
				},
			},
		},
	}

	inj := &podInjector{
		client: fake.NewClient(
			&instances[0],
			&instances[1],
		),
		decoder:   decoder,
		image:     "test-image",
		namespace: "dynatrace",
	}

	codeModules, err := FindCodeModules(context.TODO(), inj.client)
	assert.NoError(t, err)
	assert.NotNil(t, codeModules)
	assert.Equal(t, 2, len(codeModules))
}
