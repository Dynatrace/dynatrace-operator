package mutation

import (
	"context"
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/api/v1beta1"
	dtcsi "github.com/Dynatrace/dynatrace-operator/controllers/csi"
	dtingestendpoint "github.com/Dynatrace/dynatrace-operator/controllers/ingestendpoint"
	"github.com/Dynatrace/dynatrace-operator/dtclient"
	"github.com/Dynatrace/dynatrace-operator/mapper"
	"github.com/Dynatrace/dynatrace-operator/scheme"
	"github.com/Dynatrace/dynatrace-operator/scheme/fake"
	t_utils "github.com/Dynatrace/dynatrace-operator/testing"
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
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

const (
	fakeEventRecorderBufferSize = 10
	dynakubeName                = "dynakube"
	dataIngestToken             = "data-ingest-token"
)

func TestInjectionWithMissingOneAgentAPM(t *testing.T) {
	decoder, err := admission.NewDecoder(scheme.Scheme)
	require.NoError(t, err)

	inj := &podMutator{
		client: fake.NewClient(
			&corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name:   "test-namespace",
					Labels: map[string]string{mapper.InstanceLabel: dynakubeName},
				},
			}),
		apiReader: fake.NewClient(
			&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      dtwebhook.SecretConfigName,
					Namespace: "test-namespace",
				},
			},
		),
		decoder:   decoder,
		image:     "operator-image",
		namespace: "dynatrace",
		recorder:  record.NewFakeRecorder(fakeEventRecorderBufferSize),
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
	t_utils.AssertEvents(t,
		inj.recorder.(*record.FakeRecorder).Events,
		t_utils.Events{
			t_utils.Event{
				EventType: corev1.EventTypeWarning,
				Reason:    missingDynakubeEvent,
			},
		},
	)
}

func createPodInjector(_ *testing.T, decoder *admission.Decoder) (*podMutator, *dynatracev1beta1.DynaKube) {
	dynakube := &dynatracev1beta1.DynaKube{
		ObjectMeta: metav1.ObjectMeta{Name: dynakubeName, Namespace: "dynatrace"},
		Spec: dynatracev1beta1.DynaKubeSpec{
			APIURL: "https://test-api-url.com/api",
			OneAgent: dynatracev1beta1.OneAgentSpec{
				CloudNativeFullStack: &dynatracev1beta1.CloudNativeFullStackSpec{
					AppInjectionSpec: dynatracev1beta1.AppInjectionSpec{
						InitResources: corev1.ResourceRequirements{
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
			},
		},
	}

	return &podMutator{
		client: fake.NewClient(
			dynakube,
			&corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name:   "test-namespace",
					Labels: map[string]string{mapper.InstanceLabel: dynakubeName},
				},
			},
			&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      dynakubeName,
					Namespace: "dynatrace",
				},
				Data: map[string][]byte{
					dtclient.DynatraceDataIngestToken: []byte(dataIngestToken),
				},
			},
		),
		apiReader: buildTestSecrets(),
		decoder:   decoder,
		image:     "test-api-url.com/linux/codemodule",
		namespace: "dynatrace",
		recorder:  record.NewFakeRecorder(fakeEventRecorderBufferSize),
	}, dynakube
}

func TestPodInjection(t *testing.T) {
	decoder, err := admission.NewDecoder(scheme.Scheme)
	require.NoError(t, err)

	inj, instance := createPodInjector(t, decoder)
	err = inj.client.Update(context.TODO(), instance)
	require.NoError(t, err)

	basePod := corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind: "Pod",
		},
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
			CSI: &corev1.CSIVolumeSource{
				Driver: dtcsi.DriverName,
			},
		},
	}

	setEnvVar(t, &expected, "MODE", "provisioned")

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
	t_utils.AssertEvents(t,
		inj.recorder.(*record.FakeRecorder).Events,
		t_utils.Events{
			t_utils.Event{
				EventType: corev1.EventTypeNormal,
				Reason:    injectEvent,
			},
		},
	)
}

func TestPodInjectionWithCSI(t *testing.T) {
	decoder, err := admission.NewDecoder(scheme.Scheme)
	require.NoError(t, err)

	inj, _ := createPodInjector(t, decoder)

	basePod := corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind: "Pod",
		},
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

	t_utils.AssertEvents(t,
		inj.recorder.(*record.FakeRecorder).Events,
		t_utils.Events{
			t_utils.Event{
				EventType: corev1.EventTypeNormal,
				Reason:    injectEvent,
			},
		},
	)
}

