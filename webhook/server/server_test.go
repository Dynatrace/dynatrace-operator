package server

import (
	"context"
	"testing"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	dtcsi "github.com/Dynatrace/dynatrace-operator/controllers/csi"
	"github.com/Dynatrace/dynatrace-operator/dtclient"
	"github.com/Dynatrace/dynatrace-operator/scheme"
	"github.com/Dynatrace/dynatrace-operator/scheme/fake"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/webhook"
	jsonpatch "github.com/evanphx/json-patch"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

const (
	testVersion = "test-version"
)

func TestInjectionWithMissingOneAgentAPM(t *testing.T) {
	decoder, err := admission.NewDecoder(scheme.Scheme)
	require.NoError(t, err)

	inj := &podInjector{
		client: fake.NewClient(
			&corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name:   "test-namespace",
					Labels: map[string]string{"oneagent.dynatrace.com/instance": "dynakube"},
				},
			}),
		decoder:   decoder,
		image:     "operator-image",
		namespace: "dynatrace",
		recorder:  record.NewFakeRecorder(10),
	}

	basePod := corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "test-pod-123456", Namespace: "test-namespace"}}
	basePodBytes, err := json.Marshal(&basePod)
	require.NoError(t, err)

	req := admission.Request{
		AdmissionRequest: admissionv1.AdmissionRequest{
			Object:    runtime.RawExtension{Raw: basePodBytes},
			Namespace: "test-namespace",
		},
	}
	resp := inj.Handle(context.TODO(), req)
	require.NoError(t, resp.Complete(req))
	require.False(t, resp.Allowed)
	require.Equal(t, resp.Result.Message, "namespace 'test-namespace' is assigned to DynaKube instance 'dynakube' but doesn't exist")
}

func createPodInjector(_ *testing.T, decoder *admission.Decoder) (*podInjector, *dynatracev1alpha1.DynaKube) {
	dynakube := &dynatracev1alpha1.DynaKube{
		ObjectMeta: metav1.ObjectMeta{Name: "oneagent", Namespace: "dynatrace"},
		Spec: dynatracev1alpha1.DynaKubeSpec{
			APIURL: "https://test-api-url.com/api",
			InfraMonitoring: dynatracev1alpha1.FullStackSpec{
				Enabled:           true,
				UseImmutableImage: true,
			},
			CodeModules: dynatracev1alpha1.CodeModulesSpec{
				Enabled: true,
				Resources: corev1.ResourceRequirements{
					Limits: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("1"),
						corev1.ResourceMemory: resource.MustParse("500M"),
					},
					Requests: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("100m"),
						corev1.ResourceMemory: resource.MustParse("100M"),
					},
				},
			},
		},
		Status: dynatracev1alpha1.DynaKubeStatus{
			OneAgent: dynatracev1alpha1.OneAgentStatus{
				UseImmutableImage: true,
			},
		},
	}

	return &podInjector{
		client: fake.NewClient(
			dynakube,
			&corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name:   "test-namespace",
					Labels: map[string]string{"oneagent.dynatrace.com/instance": "oneagent"},
				},
			},
		),
		decoder:   decoder,
		image:     "test-api-url.com/linux/codemodule",
		namespace: "dynatrace",
		recorder:  record.NewFakeRecorder(10),
	}, dynakube
}

