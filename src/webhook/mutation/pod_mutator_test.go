package mutation

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/arch"
	dtcsi "github.com/Dynatrace/dynatrace-operator/src/controllers/csi"
	csivolumes "github.com/Dynatrace/dynatrace-operator/src/controllers/csi/driver/volumes"
	appvolumes "github.com/Dynatrace/dynatrace-operator/src/controllers/csi/driver/volumes/app"
	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
	dtingestendpoint "github.com/Dynatrace/dynatrace-operator/src/ingestendpoint"
	"github.com/Dynatrace/dynatrace-operator/src/mapper"
	"github.com/Dynatrace/dynatrace-operator/src/scheme"
	"github.com/Dynatrace/dynatrace-operator/src/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/src/standalone"
	t_utils "github.com/Dynatrace/dynatrace-operator/src/testing"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/src/webhook"
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
	require.True(t, resp.Allowed)
	require.Equal(t, resp.Result.Message, "Failed to inject into pod: test-pod-123456 because namespace 'test-namespace' is assigned to DynaKube instance 'dynakube' but doesn't exist")
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
		ObjectMeta: metav1.ObjectMeta{
			Name:      dynakubeName,
			Namespace: "dynatrace",
		},
		Spec: dynatracev1beta1.DynaKubeSpec{
			APIURL: "https://tenant.test-api-url.com/api",
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

func TestPodPartialInjection(t *testing.T) {
	type fields struct {
		injectionInfo *InjectionInfo
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "Both OA and DI enabled by default",
			fields: fields{injectionInfo: func() *InjectionInfo {
				i := NewInjectionInfo()
				return i
			}(),
			},
			want: "",
		},
		{
			name: "OA enabled by default, DI explicitly disabled",
			fields: fields{injectionInfo: func() *InjectionInfo {
				i := NewInjectionInfo()
				i.add(NewFeature(DataIngest, false))
				return i
			}(),
			},
			want: "",
		},
		{
			name: "OA explicitly enabled, DI explicitly disabled",
			fields: fields{injectionInfo: func() *InjectionInfo {
				i := NewInjectionInfo()
				i.add(NewFeature(OneAgent, true))
				i.add(NewFeature(DataIngest, false))
				return i
			}(),
			},
			want: "",
		},
		{
			name: "OA explicitly disabled, DI disabled by default (trait inherited from OA)",
			fields: fields{injectionInfo: func() *InjectionInfo {
				i := NewInjectionInfo()
				i.add(NewFeature(OneAgent, false))
				return i
			}(),
			},
			want: "",
		},
		{
			name: "OA explicitly disabled, DI explicitly disabled",
			fields: fields{injectionInfo: func() *InjectionInfo {
				i := NewInjectionInfo()
				i.add(NewFeature(OneAgent, false))
				i.add(NewFeature(DataIngest, false))
				return i
			}(),
			},
			want: "",
		},
		{
			name: "OA explicitly disabled, DI explicitly enabled",
			fields: fields{injectionInfo: func() *InjectionInfo {
				i := NewInjectionInfo()
				i.add(NewFeature(OneAgent, false))
				i.add(NewFeature(DataIngest, true))
				return i
			}(),
			},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			impl(t, tt.fields.injectionInfo)
		})
	}
}

