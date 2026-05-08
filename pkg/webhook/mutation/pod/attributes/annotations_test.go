package attributes

import (
	"encoding/json"
	"maps"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/metadataenrichment"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

		attrs.readFromEnrichmentRules(ns, dk)

		expectedKey := metadataenrichment.GetEmptyTargetEnrichmentKey(string(metadataenrichment.LabelRule), "env")
		assert.Equal(t, "production", attrs.rules[expectedKey])
		assert.Empty(t, attrs.rulesPropagate)
	})

	t.Run("LabelRule with target stores in rulesPropagate under the target key", func(t *testing.T) {
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

		attrs.readFromEnrichmentRules(ns, dk)

		assert.Equal(t, "staging", attrs.rulesPropagate["custom.env"])
		assert.Empty(t, attrs.rules)
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

		attrs.readFromEnrichmentRules(ns, dk)

		assert.Equal(t, "backend", attrs.rulesPropagate["team.name"])
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

		attrs.readFromEnrichmentRules(corev1.Namespace{}, dk)

		assert.Empty(t, attrs.rules)
		assert.Empty(t, attrs.rulesPropagate)
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

		attrs.readFromEnrichmentRules(ns, dk)

		envKey := metadataenrichment.GetEmptyTargetEnrichmentKey(string(metadataenrichment.LabelRule), "env")
		assert.Equal(t, "prod", attrs.rules[envKey])
		assert.Equal(t, "platform", attrs.rulesPropagate["custom.team"])
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
		assert.Equal(t, "prod", attrs.rulesPropagate["custom.env"])
	})
}

func TestCopyMetadataFromNamespace(t *testing.T) {
	t.Run("should copy annotations not labels with prefix from namespace to pod", func(t *testing.T) {
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

		require.Len(t, request.Pod.Annotations, 4)
		require.Empty(t, request.Pod.Labels)
		require.Equal(t, "copyofannotations", request.Pod.Annotations[metadataenrichment.Prefix+"copyofannotations"])

		checkDefaultAnnotations(t, *request.Pod)

		var actualMetadataJSON map[string]string

		require.NoError(t, json.Unmarshal([]byte(request.Pod.Annotations[metadataenrichment.Annotation]), &actualMetadataJSON))

		expectedMetadataJSON := map[string]string{
			"copyofannotations": "copyofannotations",
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

		require.Len(t, request.Pod.Annotations, 7)
		require.Empty(t, request.Pod.Labels)

		require.Equal(t, "do-not-overwrite", request.Pod.Annotations[metadataenrichment.Prefix+"copyofannotations"])
		require.Equal(t, "copyifruleexists", request.Pod.Annotations[metadataenrichment.Prefix+"dt.copyifruleexists"])

		require.Equal(t, "test-value", request.Pod.Annotations[metadataenrichment.Prefix+"dt.test-annotation"])
		require.Equal(t, "test-value", request.Pod.Annotations[metadataenrichment.Prefix+"test-label"])

		checkDefaultAnnotations(t, *request.Pod)

		var actualMetadataJSON map[string]string

		require.NoError(t, json.Unmarshal([]byte(request.Pod.Annotations[metadataenrichment.Annotation]), &actualMetadataJSON))

		expectedMetadataJSON := map[string]string{
			"copyofannotations":   "do-not-overwrite",
			"dt.copyifruleexists": "copyifruleexists",
			"dt.test-annotation":  "test-value",
			"test-label":          "test-value",
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
		}

		attrs, err := NewPodAttributes(t.Context(), *request.BaseRequest, fake.NewClient())
		require.NoError(t, err)
		require.NoError(t, attrs.ApplyAnnotationsToPod(request.Pod))

		require.Len(t, request.Pod.Annotations, 5)
		require.Empty(t, request.Pod.Labels)
		require.Equal(t, "test-label-value", request.Pod.Annotations[metadataenrichment.Prefix+"dt.test-label"])
		require.Equal(t, "test-annotation-value3", request.Pod.Annotations[metadataenrichment.Prefix+"dt.test-annotation"])

		checkDefaultAnnotations(t, *request.Pod)

		var actualMetadataJSON map[string]string

		require.NoError(t, json.Unmarshal([]byte(request.Pod.Annotations[metadataenrichment.Annotation]), &actualMetadataJSON))
		require.Len(t, actualMetadataJSON, 4)

		expectedMetadataJSON := map[string]string{
			"dt.test-annotation": "test-annotation-value3",
			"dt.test-label":      "test-label-value",
			metadataenrichment.GetEmptyTargetEnrichmentKey(string(metadataenrichment.AnnotationRule), "test4"): "test-annotation-value4",
			metadataenrichment.GetEmptyTargetEnrichmentKey(string(metadataenrichment.LabelRule), "test2"):      "test-label-value2",
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

		require.Len(t, request.Pod.Annotations, 5)

		require.Equal(t, "do-not-overwrite", request.Pod.Annotations[metadataenrichment.Prefix+"someannotation"])

		require.Equal(t, "othervalue", request.Pod.Annotations[metadataenrichment.Prefix+"anotherannotation"])

		checkDefaultAnnotations(t, *request.Pod)

		var actualMetadataJSON map[string]string

		require.NoError(t, json.Unmarshal([]byte(request.Pod.Annotations[metadataenrichment.Annotation]), &actualMetadataJSON))

		expectedMetadataJSON := map[string]string{
			"someannotation":    "do-not-overwrite",
			"anotherannotation": "othervalue",
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

func mergeWithDefaultAnnotations(expected map[string]string) map[string]string {
	maps.Copy(expected, map[string]string{
		"k8s.workload.kind": "pod",
		"k8s.workload.name": "test-pod",
	})

	return expected
}

func checkDefaultAnnotations(t *testing.T, pod corev1.Pod) {
	defaults := mergeWithDefaultAnnotations(map[string]string{})

	for key, value := range defaults {
		assert.Equal(t, value, pod.Annotations[metadataenrichment.Prefix+key])
	}
}
