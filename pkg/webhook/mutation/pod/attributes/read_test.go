package attributes

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/metadataenrichment"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8senv"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

func TestGetNamespaceAnnotationAttributes(t *testing.T) {
	t.Run("stores keys with metadata prefix, stripping the prefix", func(t *testing.T) {
		attrs := newTestPodAttributes()
		ns := corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					metadataenrichment.Prefix + "my.attr": "value1",
					metadataenrichment.Prefix + "other":   "value2",
				},
			},
		}

		attrs.readNamespaceAnnotationAttributes(ns)

		assert.Equal(t, "value1", attrs.namespaceAnnotations["my.attr"])
		assert.Equal(t, "value2", attrs.namespaceAnnotations["other"])
	})

	t.Run("ignores keys without the metadata prefix", func(t *testing.T) {
		attrs := newTestPodAttributes()
		ns := corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					"unrelated.annotation/key":         "ignored",
					metadataenrichment.Prefix + "kept": "kept-value",
				},
			},
		}

		attrs.readNamespaceAnnotationAttributes(ns)

		assert.Len(t, attrs.namespaceAnnotations, 1)
		assert.Equal(t, "kept-value", attrs.namespaceAnnotations["kept"])
	})

	t.Run("empty annotations map results in empty namespaceAnnotations", func(t *testing.T) {
		attrs := newTestPodAttributes()
		attrs.readNamespaceAnnotationAttributes(corev1.Namespace{})
		assert.Empty(t, attrs.namespaceAnnotations)
	})
}

func TestGetPodAnnotationAttributes(t *testing.T) {
	t.Run("stores keys with metadata prefix, stripping the prefix", func(t *testing.T) {
		attrs := newTestPodAttributes()
		pod := corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					metadataenrichment.Prefix + "my.attr": "pod-value",
				},
			},
		}

		attrs.readPodAnnotationAttributes(pod)

		assert.Equal(t, "pod-value", attrs.podAnnotations["my.attr"])
	})

	t.Run("ignores keys without the metadata prefix", func(t *testing.T) {
		attrs := newTestPodAttributes()
		pod := corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					"unrelated/key":                    "ignored",
					metadataenrichment.Prefix + "kept": "kept-value",
				},
			},
		}

		attrs.readPodAnnotationAttributes(pod)

		assert.Len(t, attrs.podAnnotations, 1)
	})

	t.Run("empty annotations map results in empty podAnnotations", func(t *testing.T) {
		attrs := newTestPodAttributes()
		attrs.readPodAnnotationAttributes(corev1.Pod{})
		assert.Empty(t, attrs.podAnnotations)
	})
}