func impl(t *testing.T, injectionInfo *InjectionInfo) {
	decoder, err := admission.NewDecoder(scheme.Scheme)
	require.NoError(t, err)

	inj, instance := createPodInjector(t, decoder)
	err = inj.client.Update(context.TODO(), instance)
	require.NoError(t, err)

	basePod := corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind: "Pod",
		},
		ObjectMeta: metav1.ObjectMeta{Name: "test-pod-12345", Namespace: "test-namespace", Annotations: injectionInfo.createInjectAnnotations()},
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

	if resp.PatchType == nil {
		// webhook does nothing
		return
	}

	patchType := admissionv1.PatchTypeJSONPatch
	assert.Equal(t, resp.PatchType, &patchType)

	patch, err := jsonpatch.DecodePatch(resp.Patch)
	require.NoError(t, err)

	updPodBytes, err := patch.Apply(basePodBytes)
	require.NoError(t, err)

	var updPod corev1.Pod
	require.NoError(t, json.Unmarshal(updPodBytes, &updPod))

	var oaFf, diFf FeatureFlag
	if injectionInfo.exists(OneAgent) {
		oaFf = FeatureFlag{explicitlyEnabled: injectionInfo.enabled(OneAgent)}
	} else {
		oaFf = FeatureFlag{defaultMode: true}
	}
	if injectionInfo.exists(DataIngest) {
		diFf = FeatureFlag{explicitlyEnabled: injectionInfo.enabled(DataIngest)}
	} else {
		diFf = FeatureFlag{defaultMode: true}
	}

	expected := buildResultPod(t, oaFf, diFf)
	addCSIVolumeSource(&expected)

	setEnvVar(t, &expected, "MODE", provisionedVolumeMode)

	if len(expected.Spec.InitContainers) > 0 {
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
	}

	sortPodInternals(&expected)
	sortPodInternals(&updPod)
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

	expected := buildResultPod(t, FeatureFlag{defaultMode: true}, FeatureFlag{defaultMode: true})
	addCSIVolumeSource(&expected)

	setEnvVar(t, &expected, "MODE", provisionedVolumeMode)

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

	sortPodInternals(&expected)
	sortPodInternals(&updPod)
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

	expected := buildResultPod(t, FeatureFlag{defaultMode: true}, FeatureFlag{defaultMode: true})

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

	sortPodInternals(&expected)
	sortPodInternals(&updPod)
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
			APIURL: "https://tenant.test-api-url.com/api",
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

		expected := buildResultPod(t, FeatureFlag{defaultMode: true}, FeatureFlag{defaultMode: true})
		addCSIVolumeSource(&expected)

		setEnvVar(t, &expected, "MODE", provisionedVolumeMode)

		expected.ObjectMeta.Annotations["oneagent.dynatrace.com/image"] = "customregistry/linux/codemodule"

		sortPodInternals(&expected)
		sortPodInternals(&updPod)
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

		expected := buildResultPod(t, FeatureFlag{defaultMode: true}, FeatureFlag{defaultMode: true})
		addCSIVolumeSource(&expected)

		setEnvVar(t, &expected, "MODE", provisionedVolumeMode)

		sortPodInternals(&expected)
		sortPodInternals(&updPod)
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