func TestPodInjection(t *testing.T) {
	decoder, err := admission.NewDecoder(scheme.Scheme)
	require.NoError(t, err)

	inj, instance := createPodInjector(t, decoder)
	instance.Spec.CodeModules.Volume = corev1.VolumeSource{
		EmptyDir: &corev1.EmptyDirVolumeSource{},
	}
	err = inj.client.Update(context.TODO(), instance)
	require.NoError(t, err)

	basePod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "test-pod-12345", Namespace: "test-namespace"},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{{
				Name:  "test-container",
				Image: "alpine",
			}},
		},
	}
	basePodBytes, err := json.Marshal(&basePod)
	require.NoError(t, err)

	req := admission.Request{
		AdmissionRequest: admissionv1.AdmissionRequest{
			Object: runtime.RawExtension{
				Raw: basePodBytes,
			},
			Namespace: "test-namespace",
		},
	}
	resp := inj.Handle(context.TODO(), req)
	require.NoError(t, resp.Complete(req))

	if !resp.Allowed {
		require.FailNow(t, "failed to inject", resp.Result)
	}

	patchType := admissionv1.PatchTypeJSONPatch
	assert.Equal(t, resp.PatchType, &patchType)

	patch, err := jsonpatch.DecodePatch(resp.Patch)
	require.NoError(t, err)

	updPodBytes, err := patch.Apply(basePodBytes)
	require.NoError(t, err)

	var updPod corev1.Pod
	require.NoError(t, json.Unmarshal(updPodBytes, &updPod))

	expected := buildResultPod(t)

	expected.Spec.Volumes[0] = corev1.Volume{
		Name: "oneagent-bin",
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{},
		},
	}

	setEnvVar(t, &expected, "MODE", "installer")

	expected.Spec.InitContainers[0].Image = "test-api-url.com/linux/codemodule"

	expected.Spec.InitContainers[0].Resources = corev1.ResourceRequirements{
		Limits: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("1"),
			corev1.ResourceMemory: resource.MustParse("500M"),
		},
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("100m"),
			corev1.ResourceMemory: resource.MustParse("100M"),
		},
	}

	assert.Equal(t, expected, updPod)
}

func TestPodInjectionWithCSI(t *testing.T) {
	decoder, err := admission.NewDecoder(scheme.Scheme)
	require.NoError(t, err)

	inj, _ := createPodInjector(t, decoder)

	basePod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "test-pod-12345", Namespace: "test-namespace"},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{{
				Name:  "test-container",
				Image: "alpine",
			}},
		},
	}
	basePodBytes, err := json.Marshal(&basePod)
	require.NoError(t, err)

	req := admission.Request{
		AdmissionRequest: admissionv1.AdmissionRequest{
			Object: runtime.RawExtension{
				Raw: basePodBytes,
			},
			Namespace: "test-namespace",
		},
	}
	resp := inj.Handle(context.TODO(), req)
	require.NoError(t, resp.Complete(req))

	if !resp.Allowed {
		require.FailNow(t, "failed to inject", resp.Result)
	}

	patchType := admissionv1.PatchTypeJSONPatch
	assert.Equal(t, resp.PatchType, &patchType)

	patch, err := jsonpatch.DecodePatch(resp.Patch)
	require.NoError(t, err)

	updPodBytes, err := patch.Apply(basePodBytes)
	require.NoError(t, err)

	var updPod corev1.Pod
	require.NoError(t, json.Unmarshal(updPodBytes, &updPod))

	expected := buildResultPod(t)

	expected.Spec.InitContainers[0].Image = "test-api-url.com/linux/codemodule"

	expected.Spec.InitContainers[0].Resources = corev1.ResourceRequirements{
		Limits: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("1"),
			corev1.ResourceMemory: resource.MustParse("500M"),
		},
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("100m"),
			corev1.ResourceMemory: resource.MustParse("100M"),
		},
	}

	assert.Equal(t, expected, updPod)
}

func createDynakubeInstance(_ *testing.T) *dynatracev1alpha1.DynaKube {
	instance := &dynatracev1alpha1.DynaKube{
		ObjectMeta: metav1.ObjectMeta{Name: "oneagent", Namespace: "dynatrace"},
		Spec: dynatracev1alpha1.DynaKubeSpec{
			InfraMonitoring: dynatracev1alpha1.FullStackSpec{
				Enabled:           true,
				UseImmutableImage: true,
			},
			CodeModules: dynatracev1alpha1.CodeModulesSpec{
				Enabled: true,
			},
		},
		Status: dynatracev1alpha1.DynaKubeStatus{
			OneAgent: dynatracev1alpha1.OneAgentStatus{
				UseImmutableImage: true,
			},
		},
	}

	return instance
}