func TestGetFromEnrichmentRules(t *testing.T) {
	t.Run("LabelRule without target stores under computed rules key", func(t *testing.T) {
		attrs := newTestPodAttributes()
		ns := corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{"env": "production"},
			},
		}
		dk := dynakube.DynaKube{
			Status: dynakube.DynaKubeStatus{
				MetadataEnrichment: metadataenrichment.Status{
					Rules: []metadataenrichment.Rule{
						{Type: metadataenrichment.LabelRule, Source: "env"},
					},
				},
			},
		}

		attrs.applyEnrichmentRules(ns, dk)

		expectedKey := metadataenrichment.GetEmptyTargetEnrichmentKey(string(metadataenrichment.LabelRule), "env")
		assert.Equal(t, "production", attrs.rules[expectedKey])
		assert.Len(t, attrs.rules, 1)
	})

	t.Run("LabelRule with target stores in rules under the target key", func(t *testing.T) {
		attrs := newTestPodAttributes()
		ns := corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{"env": "staging"},
			},
		}
		dk := dynakube.DynaKube{
			Status: dynakube.DynaKubeStatus{
				MetadataEnrichment: metadataenrichment.Status{
					Rules: []metadataenrichment.Rule{
						{Type: metadataenrichment.LabelRule, Source: "env", Target: "custom.env"},
					},
				},
			},
		}

		attrs.applyEnrichmentRules(ns, dk)

		assert.Equal(t, "staging", attrs.rules["custom.env"])
		assert.Len(t, attrs.rules, 1)
	})

	t.Run("AnnotationRule reads from namespace annotations", func(t *testing.T) {
		attrs := newTestPodAttributes()
		ns := corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{"team": "backend"},
			},
		}
		dk := dynakube.DynaKube{
			Status: dynakube.DynaKubeStatus{
				MetadataEnrichment: metadataenrichment.Status{
					Rules: []metadataenrichment.Rule{
						{Type: metadataenrichment.AnnotationRule, Source: "team", Target: "team.name"},
					},
				},
			},
		}

		attrs.applyEnrichmentRules(ns, dk)

		assert.Equal(t, "backend", attrs.rules["team.name"])
	})

	t.Run("rule whose source is absent from namespace is skipped", func(t *testing.T) {
		attrs := newTestPodAttributes()
		dk := dynakube.DynaKube{
			Status: dynakube.DynaKubeStatus{
				MetadataEnrichment: metadataenrichment.Status{
					Rules: []metadataenrichment.Rule{
						{Type: metadataenrichment.LabelRule, Source: "missing-label"},
					},
				},
			},
		}

		attrs.applyEnrichmentRules(corev1.Namespace{}, dk)

		assert.Empty(t, attrs.rules)
	})

	t.Run("mix of target and no-target rules routes correctly", func(t *testing.T) {
		attrs := newTestPodAttributes()
		ns := corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{
					"env":  "prod",
					"team": "platform",
				},
			},
		}
		dk := dynakube.DynaKube{
			Status: dynakube.DynaKubeStatus{
				MetadataEnrichment: metadataenrichment.Status{
					Rules: []metadataenrichment.Rule{
						{Type: metadataenrichment.LabelRule, Source: "env"},
						{Type: metadataenrichment.LabelRule, Source: "team", Target: "custom.team"},
					},
				},
			},
		}

		attrs.applyEnrichmentRules(ns, dk)

		envKey := metadataenrichment.GetEmptyTargetEnrichmentKey(string(metadataenrichment.LabelRule), "env")
		assert.Equal(t, "prod", attrs.rules[envKey])
		assert.Equal(t, "platform", attrs.rules["custom.team"])
	})

	t.Run("K8S_NAMESPACE_LABEL with target stores in rules", func(t *testing.T) {
		attrs := newTestPodAttributes()
		ns := corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{"env": "production"},
			},
		}
		dk := dynakube.DynaKube{
			Status: dynakube.DynaKubeStatus{
				MetadataEnrichment: metadataenrichment.Status{
					Rules: []metadataenrichment.Rule{
						{Type: metadataenrichment.K8sNamespaceLabelRule, Source: "env", Target: "custom.env"},
					},
				},
			},
		}

		attrs.applyEnrichmentRules(ns, dk)

		assert.Equal(t, "production", attrs.rules["custom.env"])
		assert.Len(t, attrs.rules, 1)
	})

	t.Run("K8S_NAMESPACE_LABEL without target stores under computed rules key", func(t *testing.T) {
		attrs := newTestPodAttributes()
		ns := corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{"env": "production"},
			},
		}
		dk := dynakube.DynaKube{
			Status: dynakube.DynaKubeStatus{
				MetadataEnrichment: metadataenrichment.Status{
					Rules: []metadataenrichment.Rule{
						{Type: metadataenrichment.K8sNamespaceLabelRule, Source: "env"},
					},
				},
			},
		}

		attrs.applyEnrichmentRules(ns, dk)

		expectedKey := metadataenrichment.GetEmptyTargetEnrichmentKey(string(metadataenrichment.K8sNamespaceLabelRule), "env")
		assert.Equal(t, "production", attrs.rules[expectedKey])
		assert.Len(t, attrs.rules, 1)
	})

	t.Run("K8S_NAMESPACE_ANNOTATION with target stores in rules", func(t *testing.T) {
		attrs := newTestPodAttributes()
		ns := corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{"team": "backend"},
			},
		}
		dk := dynakube.DynaKube{
			Status: dynakube.DynaKubeStatus{
				MetadataEnrichment: metadataenrichment.Status{
					Rules: []metadataenrichment.Rule{
						{Type: metadataenrichment.K8sNamespaceAnnotationRule, Source: "team", Target: "team.name"},
					},
				},
			},
		}

		attrs.applyEnrichmentRules(ns, dk)

		assert.Equal(t, "backend", attrs.rules["team.name"])
		assert.Len(t, attrs.rules, 1)
	})

	t.Run("K8S_NAMESPACE_LABEL with absent source is skipped", func(t *testing.T) {
		attrs := newTestPodAttributes()
		dk := dynakube.DynaKube{
			Status: dynakube.DynaKubeStatus{
				MetadataEnrichment: metadataenrichment.Status{
					Rules: []metadataenrichment.Rule{
						{Type: metadataenrichment.K8sNamespaceLabelRule, Source: "missing-label"},
					},
				},
			},
		}

		attrs.applyEnrichmentRules(corev1.Namespace{}, dk)

		assert.Empty(t, attrs.rules)
	})

	t.Run("CUSTOM with target stores literal source in rules", func(t *testing.T) {
		attrs := newTestPodAttributes()
		dk := dynakube.DynaKube{
			Status: dynakube.DynaKubeStatus{
				MetadataEnrichment: metadataenrichment.Status{
					Rules: []metadataenrichment.Rule{
						{Type: metadataenrichment.CustomRule, Source: "my-literal-value", Target: "dt.custom"},
					},
				},
			},
		}

		attrs.applyEnrichmentRules(corev1.Namespace{}, dk)

		assert.Equal(t, "my-literal-value", attrs.rules["dt.custom"])
		assert.Len(t, attrs.rules, 1)
	})

	t.Run("CUSTOM without target is dropped", func(t *testing.T) {
		attrs := newTestPodAttributes()
		dk := dynakube.DynaKube{
			Status: dynakube.DynaKubeStatus{
				MetadataEnrichment: metadataenrichment.Status{
					Rules: []metadataenrichment.Rule{
						{Type: metadataenrichment.CustomRule, Source: "high prio!"},
					},
				},
			},
		}

		attrs.applyEnrichmentRules(corev1.Namespace{}, dk)

		assert.Empty(t, attrs.rules)
	})
}