func createDynakubeInstance(_ *testing.T) *dynatracev1beta1.DynaKube {
	instance := &dynatracev1beta1.DynaKube{
		ObjectMeta: metav1.ObjectMeta{Name: dynakubeName, Namespace: "dynatrace"},
		Spec: dynatracev1beta1.DynaKubeSpec{
			APIURL: "https://test-api-url.com/api",
			OneAgent: dynatracev1beta1.OneAgentSpec{
				CloudNativeFullStack: &dynatracev1beta1.CloudNativeFullStackSpec{},
			},
		},
	}

	return instance
}

func TestUseImmutableImage(t *testing.T) {
	t.Run(`use immutable image`, func(t *testing.T) {
		decoder, err := admission.NewDecoder(scheme.Scheme)
		require.NoError(t, err)

		instance := createDynakubeInstance(t)

		inj := &podMutator{
			client: fake.NewClient(
				instance,
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name:   "test-namespace",
						Labels: map[string]string{mapper.InstanceLabel: dynakubeName},
					},
				},
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      dynakubeName,
						Namespace: "dynatrace",
					},
					Data: map[string][]byte{
						dtclient.DynatraceDataIngestToken: []byte(dataIngestToken),
					},
				},
			),
			apiReader: buildTestSecrets(),
			decoder:   decoder,
			image:     "test-image",
			namespace: "dynatrace",
			recorder:  record.NewFakeRecorder(fakeEventRecorderBufferSize),
		}

		basePod := corev1.Pod{
			TypeMeta: metav1.TypeMeta{
				Kind: "Pod",
			},
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
				CSI: &corev1.CSIVolumeSource{
					Driver: dtcsi.DriverName,
				},
			},
		}
		setEnvVar(t, &expected, "MODE", "provisioned")

		expected.ObjectMeta.Annotations["oneagent.dynatrace.com/image"] = "customregistry/linux/codemodule"

		assert.Equal(t, expected, updPod)
		t_utils.AssertEvents(t,
			inj.recorder.(*record.FakeRecorder).Events,
			t_utils.Events{
				t_utils.Event{
					EventType: corev1.EventTypeNormal,
					Reason:    injectEvent,
				},
			},
		)
	})

	t.Run(`honor custom image name`, func(t *testing.T) {
		decoder, err := admission.NewDecoder(scheme.Scheme)
		require.NoError(t, err)

		instance := createDynakubeInstance(t)

		inj := &podMutator{
			client: fake.NewClient(
				instance,
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name:   "test-namespace",
						Labels: map[string]string{mapper.InstanceLabel: dynakubeName},
					},
				},
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      dynakubeName,
						Namespace: "dynatrace",
					},
					Data: map[string][]byte{
						dtclient.DynatraceDataIngestToken: []byte(dataIngestToken),
					},
				},
			),
			apiReader: buildTestSecrets(),
			decoder:   decoder,
			image:     "test-image",
			namespace: "dynatrace",
			recorder:  record.NewFakeRecorder(fakeEventRecorderBufferSize),
		}

		basePod := corev1.Pod{
			TypeMeta: metav1.TypeMeta{
				Kind: "Pod",
			},
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
				CSI: &corev1.CSIVolumeSource{
					Driver: dtcsi.DriverName,
				},
			},
		}
		setEnvVar(t, &expected, "MODE", "provisioned")

		assert.Equal(t, expected, updPod)
		t_utils.AssertEvents(t,
			inj.recorder.(*record.FakeRecorder).Events,
			t_utils.Events{
				t_utils.Event{
					EventType: corev1.EventTypeNormal,
					Reason:    injectEvent,
				},
			},
		)
	})
}