func withAgentVersion(_ *testing.T, instance *dynatracev1alpha1.DynaKube, version string) {
	instance.Spec.OneAgent = dynatracev1alpha1.OneAgentSpec{
		Version: version,
	}
}

func withoutCSIDriver(_ *testing.T, instance *dynatracev1alpha1.DynaKube) {
	instance.Spec.CodeModules.Volume = corev1.VolumeSource{
		EmptyDir: &corev1.EmptyDirVolumeSource{},
	}
}

func TestUseImmutableImage(t *testing.T) {
	t.Run(`do not use immutable image`, func(t *testing.T) {
		decoder, err := admission.NewDecoder(scheme.Scheme)
		require.NoError(t, err)

		instance := createDynakubeInstance(t)
		withoutCSIDriver(t, instance)

		inj := &podInjector{
			client: fake.NewClient(
				instance,
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name:   "test-namespace",
						Labels: map[string]string{"oneagent.dynatrace.com/instance": "oneagent"},
					},
				},
			),
			decoder:   decoder,
			image:     "test-image",
			namespace: "dynatrace",
			recorder:  record.NewFakeRecorder(10),
		}

		basePod := corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-pod-12345",
				Namespace: "test-namespace",
				Annotations: map[string]string{
					"oneagent.dynatrace.com/image": "customregistry/linux/codemodule",
				}},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{{
					Name:  "test-container",
					Image: "alpine",
				}},
			},
		}
		basePodBytes, err := json.Marshal(&basePod)
		require.NoError(t, err)

		req := admission.Request{
			AdmissionRequest: admissionv1.AdmissionRequest{
				Object: runtime.RawExtension{
					Raw: basePodBytes,
				},
				Namespace: "test-namespace",
			},
		}
		resp := inj.Handle(context.TODO(), req)
		require.NoError(t, resp.Complete(req))

		if !resp.Allowed {
			require.FailNow(t, "failed to inject", resp.Result)
		}

		patchType := admissionv1.PatchTypeJSONPatch
		assert.Equal(t, resp.PatchType, &patchType)

		patch, err := jsonpatch.DecodePatch(resp.Patch)
		require.NoError(t, err)

		updPodBytes, err := patch.Apply(basePodBytes)
		require.NoError(t, err)

		var updPod corev1.Pod
		require.NoError(t, json.Unmarshal(updPodBytes, &updPod))

		expected := buildResultPod(t)
		expected.Spec.Volumes[0] = corev1.Volume{
			Name: "oneagent-bin",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		}
		setEnvVar(t, &expected, "MODE", "installer")

		expected.ObjectMeta.Annotations["oneagent.dynatrace.com/image"] = "customregistry/linux/codemodule"

		assert.Equal(t, expected, updPod)
	})
	t.Run(`use immutable image`, func(t *testing.T) {
		decoder, err := admission.NewDecoder(scheme.Scheme)
		require.NoError(t, err)

		instance := createDynakubeInstance(t)
		withoutCSIDriver(t, instance)

		inj := &podInjector{
			client: fake.NewClient(
				instance,
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name:   "test-namespace",
						Labels: map[string]string{"oneagent.dynatrace.com/instance": "oneagent"},
					},
				},
			),
			decoder:   decoder,
			image:     "test-image",
			namespace: "dynatrace",
			recorder:  record.NewFakeRecorder(10),
		}

		basePod := corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-pod-12345",
				Namespace: "test-namespace",
				Annotations: map[string]string{
					"oneagent.dynatrace.com/image": "customregistry/linux/codemodule",
				}},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{{
					Name:  "test-container",
					Image: "alpine",
				}},
			},
		}
		basePodBytes, err := json.Marshal(&basePod)
		require.NoError(t, err)

		req := admission.Request{
			AdmissionRequest: admissionv1.AdmissionRequest{
				Object: runtime.RawExtension{
					Raw: basePodBytes,
				},
				Namespace: "test-namespace",
			},
		}
		resp := inj.Handle(context.TODO(), req)
		require.NoError(t, resp.Complete(req))

		if !resp.Allowed {
			require.FailNow(t, "failed to inject", resp.Result)
		}

		patchType := admissionv1.PatchTypeJSONPatch
		assert.Equal(t, resp.PatchType, &patchType)

		patch, err := jsonpatch.DecodePatch(resp.Patch)
		require.NoError(t, err)

		updPodBytes, err := patch.Apply(basePodBytes)
		require.NoError(t, err)

		var updPod corev1.Pod
		require.NoError(t, json.Unmarshal(updPodBytes, &updPod))

		expected := buildResultPod(t)
		expected.Spec.Volumes[0] = corev1.Volume{
			Name: "oneagent-bin",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		}
		setEnvVar(t, &expected, "MODE", "installer")

		expected.ObjectMeta.Annotations["oneagent.dynatrace.com/image"] = "customregistry/linux/codemodule"

		assert.Equal(t, expected, updPod)
	})

	t.Run(`honor custom image name`, func(t *testing.T) {
		decoder, err := admission.NewDecoder(scheme.Scheme)
		require.NoError(t, err)

		instance := createDynakubeInstance(t)
		withoutCSIDriver(t, instance)

		inj := &podInjector{
			client: fake.NewClient(
				instance,
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name:   "test-namespace",
						Labels: map[string]string{"oneagent.dynatrace.com/instance": "oneagent"},
					},
				},
			),
			decoder:   decoder,
			image:     "test-image",
			namespace: "dynatrace",
			recorder:  record.NewFakeRecorder(10),
		}

		basePod := corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:        "test-pod-12345",
				Namespace:   "test-namespace",
				Annotations: map[string]string{}},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{{
					Name:  "test-container",
					Image: "alpine",
				}},
			},
		}
		basePodBytes, err := json.Marshal(&basePod)
		require.NoError(t, err)

		req := admission.Request{
			AdmissionRequest: admissionv1.AdmissionRequest{
				Object: runtime.RawExtension{
					Raw: basePodBytes,
				},
				Namespace: "test-namespace",
			},
		}
		resp := inj.Handle(context.TODO(), req)
		require.NoError(t, resp.Complete(req))

		if !resp.Allowed {
			require.FailNow(t, "failed to inject", resp.Result)
		}

		patchType := admissionv1.PatchTypeJSONPatch
		assert.Equal(t, resp.PatchType, &patchType)

		patch, err := jsonpatch.DecodePatch(resp.Patch)
		require.NoError(t, err)

		updPodBytes, err := patch.Apply(basePodBytes)
		require.NoError(t, err)

		var updPod corev1.Pod
		require.NoError(t, json.Unmarshal(updPodBytes, &updPod))

		expected := buildResultPod(t)
		expected.Spec.Volumes[0] = corev1.Volume{
			Name: "oneagent-bin",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		}
		setEnvVar(t, &expected, "MODE", "installer")

		assert.Equal(t, expected, updPod)
	})
}

