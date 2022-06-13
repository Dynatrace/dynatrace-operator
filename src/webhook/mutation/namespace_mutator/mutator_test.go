package namespace_mutator

import (
	"context"
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/mapper"
	"github.com/Dynatrace/dynatrace-operator/src/scheme/fake"
	jsonpatch "github.com/evanphx/json-patch"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/json"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

func TestInjection(t *testing.T) {
	dk := &dynatracev1beta1.DynaKube{
		ObjectMeta: metav1.ObjectMeta{Name: "codeModules-1", Namespace: "dynatrace"},
		Spec: dynatracev1beta1.DynaKubeSpec{
			NamespaceSelector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					"inject": "true",
				},
			},
			OneAgent: dynatracev1beta1.OneAgentSpec{
				ApplicationMonitoring: &dynatracev1beta1.ApplicationMonitoringSpec{
					AppInjectionSpec: dynatracev1beta1.AppInjectionSpec{},
				},
			},
		},
	}
	baseNs := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-namespace",
			Labels: map[string]string{
				"inject": "true",
			},
		},
	}
	clt := fake.NewClient(dk)
	inj := &namespaceMutator{
		client:    clt,
		apiReader: clt,
		namespace: "dynatrace",
	}
	t.Run("Don't inject into operator ns", func(t *testing.T) {
		baseNs := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: inj.namespace,
				Labels: map[string]string{
					"inject": "true",
				},
			},
		}
		baseNsBytes, err := json.Marshal(&baseNs)
		require.NoError(t, err)

		req := admission.Request{
			AdmissionRequest: admissionv1.AdmissionRequest{
				Object:    runtime.RawExtension{Raw: baseNsBytes},
				Name:      baseNs.Name,
				Namespace: baseNs.Name,
				Operation: admissionv1.Create,
			},
		}
		resp := inj.Handle(context.TODO(), req)
		assert.NoError(t, resp.Complete(req))
		assert.True(t, resp.Allowed)

		_, err = jsonpatch.DecodePatch(resp.Patch)
		assert.Error(t, err)
	})

	t.Run("Don't inject into namespace not matching dynakube", func(t *testing.T) {
		baseNs := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: inj.namespace,
			},
		}
		baseNsBytes, err := json.Marshal(&baseNs)
		require.NoError(t, err)

		req := admission.Request{
			AdmissionRequest: admissionv1.AdmissionRequest{
				Object:    runtime.RawExtension{Raw: baseNsBytes},
				Name:      baseNs.Name,
				Namespace: baseNs.Name,
				Operation: admissionv1.Create,
			},
		}
		resp := inj.Handle(context.TODO(), req)
		assert.NoError(t, resp.Complete(req))
		assert.True(t, resp.Allowed)

		_, err = jsonpatch.DecodePatch(resp.Patch)
		assert.Error(t, err)
	})

	t.Run("Create", func(t *testing.T) {
		baseNsBytes, err := json.Marshal(&baseNs)
		require.NoError(t, err)

		req := admission.Request{
			AdmissionRequest: admissionv1.AdmissionRequest{
				Object:    runtime.RawExtension{Raw: baseNsBytes},
				Name:      baseNs.Name,
				Namespace: baseNs.Name,
				Operation: admissionv1.Create,
			},
		}
		resp := inj.Handle(context.TODO(), req)
		assert.NoError(t, resp.Complete(req))
		assert.True(t, resp.Allowed)

		patch, err := jsonpatch.DecodePatch(resp.Patch)
		assert.NoError(t, err)

		updNsBytes, err := patch.Apply(baseNsBytes)
		assert.NoError(t, err)

		var updNs corev1.Namespace
		assert.NoError(t, json.Unmarshal(updNsBytes, &updNs))

		dkName, ok := updNs.Labels[mapper.InstanceLabel]
		assert.True(t, ok)
		assert.Equal(t, dk.Name, dkName)
	})

	t.Run("Update", func(t *testing.T) {
		baseNs := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-namespace",
				Labels: map[string]string{
					"inject": "true",
				},
			},
		}
		baseNsBytes, err := json.Marshal(&baseNs)
		require.NoError(t, err)

		req := admission.Request{
			AdmissionRequest: admissionv1.AdmissionRequest{
				Object:    runtime.RawExtension{Raw: baseNsBytes},
				Name:      baseNs.Name,
				Namespace: baseNs.Name,
				Operation: admissionv1.Update,
			},
		}
		resp := inj.Handle(context.TODO(), req)
		assert.NoError(t, resp.Complete(req))
		assert.True(t, resp.Allowed)

		patch, err := jsonpatch.DecodePatch(resp.Patch)
		assert.NoError(t, err)

		updNsBytes, err := patch.Apply(baseNsBytes)
		assert.NoError(t, err)

		var updNs corev1.Namespace
		assert.NoError(t, json.Unmarshal(updNsBytes, &updNs))

		dkName, ok := updNs.Labels[mapper.InstanceLabel]
		assert.True(t, ok)
		assert.Equal(t, dk.Name, dkName)
		assert.Equal(t, 2, len(updNs.Labels))
	})

	t.Run("Remove stale", func(t *testing.T) {
		baseNs := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-namespace",
				Labels: map[string]string{
					"inject":             "true",
					mapper.InstanceLabel: "stale",
				},
			},
		}
		baseNsBytes, err := json.Marshal(&baseNs)
		require.NoError(t, err)

		req := admission.Request{
			AdmissionRequest: admissionv1.AdmissionRequest{
				Object:    runtime.RawExtension{Raw: baseNsBytes},
				Name:      baseNs.Name,
				Namespace: baseNs.Name,
				Operation: admissionv1.Update,
			},
		}
		resp := inj.Handle(context.TODO(), req)
		assert.NoError(t, resp.Complete(req))
		assert.True(t, resp.Allowed)

		patch, err := jsonpatch.DecodePatch(resp.Patch)
		assert.NoError(t, err)

		updNsBytes, err := patch.Apply(baseNsBytes)
		assert.NoError(t, err)

		var updNs corev1.Namespace
		assert.NoError(t, json.Unmarshal(updNsBytes, &updNs))

		dkName, ok := updNs.Labels[mapper.InstanceLabel]
		assert.True(t, ok)
		assert.Equal(t, dk.Name, dkName)
		assert.Equal(t, 2, len(updNs.Labels))
	})
}