func TestUseImmutableImageWithCSI(t *testing.T) {
	t.Run(`do not use immutable image`, func(t *testing.T) {
		decoder, err := admission.NewDecoder(scheme.Scheme)
		require.NoError(t, err)

		instance := createDynakubeInstance(t)

		inj := &podMutator{
			client: fake.NewClient(
				instance,
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name:   "test-namespace",
						Labels: map[string]string{mapper.InstanceLabel: instance.Name},
					},
				},
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      dynakubeName,
						Namespace: "dynatrace",
					},
					Data: map[string][]byte{
						dtclient.DynatraceDataIngestToken: []byte(dataIngestToken),
					},
				},
			),
			apiReader: buildTestSecrets(),
			decoder:   decoder,
			image:     "test-image",
			namespace: "dynatrace",
			recorder:  record.NewFakeRecorder(fakeEventRecorderBufferSize),
		}

		basePod := corev1.Pod{
			TypeMeta: metav1.TypeMeta{
				Kind: "Pod",
			},
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
		t_utils.AssertEvents(t,
			inj.recorder.(*record.FakeRecorder).Events,
			t_utils.Events{
				t_utils.Event{
					EventType: corev1.EventTypeNormal,
					Reason:    injectEvent,
				},
			},
		)
	})
	t.Run(`use immutable image`, func(t *testing.T) {
		decoder, err := admission.NewDecoder(scheme.Scheme)
		require.NoError(t, err)

		instance := createDynakubeInstance(t)

		inj := &podMutator{
			client: fake.NewClient(
				instance,
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name:   "test-namespace",
						Labels: map[string]string{mapper.InstanceLabel: instance.Name},
					},
				},
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      dynakubeName,
						Namespace: "dynatrace",
					},
					Data: map[string][]byte{
						dtclient.DynatraceDataIngestToken: []byte(dataIngestToken),
					},
				},
			),
			apiReader: buildTestSecrets(),
			decoder:   decoder,
			image:     "test-image",
			namespace: "dynatrace",
			recorder:  record.NewFakeRecorder(fakeEventRecorderBufferSize),
		}

		basePod := corev1.Pod{
			TypeMeta: metav1.TypeMeta{
				Kind: "Pod",
			},
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
		t_utils.AssertEvents(t,
			inj.recorder.(*record.FakeRecorder).Events,
			t_utils.Events{
				t_utils.Event{
					EventType: corev1.EventTypeNormal,
					Reason:    injectEvent,
				},
			},
		)
	})

	t.Run(`honor custom image name`, func(t *testing.T) {
		decoder, err := admission.NewDecoder(scheme.Scheme)
		require.NoError(t, err)

		instance := createDynakubeInstance(t)

		inj := &podMutator{
			client: fake.NewClient(
				instance,
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name:   "test-namespace",
						Labels: map[string]string{mapper.InstanceLabel: instance.Name},
					},
				},
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      dynakubeName,
						Namespace: "dynatrace",
					},
					Data: map[string][]byte{
						dtclient.DynatraceDataIngestToken: []byte(dataIngestToken),
					},
				},
			),
			apiReader: buildTestSecrets(),
			decoder:   decoder,
			image:     "test-image",
			namespace: "dynatrace",
			recorder:  record.NewFakeRecorder(fakeEventRecorderBufferSize),
		}

		basePod := corev1.Pod{
			TypeMeta: metav1.TypeMeta{
				Kind: "Pod",
			},
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
		t_utils.AssertEvents(t,
			inj.recorder.(*record.FakeRecorder).Events,
			t_utils.Events{
				t_utils.Event{
					EventType: corev1.EventTypeNormal,
					Reason:    injectEvent,
				},
			},
		)
	})
}

func TestAgentVersion(t *testing.T) {
	decoder, err := admission.NewDecoder(scheme.Scheme)
	require.NoError(t, err)

	instance := createDynakubeInstance(t)

	inj := &podMutator{
		client: fake.NewClient(
			instance,
			&corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name:   "test-namespace",
					Labels: map[string]string{mapper.InstanceLabel: instance.Name},
				},
			},
			&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      dynakubeName,
					Namespace: "dynatrace",
				},
				Data: map[string][]byte{
					dtclient.DynatraceDataIngestToken: []byte(dataIngestToken),
				},
			},
		),
		apiReader: buildTestSecrets(),
		decoder:   decoder,
		image:     "test-image",
		namespace: "dynatrace",
		recorder:  record.NewFakeRecorder(fakeEventRecorderBufferSize),
	}

	basePod := corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind: "Pod",
		},
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
			CSI: &corev1.CSIVolumeSource{
				Driver: dtcsi.DriverName,
			},
		},
	}
	setEnvVar(t, &expected, "MODE", "provisioned")

	assert.Equal(t, expected, updPod)
	t_utils.AssertEvents(t,
		inj.recorder.(*record.FakeRecorder).Events,
		t_utils.Events{
			t_utils.Event{
				EventType: corev1.EventTypeNormal,
				Reason:    injectEvent,
			},
		},
	)
}