func TestUseImmutableImageWithCSI(t *testing.T) {
	t.Run(`do not use immutable image`, func(t *testing.T) {
		decoder, err := admission.NewDecoder(scheme.Scheme)
		require.NoError(t, err)

		instance := createDynakubeInstance(t)

		inj := &podInjector{
			client: fake.NewClient(
				instance,
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name:   "test-namespace",
						Labels: map[string]string{"oneagent.dynatrace.com/instance": "oneagent"},
					},
				},
			),
			decoder:   decoder,
			image:     "test-image",
			namespace: "dynatrace",
			recorder:  record.NewFakeRecorder(10),
		}

		basePod := corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-pod-12345",
				Namespace: "test-namespace",
				Annotations: map[string]string{
					"oneagent.dynatrace.com/image": "customregistry/linux/codemodule",
				}},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{{
					Name:  "test-container",
					Image: "alpine",
				}},
			},
		}
		basePodBytes, err := json.Marshal(&basePod)
		require.NoError(t, err)

		req := admission.Request{
			AdmissionRequest: admissionv1.AdmissionRequest{
				Object: runtime.RawExtension{
					Raw: basePodBytes,
				},
				Namespace: "test-namespace",
			},
		}
		resp := inj.Handle(context.TODO(), req)
		require.NoError(t, resp.Complete(req))

		if !resp.Allowed {
			require.FailNow(t, "failed to inject", resp.Result)
		}

		patchType := admissionv1.PatchTypeJSONPatch
		assert.Equal(t, resp.PatchType, &patchType)

		patch, err := jsonpatch.DecodePatch(resp.Patch)
		require.NoError(t, err)

		updPodBytes, err := patch.Apply(basePodBytes)
		require.NoError(t, err)

		var updPod corev1.Pod
		require.NoError(t, json.Unmarshal(updPodBytes, &updPod))

		expected := buildResultPod(t)

		expected.ObjectMeta.Annotations["oneagent.dynatrace.com/image"] = "customregistry/linux/codemodule"

		assert.Equal(t, expected, updPod)
	})
	t.Run(`use immutable image`, func(t *testing.T) {
		decoder, err := admission.NewDecoder(scheme.Scheme)
		require.NoError(t, err)

		instance := createDynakubeInstance(t)

		inj := &podInjector{
			client: fake.NewClient(
				instance,
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name:   "test-namespace",
						Labels: map[string]string{"oneagent.dynatrace.com/instance": "oneagent"},
					},
				},
			),
			decoder:   decoder,
			image:     "test-image",
			namespace: "dynatrace",
			recorder:  record.NewFakeRecorder(10),
		}

		basePod := corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-pod-12345",
				Namespace: "test-namespace",
				Annotations: map[string]string{
					"oneagent.dynatrace.com/image": "customregistry/linux/codemodule",
				}},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{{
					Name:  "test-container",
					Image: "alpine",
				}},
			},
		}
		basePodBytes, err := json.Marshal(&basePod)
		require.NoError(t, err)

		req := admission.Request{
			AdmissionRequest: admissionv1.AdmissionRequest{
				Object: runtime.RawExtension{
					Raw: basePodBytes,
				},
				Namespace: "test-namespace",
			},
		}
		resp := inj.Handle(context.TODO(), req)
		require.NoError(t, resp.Complete(req))

		if !resp.Allowed {
			require.FailNow(t, "failed to inject", resp.Result)
		}

		patchType := admissionv1.PatchTypeJSONPatch
		assert.Equal(t, resp.PatchType, &patchType)

		patch, err := jsonpatch.DecodePatch(resp.Patch)
		require.NoError(t, err)

		updPodBytes, err := patch.Apply(basePodBytes)
		require.NoError(t, err)

		var updPod corev1.Pod
		require.NoError(t, json.Unmarshal(updPodBytes, &updPod))

		expected := buildResultPod(t)

		expected.ObjectMeta.Annotations["oneagent.dynatrace.com/image"] = "customregistry/linux/codemodule"

		assert.Equal(t, expected, updPod)
	})

	t.Run(`honor custom image name`, func(t *testing.T) {
		decoder, err := admission.NewDecoder(scheme.Scheme)
		require.NoError(t, err)

		instance := createDynakubeInstance(t)

		inj := &podInjector{
			client: fake.NewClient(
				instance,
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name:   "test-namespace",
						Labels: map[string]string{"oneagent.dynatrace.com/instance": "oneagent"},
					},
				},
			),
			decoder:   decoder,
			image:     "test-image",
			namespace: "dynatrace",
			recorder:  record.NewFakeRecorder(10),
		}

		basePod := corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:        "test-pod-12345",
				Namespace:   "test-namespace",
				Annotations: map[string]string{}},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{{
					Name:  "test-container",
					Image: "alpine",
				}},
			},
		}
		basePodBytes, err := json.Marshal(&basePod)
		require.NoError(t, err)

		req := admission.Request{
			AdmissionRequest: admissionv1.AdmissionRequest{
				Object: runtime.RawExtension{
					Raw: basePodBytes,
				},
				Namespace: "test-namespace",
			},
		}
		resp := inj.Handle(context.TODO(), req)
		require.NoError(t, resp.Complete(req))

		if !resp.Allowed {
			require.FailNow(t, "failed to inject", resp.Result)
		}

		patchType := admissionv1.PatchTypeJSONPatch
		assert.Equal(t, resp.PatchType, &patchType)

		patch, err := jsonpatch.DecodePatch(resp.Patch)
		require.NoError(t, err)

		updPodBytes, err := patch.Apply(basePodBytes)
		require.NoError(t, err)

		var updPod corev1.Pod
		require.NoError(t, json.Unmarshal(updPodBytes, &updPod))

		expected := buildResultPod(t)

		assert.Equal(t, expected, updPod)
	})
}

