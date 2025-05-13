package metadata

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestCopyMetadataFromNamespace(t *testing.T) {
	t.Run("should copy annotations not labels with prefix from namespace to pod", func(t *testing.T) {
		request := createTestMutationRequest(nil, nil)
		request.Namespace.Labels = map[string]string{
			dynakube.MetadataPrefix + "nocopyoflabels": "nocopyoflabels",
			"test-label": "test-value",
		}
		request.Namespace.Annotations = map[string]string{
			dynakube.MetadataPrefix + "copyofannotations": "copyofannotations",
			"test-annotation": "test-value",
		}

		CopyMetadataFromNamespace(request.Pod, request.Namespace, request.DynaKube)
		require.Len(t, request.Pod.Annotations, 1)
		require.Empty(t, request.Pod.Labels)
		require.Equal(t, "copyofannotations", request.Pod.Annotations[dynakube.MetadataPrefix+"copyofannotations"])
	})

	t.Run("should copy all labels and annotations defined without override", func(t *testing.T) {
		request := createTestMutationRequest(nil, nil)
		request.Namespace.Labels = map[string]string{
			dynakube.MetadataPrefix + "nocopyoflabels":   "nocopyoflabels",
			dynakube.MetadataPrefix + "copyifruleexists": "copyifruleexists",
			"test-label": "test-value",
		}
		request.Namespace.Annotations = map[string]string{
			dynakube.MetadataPrefix + "copyofannotations": "copyofannotations",
			"test-annotation": "test-value",
		}

		request.DynaKube.Status.MetadataEnrichment.Rules = []dynakube.EnrichmentRule{
			{
				Type:   dynakube.EnrichmentAnnotationRule,
				Source: "test-annotation",
				Target: "dt.test-annotation",
			},
			{
				Type:   dynakube.EnrichmentLabelRule,
				Source: "test-label",
				Target: "test-label",
			},
			{
				Type:   dynakube.EnrichmentLabelRule,
				Source: dynakube.MetadataPrefix + "copyifruleexists",
				Target: "dt.copyifruleexists",
			},
			{
				Type:   dynakube.EnrichmentLabelRule,
				Source: "does-not-exist-in-namespace",
				Target: "dt.does-not-exist-in-namespace",
			},
		}
		request.Pod.Annotations = map[string]string{
			dynakube.MetadataPrefix + "copyofannotations": "do-not-overwrite",
		}

		CopyMetadataFromNamespace(request.Pod, request.Namespace, request.DynaKube)
		require.Len(t, request.Pod.Annotations, 4)
		require.Empty(t, request.Pod.Labels)

		require.Equal(t, "do-not-overwrite", request.Pod.Annotations[dynakube.MetadataPrefix+"copyofannotations"])
		require.Equal(t, "copyifruleexists", request.Pod.Annotations[dynakube.MetadataPrefix+"dt.copyifruleexists"])

		require.Equal(t, "test-value", request.Pod.Annotations[dynakube.MetadataPrefix+"dt.test-annotation"])
		require.Equal(t, "test-value", request.Pod.Annotations[dynakube.MetadataPrefix+"test-label"])
	})

	t.Run("are custom rule types handled correctly", func(t *testing.T) {
		request := createTestMutationRequest(nil, nil)
		request.Namespace.Labels = map[string]string{
			"test":  "test-label-value",
			"test2": "test-label-value2",
			"test3": "test-label-value3",
		}
		request.Namespace.Annotations = map[string]string{
			"test":  "test-annotation-value",
			"test2": "test-annotation-value2",
			"test3": "test-annotation-value3",
		}

		request.DynaKube.Status.MetadataEnrichment.Rules = []dynakube.EnrichmentRule{
			{
				Type:   dynakube.EnrichmentLabelRule,
				Source: "test",
				Target: "dt.test-label",
			},
			{
				Type:   dynakube.EnrichmentAnnotationRule,
				Source: "test2",
				Target: "dt.test-annotation",
			},
			{
				Type:   dynakube.EnrichmentAnnotationRule,
				Source: "test3",
				Target: "", // mapping missing => rule ignored
			},
		}

		CopyMetadataFromNamespace(request.Pod, request.Namespace, request.DynaKube)
		require.Len(t, request.Pod.Annotations, 2)
		require.Empty(t, request.Pod.Labels)
		require.Equal(t, "test-label-value", request.Pod.Annotations[dynakube.MetadataPrefix+"dt.test-label"])
		require.Equal(t, "test-annotation-value2", request.Pod.Annotations[dynakube.MetadataPrefix+"dt.test-annotation"])
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
