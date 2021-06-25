package server

import (
	"encoding/json"
	"testing"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	"github.com/Dynatrace/dynatrace-operator/scheme"
	"github.com/Dynatrace/dynatrace-operator/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/webhook"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	admissionv1 "k8s.io/api/admission/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

const (
	testNamespace = "test-namespace"
)

func TestInjectAnnotation(t *testing.T) {
	decoder, err := admission.NewDecoder(scheme.Scheme)
	require.NoError(t, err)

	injector := &podInjector{
		client: fake.NewClient(
			&v1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-namespace",
					Labels: map[string]string{
						"inject": "true",
					},
				},
			},
			&dynatracev1alpha1.DynaKube{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "dynakube",
					Namespace: "dynatrace",
				},
				Spec: dynatracev1alpha1.DynaKubeSpec{
					CodeModules: dynatracev1alpha1.CodeModulesSpec{
						Enabled: true,
						Selector: metav1.LabelSelector{
							MatchLabels: map[string]string{
								"inject": "true",
							},
						},
					},
				},
			}),
		decoder: decoder,
	}

	t.Run(`Do not inject if inject annotation is false`, func(t *testing.T) {
		pod := v1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-pod-123456",
				Namespace: testNamespace,
				Annotations: map[string]string{
					webhook.AnnotationInject: "false",
				},
			},
			Spec: v1.PodSpec{
				Containers: []v1.Container{{
					Name:  "test-container",
					Image: "alpine",
				}},
			},
		}

		basePodBytes, err := json.Marshal(&pod)
		require.NoError(t, err)

		req := admission.Request{
			AdmissionRequest: admissionv1.AdmissionRequest{
				Namespace: testNamespace,
				Object: runtime.RawExtension{
					Raw: basePodBytes,
				},
			},
		}

		resp := handleRequest(t, injector, req)
		assert.Nil(t, resp.Patches)
	})
}