func TestAgentVersion(t *testing.T) {
	decoder, err := admission.NewDecoder(scheme.Scheme)
	require.NoError(t, err)

	instance := createDynakubeInstance(t)
	withoutCSIDriver(t, instance)
	withAgentVersion(t, instance, testVersion)

	inj := &podInjector{
		client: fake.NewClient(
			instance,
			&corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name:   "test-namespace",
					Labels: map[string]string{"oneagent.dynatrace.com/instance": "oneagent"},
				},
			},
		),
		decoder:   decoder,
		image:     "test-image",
		namespace: "dynatrace",
		recorder:  record.NewFakeRecorder(10),
	}

	basePod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "test-pod-12345",
			Namespace:   "test-namespace",
			Annotations: map[string]string{}},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{{
				Name:  "test-container",
				Image: "alpine",
			}},
		},
	}
	basePodBytes, err := json.Marshal(&basePod)
	require.NoError(t, err)

	req := admission.Request{
		AdmissionRequest: admissionv1.AdmissionRequest{
			Object: runtime.RawExtension{
				Raw: basePodBytes,
			},
			Namespace: "test-namespace",
		},
	}
	resp := inj.Handle(context.TODO(), req)
	require.NoError(t, resp.Complete(req))

	if !resp.Allowed {
		require.FailNow(t, "failed to inject", resp.Result)
	}

	patchType := admissionv1.PatchTypeJSONPatch
	assert.Equal(t, resp.PatchType, &patchType)

	patch, err := jsonpatch.DecodePatch(resp.Patch)
	require.NoError(t, err)

	updPodBytes, err := patch.Apply(basePodBytes)
	require.NoError(t, err)

	var updPod corev1.Pod
	require.NoError(t, json.Unmarshal(updPodBytes, &updPod))

	expected := buildResultPod(t)
	expected.Spec.Volumes[0] = corev1.Volume{
		Name: "oneagent-bin",
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{},
		},
	}
	setEnvVar(t, &expected, "MODE", "installer")

	assert.Equal(t, expected, updPod)
}