func TestGetMetadataAnnotations(t *testing.T) {
	t.Run("collects namespace annotations, pod annotations, and enrichment rules", func(t *testing.T) {
		attrs := newTestPodAttributes()
		ns := corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{metadataenrichment.Prefix + "ns-key": "ns-val"},
				Labels:      map[string]string{"env": "prod"},
			},
		}
		pod := corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{metadataenrichment.Prefix + "pod-key": "pod-val"},
			},
		}
		dk := dynakube.DynaKube{
			Status: dynakube.DynaKubeStatus{
				MetadataEnrichment: metadataenrichment.Status{
					Rules: []metadataenrichment.Rule{
						{Type: metadataenrichment.LabelRule, Source: "env", Target: "custom.env"},
					},
				},
			},
		}

		attrs.readMetadataAnnotations(dtwebhook.BaseRequest{Pod: &pod, Namespace: ns, DynaKube: dk})

		assert.Equal(t, "ns-val", attrs.namespaceAnnotations["ns-key"])
		assert.Equal(t, "pod-val", attrs.podAnnotations["pod-key"])
		assert.Equal(t, "prod", attrs.rules["custom.env"])
	})
}

func TestCopyMetadataFromNamespace(t *testing.T) {
	t.Run("namespace annotations appear in JSON block", func(t *testing.T) {
		request := createTestMutationRequest(t, nil, nil)
		request.Namespace.Labels = map[string]string{
			metadataenrichment.Prefix + "nocopyoflabels": "nocopyoflabels",
			"test-label": "test-value",
		}
		request.Namespace.Annotations = map[string]string{
			metadataenrichment.Prefix + "copyofannotations": "copyofannotations",
			"test-annotation": "test-value",
		}

		attrs, err := NewPodAttributes(t.Context(), *request.BaseRequest, fake.NewClient())
		require.NoError(t, err)
		require.NoError(t, attrs.ApplyAnnotationsToPod(request.Pod))

		require.Len(t, request.Pod.Annotations, 1)
		require.Empty(t, request.Pod.Labels)

		var actualMetadataJSON map[string]string

		require.NoError(t, json.Unmarshal([]byte(request.Pod.Annotations[metadataenrichment.Annotation]), &actualMetadataJSON))

		expectedMetadataJSON := map[string]string{
			"copyofannotations": "copyofannotations",
			"k8s.workload.kind": "pod",
			"k8s.workload.name": "test-pod",
		}
		require.Equal(t, expectedMetadataJSON, actualMetadataJSON)
	})

	t.Run("should copy all labels and annotations defined without override", func(t *testing.T) {
		request := createTestMutationRequest(t, nil, nil)
		request.Pod.Annotations = map[string]string{
			metadataenrichment.Prefix + "copyofannotations": "do-not-overwrite",
		}
		request.Namespace.Labels = map[string]string{
			metadataenrichment.Prefix + "nocopyoflabels":   "nocopyoflabels",
			metadataenrichment.Prefix + "copyifruleexists": "copyifruleexists",
			"test-label": "test-value",
		}
		request.Namespace.Annotations = map[string]string{
			metadataenrichment.Prefix + "copyofannotations": "copyofannotations",
			"test-annotation": "test-value",
		}
		request.DynaKube.Status.MetadataEnrichment.Rules = []metadataenrichment.Rule{
			{
				Type:   metadataenrichment.AnnotationRule,
				Source: "test-annotation",
				Target: "dt.test-annotation",
			},
			{
				Type:   metadataenrichment.LabelRule,
				Source: "test-label",
				Target: "test-label",
			},
			{
				Type:   metadataenrichment.LabelRule,
				Source: metadataenrichment.Prefix + "copyifruleexists",
				Target: "dt.copyifruleexists",
			},
			{
				Type:   metadataenrichment.LabelRule,
				Source: "does-not-exist-in-namespace",
				Target: "dt.does-not-exist-in-namespace",
			},
		}

		attrs, err := NewPodAttributes(t.Context(), *request.BaseRequest, fake.NewClient())
		require.NoError(t, err)
		require.NoError(t, attrs.ApplyAnnotationsToPod(request.Pod))

		// pod was pre-seeded with one annotation; ApplyAnnotationsToPod only adds the JSON block
		require.Len(t, request.Pod.Annotations, 2)
		require.Empty(t, request.Pod.Labels)

		var actualMetadataJSON map[string]string

		require.NoError(t, json.Unmarshal([]byte(request.Pod.Annotations[metadataenrichment.Annotation]), &actualMetadataJSON))

		expectedMetadataJSON := map[string]string{
			"copyofannotations":   "do-not-overwrite",
			"dt.copyifruleexists": "copyifruleexists",
			"dt.test-annotation":  "test-value",
			"test-label":          "test-value",
			"k8s.workload.kind":   "pod",
			"k8s.workload.name":   "test-pod",
		}
		require.Equal(t, expectedMetadataJSON, actualMetadataJSON)
	})

	t.Run("are custom rule types handled correctly", func(t *testing.T) {
		request := createTestMutationRequest(t, nil, nil)
		request.Namespace.Labels = map[string]string{
			"test":  "test-label-value",
			"test2": "test-label-value2",
			"test3": "test-label-value3",
			"test4": "test-label-value4",
		}
		request.Namespace.Annotations = map[string]string{
			"test":  "test-annotation-value",
			"test3": "test-annotation-value3",
			"test4": "test-annotation-value4",
		}

		request.DynaKube.Status.MetadataEnrichment.Rules = []metadataenrichment.Rule{
			{
				Type:   metadataenrichment.LabelRule,
				Source: "test",
				Target: "dt.test-label",
			},
			{
				Type:   metadataenrichment.LabelRule,
				Source: "test2",
				Target: "", // mapping missing => rule used as primary grail tag with the source name for data enrichment
			},
			{
				Type:   metadataenrichment.AnnotationRule,
				Source: "test3",
				Target: "dt.test-annotation",
			},
			{
				Type:   metadataenrichment.AnnotationRule,
				Source: "test4",
				Target: "", // mapping missing => rule used as primary grail tag with the source name for data enrichment
			},
			{
				Type:   metadataenrichment.CustomRule,
				Source: "my-custom-value",
				Target: "dt.custom",
			},
		}

		attrs, err := NewPodAttributes(t.Context(), *request.BaseRequest, fake.NewClient())
		require.NoError(t, err)
		require.NoError(t, attrs.ApplyAnnotationsToPod(request.Pod))

		require.Len(t, request.Pod.Annotations, 1)
		require.Empty(t, request.Pod.Labels)

		var actualMetadataJSON map[string]string

		require.NoError(t, json.Unmarshal([]byte(request.Pod.Annotations[metadataenrichment.Annotation]), &actualMetadataJSON))
		require.Len(t, actualMetadataJSON, 7)

		expectedMetadataJSON := map[string]string{
			"dt.test-annotation": "test-annotation-value3",
			"dt.test-label":      "test-label-value",
			"dt.custom":          "my-custom-value",
			metadataenrichment.GetEmptyTargetEnrichmentKey(string(metadataenrichment.AnnotationRule), "test4"): "test-annotation-value4",
			metadataenrichment.GetEmptyTargetEnrichmentKey(string(metadataenrichment.LabelRule), "test2"):      "test-label-value2",
			"k8s.workload.kind": "pod",
			"k8s.workload.name": "test-pod",
		}
		require.Equal(t, expectedMetadataJSON, actualMetadataJSON)
	})

	t.Run("should copy all annotations without rules", func(t *testing.T) {
		request := createTestMutationRequest(t, nil, nil)

		request.Pod.Annotations = map[string]string{
			metadataenrichment.Prefix + "someannotation": "do-not-overwrite",
		}

		request.Namespace.Annotations = map[string]string{
			metadataenrichment.Prefix + "someannotation":    "somevalue",
			metadataenrichment.Prefix + "anotherannotation": "othervalue",
			"test-annotation": "test-value",
		}

		attrs, err := NewPodAttributes(t.Context(), *request.BaseRequest, fake.NewClient())
		require.NoError(t, err)
		require.NoError(t, attrs.ApplyAnnotationsToPod(request.Pod))

		// pod was pre-seeded with one annotation; ApplyAnnotationsToPod only adds the JSON block
		require.Len(t, request.Pod.Annotations, 2)

		require.Equal(t, "do-not-overwrite", request.Pod.Annotations[metadataenrichment.Prefix+"someannotation"])

		var actualMetadataJSON map[string]string

		require.NoError(t, json.Unmarshal([]byte(request.Pod.Annotations[metadataenrichment.Annotation]), &actualMetadataJSON))

		expectedMetadataJSON := map[string]string{
			"someannotation":    "do-not-overwrite",
			"anotherannotation": "othervalue",
			"k8s.workload.kind": "pod",
			"k8s.workload.name": "test-pod",
		}
		require.Equal(t, expectedMetadataJSON, actualMetadataJSON)
	})
}

