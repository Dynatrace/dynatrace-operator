package attributes

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/metadataenrichment"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestCopyMetadataFromNamespace(t *testing.T) {
	t.Run("should copy annotations not labels with prefix from namespace to pod", func(t *testing.T) {
		request := createTestMutationRequest(nil, nil)
		request.Namespace.Labels = map[string]string{
			metadataenrichment.Prefix + "nocopyoflabels": "nocopyoflabels",
			"test-label": "test-value",
		}
		request.Namespace.Annotations = map[string]string{
			metadataenrichment.Prefix + "copyofannotations": "copyofannotations",
			"test-annotation": "test-value",
		}

		copyMetadataFromNamespace(request.BaseRequest)
		require.Len(t, request.Pod.Annotations, 2)
		require.Empty(t, request.Pod.Labels)
		require.Equal(t, "copyofannotations", request.Pod.Annotations[metadataenrichment.Prefix+"copyofannotations"])

		var actualMetadataJSON map[string]string

		require.NoError(t, json.Unmarshal([]byte(request.Pod.Annotations[metadataenrichment.Annotation]), &actualMetadataJSON))

		expectedMetadataJSON := map[string]string{
			"copyofannotations": "copyofannotations",
		}
		require.Equal(t, expectedMetadataJSON, actualMetadataJSON)
	})

	t.Run("should copy all labels and annotations defined without override", func(t *testing.T) {
		request := createTestMutationRequest(nil, nil)
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

		copyMetadataFromNamespace(request.BaseRequest)
		require.Len(t, request.Pod.Annotations, 5)
		require.Empty(t, request.Pod.Labels)

		require.Equal(t, "do-not-overwrite", request.Pod.Annotations[metadataenrichment.Prefix+"copyofannotations"])
		require.Equal(t, "copyifruleexists", request.Pod.Annotations[metadataenrichment.Prefix+"dt.copyifruleexists"])

		require.Equal(t, "test-value", request.Pod.Annotations[metadataenrichment.Prefix+"dt.test-annotation"])
		require.Equal(t, "test-value", request.Pod.Annotations[metadataenrichment.Prefix+"test-label"])

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
		request := createTestMutationRequest(nil, nil)
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

		copyMetadataFromNamespace(request.BaseRequest)
		require.Len(t, request.Pod.Annotations, 3)
		require.Empty(t, request.Pod.Labels)
		require.Equal(t, "test-label-value", request.Pod.Annotations[metadataenrichment.Prefix+"dt.test-label"])
		require.Equal(t, "test-annotation-value3", request.Pod.Annotations[metadataenrichment.Prefix+"dt.test-annotation"])

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
		request := createTestMutationRequest(nil, nil)

		request.Pod.Annotations = map[string]string{
			metadataenrichment.Prefix + "someannotation": "do-not-overwrite",
		}

		request.Namespace.Annotations = map[string]string{
			metadataenrichment.Prefix + "someannotation":    "somevalue",
			metadataenrichment.Prefix + "anotherannotation": "othervalue",
			"test-annotation": "test-value",
		}

		annotations := copyMetadataFromNamespace(request.BaseRequest)

		require.Len(t, annotations, 2)
		require.Len(t, request.Pod.Annotations, 3)

		require.Equal(t, "do-not-overwrite", request.Pod.Annotations[metadataenrichment.Prefix+"someannotation"])

		require.Equal(t, "othervalue", request.Pod.Annotations[metadataenrichment.Prefix+"anotherannotation"])

		var actualMetadataJSON map[string]string

		require.NoError(t, json.Unmarshal([]byte(request.Pod.Annotations[metadataenrichment.Annotation]), &actualMetadataJSON))

		expectedMetadataJSON := map[string]string{
			"someannotation":    "do-not-overwrite",
			"anotherannotation": "othervalue",
		}
		require.Equal(t, expectedMetadataJSON, actualMetadataJSON)
	})
}

func createTestMutationRequest(dk *dynakube.DynaKube, annotations map[string]string) *dtwebhook.MutationRequest {
	if dk == nil {
		dk = &dynakube.DynaKube{}
	}

	return dtwebhook.NewMutationRequest(
		context.Background(),
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