func TestAgentVersionWithCSI(t *testing.T) {
	decoder, err := admission.NewDecoder(scheme.Scheme)
	require.NoError(t, err)

	instance := createDynakubeInstance(t)

	inj := &podMutator{
		client: fake.NewClient(
			instance,
			&corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name:   "test-namespace",
					Labels: map[string]string{mapper.InstanceLabel: instance.Name},
				},
			},
			&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      dynakubeName,
					Namespace: "dynatrace",
				},
				Data: map[string][]byte{
					dtclient.DynatraceDataIngestToken: []byte(dataIngestToken),
				},
			},
		),
		apiReader: buildTestSecrets(),
		decoder:   decoder,
		image:     "test-image",
		namespace: "dynatrace",
		recorder:  record.NewFakeRecorder(fakeEventRecorderBufferSize),
	}

	basePod := corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind: "Pod",
		},
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

	t_utils.AssertEvents(t,
		inj.recorder.(*record.FakeRecorder).Events,
		t_utils.Events{
			t_utils.Event{
				EventType: corev1.EventTypeNormal,
				Reason:    injectEvent,
			},
		},
	)
}

func buildResultPod(_ *testing.T) corev1.Pod {
	return corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind: "Pod",
		},
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
				ImagePullPolicy: corev1.PullIfNotPresent,
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
					{Name: "DT_WORKLOAD_KIND", Value: "Pod"},
					{Name: "DT_WORKLOAD_NAME", Value: "test-pod-12345"},
					{Name: "CONTAINER_1_NAME", Value: "test-container"},
					{Name: "CONTAINER_1_IMAGE", Value: "alpine"},
				},
				VolumeMounts: []corev1.VolumeMount{
					{Name: "oneagent-bin", MountPath: "/mnt/bin"},
					{Name: "oneagent-share", MountPath: "/mnt/share"},
					{Name: "oneagent-config", MountPath: "/mnt/config"},
					{Name: "mint-enrichment", MountPath: "/var/lib/dynatrace/enrichment"},
				},
			}},
			Containers: []corev1.Container{{
				Name:  "test-container",
				Image: "alpine",
				Env: []corev1.EnvVar{
					{Name: "LD_PRELOAD", Value: "/opt/dynatrace/oneagent-paas/agent/lib64/liboneagentproc.so"},
					{Name: "DT_DEPLOYMENT_METADATA", Value: "orchestration_tech=Operator-cloud_native_fullstack;script_version=snapshot;orchestrator_id="},
					{Name: dtingestendpoint.UrlSecretField, Value: "https://test-api-url.com/api/v2/metrics/ingest"},
					{Name: dtingestendpoint.TokenSecretField, Value: dataIngestToken},
				},
				VolumeMounts: []corev1.VolumeMount{
					{Name: "oneagent-share", MountPath: "/etc/ld.so.preload", SubPath: "ld.so.preload"},
					{Name: "oneagent-bin", MountPath: "/opt/dynatrace/oneagent-paas"},
					{
						Name:      "oneagent-share",
						MountPath: "/var/lib/dynatrace/oneagent/agent/config/container.conf",
						SubPath:   "container_test-container.conf",
					},
					{Name: "data-ingest-endpoint", MountPath: "/var/lib/dynatrace/enrichment/endpoint"},
					{Name: "mint-enrichment", MountPath: "/var/lib/dynatrace/enrichment"},
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
				{
					Name: "data-ingest-endpoint",
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName: dtingestendpoint.SecretEndpointName,
						},
					},
				},
				{
					Name: "mint-enrichment",
					VolumeSource: corev1.VolumeSource{
						EmptyDir: &corev1.EmptyDirVolumeSource{},
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

	t_utils.AssertEvents(t,
		inj.recorder.(*record.FakeRecorder).Events,
		t_utils.Events{
			t_utils.Event{
				EventType: corev1.EventTypeNormal,
				Reason:    updatePodEvent,
			},
		},
	)
}

func buildTestSecrets() client.Client {
	return fake.NewClient(
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      dtwebhook.SecretConfigName,
				Namespace: "test-namespace",
			},
		},
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      dtingestendpoint.SecretEndpointName,
				Namespace: "test-namespace",
			},
		},
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      dynakubeName,
				Namespace: "dynatrace",
			},
			Data: map[string][]byte{
				dtclient.DynatraceDataIngestToken: []byte(dataIngestToken),
			},
		},
	)
}