func TestAgentVersionWithCSI(t *testing.T) {
	decoder, err := admission.NewDecoder(scheme.Scheme)
	require.NoError(t, err)

	instance := createDynakubeInstance(t)
	withAgentVersion(t, instance, testVersion)

	inj := &podInjector{
		client: fake.NewClient(
			instance,
			&corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name:   "test-namespace",
					Labels: map[string]string{"oneagent.dynatrace.com/instance": "oneagent"},
				},
			},
		),
		decoder:   decoder,
		image:     "test-image",
		namespace: "dynatrace",
		recorder:  record.NewFakeRecorder(10),
	}

	basePod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "test-pod-12345",
			Namespace:   "test-namespace",
			Annotations: map[string]string{}},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{{
				Name:  "test-container",
				Image: "alpine",
			}},
		},
	}
	basePodBytes, err := json.Marshal(&basePod)
	require.NoError(t, err)

	req := admission.Request{
		AdmissionRequest: admissionv1.AdmissionRequest{
			Object: runtime.RawExtension{
				Raw: basePodBytes,
			},
			Namespace: "test-namespace",
		},
	}
	resp := inj.Handle(context.TODO(), req)
	require.NoError(t, resp.Complete(req))

	if !resp.Allowed {
		require.FailNow(t, "failed to inject", resp.Result)
	}

	patchType := admissionv1.PatchTypeJSONPatch
	assert.Equal(t, resp.PatchType, &patchType)

	patch, err := jsonpatch.DecodePatch(resp.Patch)
	require.NoError(t, err)

	updPodBytes, err := patch.Apply(basePodBytes)
	require.NoError(t, err)

	var updPod corev1.Pod
	require.NoError(t, json.Unmarshal(updPodBytes, &updPod))

	expected := buildResultPod(t)
	assert.Equal(t, expected, updPod)
}