func addCSIVolumeSource(expected *corev1.Pod) {
	idx := sort.Search(len(expected.Spec.Volumes), func(i int) bool {
		return expected.Spec.Volumes[i].Name >= oneAgentBinVolumeName
	})

	if idx < len(expected.Spec.Volumes) {
		expected.Spec.Volumes[idx].VolumeSource = corev1.VolumeSource{
			CSI: &corev1.CSIVolumeSource{
				Driver: dtcsi.DriverName,
				VolumeAttributes: map[string]string{
					csivolumes.CSIVolumeAttributeModeField:     appvolumes.Mode,
					csivolumes.CSIVolumeAttributeDynakubeField: dynakubeName,
				},
			},
		}
	}
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

		expected := buildResultPod(t, FeatureFlag{defaultMode: true}, FeatureFlag{defaultMode: true})

		expected.ObjectMeta.Annotations["oneagent.dynatrace.com/image"] = "customregistry/linux/codemodule"

		sortPodInternals(&expected)
		sortPodInternals(&updPod)
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

		expected := buildResultPod(t, FeatureFlag{defaultMode: true}, FeatureFlag{defaultMode: true})

		expected.ObjectMeta.Annotations["oneagent.dynatrace.com/image"] = "customregistry/linux/codemodule"

		sortPodInternals(&expected)
		sortPodInternals(&updPod)
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

		expected := buildResultPod(t, FeatureFlag{defaultMode: true}, FeatureFlag{defaultMode: true})

		sortPodInternals(&expected)
		sortPodInternals(&updPod)
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

	expected := buildResultPod(t, FeatureFlag{defaultMode: true}, FeatureFlag{defaultMode: true})
	addCSIVolumeSource(&expected)

	setEnvVar(t, &expected, "MODE", provisionedVolumeMode)

	sortPodInternals(&expected)
	sortPodInternals(&updPod)
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

	expected := buildResultPod(t, FeatureFlag{defaultMode: true}, FeatureFlag{defaultMode: true})

	sortPodInternals(&expected)
	sortPodInternals(&updPod)
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

// `defaultMode` overrides `explicitlyEnabled`
type FeatureFlag struct {
	explicitlyEnabled bool
	defaultMode       bool
}

func buildResultPod(_ *testing.T, oneAgentFf FeatureFlag, dataIngestFf FeatureFlag) corev1.Pod {
	pod := corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind: "Pod",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod-12345",
			Namespace: "test-namespace",
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{{
				Name:  "test-container",
				Image: "alpine",
			}},
		},
	}

	oaEnabled := oneAgentFf.defaultMode || oneAgentFf.explicitlyEnabled
	diEnabled := (oaEnabled && dataIngestFf.defaultMode) || dataIngestFf.explicitlyEnabled

	if pod.ObjectMeta.Annotations == nil {
		pod.ObjectMeta.Annotations = make(map[string]string)
	}

	if oaEnabled && diEnabled {
		pod.ObjectMeta.Annotations["dynakube.dynatrace.com/injected"] = "data-ingest,oneagent"
	} else if oaEnabled {
		pod.ObjectMeta.Annotations["dynakube.dynatrace.com/injected"] = "oneagent"
	} else if diEnabled {
		pod.ObjectMeta.Annotations["dynakube.dynatrace.com/injected"] = "data-ingest"
	}

	if !oneAgentFf.defaultMode {
		pod.ObjectMeta.Annotations[dtwebhook.AnnotationOneAgentInject] = strconv.FormatBool(oneAgentFf.explicitlyEnabled)
	}
	if !dataIngestFf.defaultMode {
		pod.ObjectMeta.Annotations[dtwebhook.AnnotationDataIngestInject] = strconv.FormatBool(dataIngestFf.explicitlyEnabled)
	}

	if oaEnabled || diEnabled {
		pod.Spec.Volumes = []corev1.Volume{
			{
				Name: injectionConfigVolumeName,
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName: dtwebhook.SecretConfigName,
					},
				},
			},
		}

		pod.Spec.InitContainers = []corev1.Container{{
			Name:            dtwebhook.InstallContainerName,
			Image:           "test-image",
			ImagePullPolicy: corev1.PullIfNotPresent,
			Args:            []string{"init"},
			Env: []corev1.EnvVar{
				{Name: standalone.CanFailEnv, Value: "silent"},
				{Name: standalone.ContainerCountEnv, Value: "1"},
				{Name: standalone.K8PodNameEnv, ValueFrom: &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.name"}}},
				{Name: standalone.K8PodUIDEnv, ValueFrom: &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.uid"}}},
				{Name: standalone.K8BasePodNameEnv, Value: "test-pod"},
				{Name: standalone.K8NamespaceEnv, ValueFrom: &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.namespace"}}},
				{Name: standalone.K8NodeNameEnv, ValueFrom: &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{FieldPath: "spec.nodeName"}}},
			},
			VolumeMounts: []corev1.VolumeMount{
				{Name: injectionConfigVolumeName, MountPath: standalone.ConfigDirMount},
			},
		}}

		pod.Spec.Containers[0].Env = []corev1.EnvVar{
			{Name: "DT_DEPLOYMENT_METADATA", Value: "orchestration_tech=Operator-cloud_native_fullstack;script_version=snapshot;orchestrator_id="},
		}
	}

	if oaEnabled {
		pod.Spec.InitContainers[0].Env = append(pod.Spec.InitContainers[0].Env,
			corev1.EnvVar{Name: oneAgentInjectedEnvVarName, Value: "true"},
			corev1.EnvVar{Name: standalone.InstallerFlavorEnv, Value: arch.FlavorMultidistro},
			corev1.EnvVar{Name: standalone.InstallerTechEnv, Value: "all"},
			corev1.EnvVar{Name: standalone.InstallPathEnv, Value: "/opt/dynatrace/oneagent-paas"},
			corev1.EnvVar{Name: standalone.InstallerUrlEnv, Value: ""},
			corev1.EnvVar{Name: standalone.ModeEnv, Value: provisionedVolumeMode},
			corev1.EnvVar{Name: fmt.Sprintf(standalone.ContainerNameEnvTemplate, 1), Value: "test-container"},
			corev1.EnvVar{Name: fmt.Sprintf(standalone.ContainerImageEnvTemplate, 1), Value: "alpine"},
		)

		pod.Spec.InitContainers[0].VolumeMounts = append(pod.Spec.InitContainers[0].VolumeMounts,
			corev1.VolumeMount{Name: oneAgentBinVolumeName, MountPath: standalone.BinDirMount},
			corev1.VolumeMount{Name: oneAgentShareVolumeName, MountPath: standalone.ShareDirMount},
		)

		pod.Spec.Containers[0].VolumeMounts = append(pod.Spec.Containers[0].VolumeMounts,
			corev1.VolumeMount{Name: oneAgentShareVolumeName, MountPath: "/etc/ld.so.preload", SubPath: "ld.so.preload"},
			corev1.VolumeMount{Name: oneAgentBinVolumeName, MountPath: "/opt/dynatrace/oneagent-paas"},
			corev1.VolumeMount{
				Name:      oneAgentShareVolumeName,
				MountPath: "/var/lib/dynatrace/oneagent/agent/config/container.conf",
				SubPath:   "container_test-container.conf",
			},
		)

		pod.Spec.Containers[0].Env = append(pod.Spec.Containers[0].Env,
			corev1.EnvVar{Name: "LD_PRELOAD", Value: "/opt/dynatrace/oneagent-paas/agent/lib64/liboneagentproc.so"},
		)

		pod.Spec.Volumes = append(pod.Spec.Volumes,
			corev1.Volume{
				Name: oneAgentBinVolumeName,
				VolumeSource: corev1.VolumeSource{
					CSI: &corev1.CSIVolumeSource{
						Driver: dtcsi.DriverName,
						VolumeAttributes: map[string]string{
							csivolumes.CSIVolumeAttributeModeField:     appvolumes.Mode,
							csivolumes.CSIVolumeAttributeDynakubeField: dynakubeName,
						},
					},
				},
			},
			corev1.Volume{
				Name: oneAgentShareVolumeName,
				VolumeSource: corev1.VolumeSource{
					EmptyDir: &corev1.EmptyDirVolumeSource{},
				},
			},
		)
	} else {
		if len(pod.Spec.InitContainers) > 0 {
			pod.Spec.InitContainers[0].Env = append(pod.Spec.InitContainers[0].Env,
				corev1.EnvVar{Name: oneAgentInjectedEnvVarName, Value: "false"},
			)
		}
	}

	if diEnabled {
		if pod.ObjectMeta.Annotations == nil {
			pod.ObjectMeta.Annotations = make(map[string]string)
		}

		pod.Spec.InitContainers[0].Env = append(pod.Spec.InitContainers[0].Env,
			corev1.EnvVar{Name: dataIngestInjectedEnvVarName, Value: "true"},
			corev1.EnvVar{Name: workloadKindEnvVarName, Value: ""},
			corev1.EnvVar{Name: workloadNameEnvVarName, Value: "test-pod-12345"},
		)

		pod.Spec.InitContainers[0].VolumeMounts = append(pod.Spec.InitContainers[0].VolumeMounts,
			corev1.VolumeMount{Name: dataIngestVolumeName, MountPath: standalone.EnrichmentPath},
		)

		pod.Spec.Containers[0].VolumeMounts = append(pod.Spec.Containers[0].VolumeMounts,
			corev1.VolumeMount{Name: dataIngestVolumeName, MountPath: standalone.EnrichmentPath},
			corev1.VolumeMount{Name: dataIngestEndpointVolumeName, MountPath: "/var/lib/dynatrace/enrichment/endpoint"},
		)

		pod.Spec.Volumes = append(pod.Spec.Volumes,
			corev1.Volume{
				Name: dataIngestVolumeName,
				VolumeSource: corev1.VolumeSource{
					EmptyDir: &corev1.EmptyDirVolumeSource{},
				},
			},
			corev1.Volume{
				Name: dataIngestEndpointVolumeName,
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName: dtingestendpoint.SecretEndpointName,
					},
				},
			},
		)
	} else {
		if len(pod.Spec.InitContainers) > 0 {
			pod.Spec.InitContainers[0].Env = append(pod.Spec.InitContainers[0].Env,
				corev1.EnvVar{Name: dataIngestInjectedEnvVarName, Value: "false"},
			)
		}
	}

	sortPodInternals(&pod)
	return pod
}

func setEnvVar(_ *testing.T, pod *corev1.Pod, name string, value string) {
	if len(pod.Spec.InitContainers) == 0 {
		return
	}

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
	instance.Annotations[dynatracev1beta1.AnnotationFeatureEnableWebhookReinvocationPolicy] = "true"
	err = inj.client.Update(context.TODO(), instance)
	require.NoError(t, err)

	var testContainerName, testContainerImage = "test-container", "alpine"
	var thirdPartyContainerName, thirdPartyContainerImage = "third-party-container", "sidecar-image"
	basePod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "test-pod-12345",
			Namespace:   "test-namespace",
			Annotations: map[string]string{dtwebhook.AnnotationDynatraceInjected: "oneagent"}},
		//Annotations: map[string]string{dtwebhook.AnnotationDynatraceInjected: "data-ingest,oneagent"}},
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
	require.Equal(t, "DT_DEPLOYMENT_METADATA", updPod.Spec.Containers[1].Env[0].Name)

	var updInstallContainer = updPod.Spec.InitContainers[0]

	fmt.Println(updInstallContainer.Env)
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