func createTestMutationRequest(t *testing.T, dk *dynakube.DynaKube, annotations map[string]string) *dtwebhook.MutationRequest {
	if dk == nil {
		dk = &dynakube.DynaKube{}
	}

	return dtwebhook.NewMutationRequest(
		t.Context(),
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
		TypeMeta: metav1.TypeMeta{
			Kind: "Pod",
		},
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

func TestReadWorkloadInfoAttributes(t *testing.T) {
	t.Run("sets workload kind and name from pod with no owner (pod is its own root owner)", func(t *testing.T) {
		ctx := t.Context()
		attrs := newTestPodAttributes()
		pod := corev1.Pod{
			TypeMeta:   metav1.TypeMeta{Kind: "Pod", APIVersion: "v1"},
			ObjectMeta: metav1.ObjectMeta{Name: "my-pod", Namespace: "my-ns"},
		}
		request := dtwebhook.BaseRequest{
			Pod:       &pod,
			Namespace: corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "my-ns"}},
		}

		err := attrs.readWorkloadInfoAttributes(ctx, request, fake.NewClient())

		require.NoError(t, err)
		assert.Equal(t, "pod", attrs.workloadInfo[K8sWorkloadKindAttr])
		assert.Equal(t, "my-pod", attrs.workloadInfo[K8sWorkloadNameAttr])
	})

	t.Run("propagates error when owner lookup fails", func(t *testing.T) {
		ctx := t.Context()
		attrs := newTestPodAttributes()
		pod := corev1.Pod{
			TypeMeta: metav1.TypeMeta{Kind: "Pod", APIVersion: "v1"},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "my-pod",
				Namespace: "my-ns",
				OwnerReferences: []metav1.OwnerReference{
					{APIVersion: "apps/v1", Kind: "Deployment", Name: "my-deploy", Controller: new(true)},
				},
			},
		}
		request := dtwebhook.BaseRequest{
			Pod:       &pod,
			Namespace: corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "my-ns"}},
		}
		failClient := fake.NewClientWithInterceptors(interceptor.Funcs{
			Get: func(ctx context.Context, c client.WithWatch, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
				return errors.New("boom")
			},
		})

		err := attrs.readWorkloadInfoAttributes(ctx, request, failClient)

		assert.Error(t, err)
	})
}