func buildResultPod(_ *testing.T) corev1.Pod {
	return corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod-12345",
			Namespace: "test-namespace",
			Annotations: map[string]string{
				"oneagent.dynatrace.com/injected": "true",
			},
		},
		Spec: corev1.PodSpec{
			InitContainers: []corev1.Container{{
				Name:            dtwebhook.InstallContainerName,
				Image:           "test-image",
				ImagePullPolicy: corev1.PullAlways,
				Command:         []string{"/usr/bin/env"},
				Args:            []string{"bash", "/mnt/config/init.sh"},
				Env: []corev1.EnvVar{
					{Name: "FLAVOR", Value: dtclient.FlavorMultidistro},
					{Name: "TECHNOLOGIES", Value: "all"},
					{Name: "INSTALLPATH", Value: "/opt/dynatrace/oneagent-paas"},
					{Name: "INSTALLER_URL", Value: ""},
					{Name: "FAILURE_POLICY", Value: "silent"},
					{Name: "CONTAINERS_COUNT", Value: "1"},
					{Name: "MODE", Value: "provisioned"},
					{Name: "K8S_PODNAME", ValueFrom: &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.name"}}},
					{Name: "K8S_PODUID", ValueFrom: &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.uid"}}},
					{Name: "K8S_BASEPODNAME", Value: "test-pod"},
					{Name: "K8S_NAMESPACE", ValueFrom: &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.namespace"}}},
					{Name: "K8S_NODE_NAME", ValueFrom: &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{FieldPath: "spec.nodeName"}}},
					{Name: "CONTAINER_1_NAME", Value: "test-container"},
					{Name: "CONTAINER_1_IMAGE", Value: "alpine"},
				},
				VolumeMounts: []corev1.VolumeMount{
					{Name: "oneagent-bin", MountPath: "/mnt/bin"},
					{Name: "oneagent-share", MountPath: "/mnt/share"},
					{Name: "oneagent-config", MountPath: "/mnt/config"},
				},
			}},
			Containers: []corev1.Container{{
				Name:  "test-container",
				Image: "alpine",
				Env: []corev1.EnvVar{
					{Name: "LD_PRELOAD", Value: "/opt/dynatrace/oneagent-paas/agent/lib64/liboneagentproc.so"},
					{Name: "DT_DEPLOYMENT_METADATA", Value: "orchestration_tech=Operator;script_version=snapshot;orchestrator_id="},
				},
				VolumeMounts: []corev1.VolumeMount{
					{Name: "oneagent-share", MountPath: "/etc/ld.so.preload", SubPath: "ld.so.preload"},
					{Name: "oneagent-bin", MountPath: "/opt/dynatrace/oneagent-paas"},
					{
						Name:      "oneagent-share",
						MountPath: "/var/lib/dynatrace/oneagent/agent/config/container.conf",
						SubPath:   "container_test-container.conf",
					},
				},
			}},
			Volumes: []corev1.Volume{
				{
					Name: "oneagent-bin",
					VolumeSource: corev1.VolumeSource{
						CSI: &corev1.CSIVolumeSource{
							Driver: dtcsi.DriverName,
						},
					},
				},
				{
					Name: "oneagent-share",
					VolumeSource: corev1.VolumeSource{
						EmptyDir: &corev1.EmptyDirVolumeSource{},
					},
				},
				{
					Name: "oneagent-config",
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName: dtwebhook.SecretConfigName,
						},
					},
				},
			},
		},
	}
}

func setEnvVar(_ *testing.T, pod *corev1.Pod, name string, value string) {
	for idx := range pod.Spec.InitContainers[0].Env {
		if pod.Spec.InitContainers[0].Env[idx].Name == name {
			pod.Spec.InitContainers[0].Env[idx].Value = value
			break
		}
	}
}

func TestInstrumentThirdPartyContainers(t *testing.T) {
	decoder, err := admission.NewDecoder(scheme.Scheme)
	require.NoError(t, err)

	inj, instance := createPodInjector(t, decoder)

	// enable feature
	instance.Annotations = map[string]string{}
	instance.Annotations[instance.GetFeatureEnableWebhookReinvocationPolicy()] = "true"
	err = inj.client.Update(context.TODO(), instance)
	require.NoError(t, err)

	var testContainerName, testContainerImage = "test-container", "alpine"
	var thirdPartyContainerName, thirdPartyContainerImage = "third-party-container", "sidecar-image"
	basePod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "test-pod-12345",
			Namespace:   "test-namespace",
			Annotations: map[string]string{dtwebhook.AnnotationInjected: "true"}},
		Spec: corev1.PodSpec{
			InitContainers: []corev1.Container{{
				Name:  dtwebhook.InstallContainerName,
				Image: "test-installer",
				Env: []corev1.EnvVar{
					{
						Name:  "CONTAINER_1_NAME",
						Value: testContainerName,
					},
					{
						Name:  "CONTAINER_1_IMAGE",
						Value: testContainerImage,
					},
				},
			}},
			Containers: []corev1.Container{
				{
					Name:  testContainerName,
					Image: testContainerImage,
					Env: []corev1.EnvVar{{
						Name:  "LD_PRELOAD",
						Value: "test-install-path",
					}},
				},
				{
					Name:  thirdPartyContainerName,
					Image: thirdPartyContainerImage,
				},
			},
		},
	}

	basePodBytes, err := json.Marshal(&basePod)
	require.NoError(t, err)

	// check setup
	require.Equal(t, "LD_PRELOAD", basePod.Spec.Containers[0].Env[0].Name)

	var baseInstallContainer = basePod.Spec.InitContainers[0]
	require.Equal(t, 2, len(baseInstallContainer.Env))
	require.Equal(t, "CONTAINER_1_NAME", baseInstallContainer.Env[0].Name)
	require.Equal(t, testContainerName, baseInstallContainer.Env[0].Value)
	require.Equal(t, "CONTAINER_1_IMAGE", baseInstallContainer.Env[1].Name)
	require.Equal(t, testContainerImage, baseInstallContainer.Env[1].Value)

	// handle request
	req := admission.Request{
		AdmissionRequest: admissionv1.AdmissionRequest{
			Object: runtime.RawExtension{
				Raw: basePodBytes,
			},
			Namespace: "test-namespace",
		},
	}
	resp := inj.Handle(context.TODO(), req)
	require.NoError(t, resp.Complete(req))

	// update pod with response
	patchType := admissionv1.PatchTypeJSONPatch
	assert.Equal(t, &patchType, resp.PatchType)

	patch, err := jsonpatch.DecodePatch(resp.Patch)
	require.NoError(t, err)

	updPodBytes, err := patch.Apply(basePodBytes)
	require.NoError(t, err)

	var updPod corev1.Pod
	require.NoError(t, json.Unmarshal(updPodBytes, &updPod))

	// check updated pod
	require.Equal(t, "LD_PRELOAD", updPod.Spec.Containers[1].Env[0].Name)

	var updInstallContainer = updPod.Spec.InitContainers[0]
	require.Equal(t, 4, len(updInstallContainer.Env))
	require.Equal(t, "CONTAINER_2_NAME", updInstallContainer.Env[2].Name)
	require.Equal(t, thirdPartyContainerName, updInstallContainer.Env[2].Value)
	require.Equal(t, "CONTAINER_2_IMAGE", updInstallContainer.Env[3].Name)
	require.Equal(t, thirdPartyContainerImage, updInstallContainer.Env[3].Value)
}