func TestReadPodAttributes(t *testing.T) {
	t.Run("appends three env vars with field-path references", func(t *testing.T) {
		attrs := newTestPodAttributes()
		request := dtwebhook.BaseRequest{
			Pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{Namespace: "my-ns"},
			},
			DynaKube: dynakube.DynaKube{},
		}

		attrs.readPodAttributes(request)

		require.Len(t, attrs.podEnvVars, 3)

		podNameEnv := k8senv.Find(attrs.podEnvVars, K8sPodNameEnv)
		require.NotNil(t, podNameEnv)
		require.NotNil(t, podNameEnv.ValueFrom)
		assert.Equal(t, "metadata.name", podNameEnv.ValueFrom.FieldRef.FieldPath)

		podUIDEnv := k8senv.Find(attrs.podEnvVars, K8sPodUIDEnv)
		require.NotNil(t, podUIDEnv)
		require.NotNil(t, podUIDEnv.ValueFrom)
		assert.Equal(t, "metadata.uid", podUIDEnv.ValueFrom.FieldRef.FieldPath)

		nodeNameEnv := k8senv.Find(attrs.podEnvVars, K8sNodeNameEnv)
		require.NotNil(t, nodeNameEnv)
		require.NotNil(t, nodeNameEnv.ValueFrom)
		assert.Equal(t, "spec.nodeName", nodeNameEnv.ValueFrom.FieldRef.FieldPath)
	})

	t.Run("sets podInfo with env var references and namespace name", func(t *testing.T) {
		attrs := newTestPodAttributes()
		request := dtwebhook.BaseRequest{
			Pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{Namespace: "my-ns"},
			},
			DynaKube: dynakube.DynaKube{},
		}

		attrs.readPodAttributes(request)

		assert.Equal(t, "$(K8S_PODNAME)", attrs.podInfo[K8sPodNameAttr])
		assert.Equal(t, "$(K8S_PODUID)", attrs.podInfo[K8sPodUIDAttr])
		assert.Equal(t, "$(K8S_NODE_NAME)", attrs.podInfo[K8sNodeNameAttr])
		assert.Equal(t, "my-ns", attrs.podInfo[K8sNamespaceNameAttr])
	})

	t.Run("sets clusterInfo from DynaKube status", func(t *testing.T) {
		attrs := newTestPodAttributes()
		request := dtwebhook.BaseRequest{
			Pod: &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Namespace: "ns"}},
			DynaKube: dynakube.DynaKube{
				Status: dynakube.DynaKubeStatus{
					KubeSystemUUID:        "uid-123",
					KubernetesClusterName: "my-cluster",
					KubernetesClusterMEID: "KUBERNETES_CLUSTER-ABC",
				},
			},
		}

		attrs.readPodAttributes(request)

		assert.Equal(t, "uid-123", attrs.clusterInfo[K8sClusterUIDAttr])
		assert.Equal(t, "my-cluster", attrs.clusterInfo[K8sClusterNameAttr])
		assert.Equal(t, "KUBERNETES_CLUSTER-ABC", attrs.clusterInfo[K8sDTClusterEntityAttr])
	})
}
