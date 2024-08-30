package metadata

import (
	"encoding/json"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/env"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCopyMetadataFromNamespace(t *testing.T) {
	t.Run("should copy annotations not labels with prefix from namespace to pod", func(t *testing.T) {
		mutator := createTestPodMutator(nil)
		request := createTestMutationRequest(nil, nil, false)
		request.Namespace.Labels = map[string]string{
			dynakube.MetadataPrefix + "nocopyoflabels": "nocopyoflabels",
			"test-label": "test-value",
		}
		request.Namespace.Annotations = map[string]string{
			dynakube.MetadataPrefix + "copyofannotations": "copyofannotations",
			"test-annotation": "test-value",
		}

		require.False(t, mutator.Injected(request.BaseRequest))
		copyMetadataFromNamespace(request.Pod, request.Namespace, request.DynaKube)
		require.Len(t, request.Pod.Annotations, 1)
		require.Empty(t, request.Pod.Labels)
		require.Equal(t, "copyofannotations", request.Pod.Annotations[dynakube.MetadataPrefix+"copyofannotations"])
	})

	t.Run("should copy all labels and annotations defined without override", func(t *testing.T) {
		mutator := createTestPodMutator(nil)
		request := createTestMutationRequest(nil, nil, false)
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

		require.False(t, mutator.Injected(request.BaseRequest))
		copyMetadataFromNamespace(request.Pod, request.Namespace, request.DynaKube)
		require.Len(t, request.Pod.Annotations, 4)
		require.Empty(t, request.Pod.Labels)

		require.Equal(t, "do-not-overwrite", request.Pod.Annotations[dynakube.MetadataPrefix+"copyofannotations"])
		require.Equal(t, "copyifruleexists", request.Pod.Annotations[dynakube.MetadataPrefix+"dt.copyifruleexists"])

		require.Equal(t, "test-value", request.Pod.Annotations[dynakube.MetadataPrefix+"dt.test-annotation"])
		require.Equal(t, "test-value", request.Pod.Annotations[dynakube.MetadataPrefix+"test-label"])
	})

	t.Run("are custom rule types handled correctly", func(t *testing.T) {
		mutator := createTestPodMutator(nil)
		request := createTestMutationRequest(nil, nil, false)
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

		require.False(t, mutator.Injected(request.BaseRequest))
		copyMetadataFromNamespace(request.Pod, request.Namespace, request.DynaKube)
		require.Len(t, request.Pod.Annotations, 2)
		require.Empty(t, request.Pod.Labels)
		require.Equal(t, "test-label-value", request.Pod.Annotations[dynakube.MetadataPrefix+"dt.test-label"])
		require.Equal(t, "test-annotation-value2", request.Pod.Annotations[dynakube.MetadataPrefix+"dt.test-annotation"])
	})
}

func TestAddMetadataToInitEnv(t *testing.T) {
	t.Run("should copy annotations not labels with prefix from pod to env", func(t *testing.T) {
		expectedKeys := []string{
			"beep",
			"boop",
			"hello",
		}
		notExpectedKey := "no-prop"
		request := createTestMutationRequest(nil, nil, false)
		request.Pod.Labels = map[string]string{
			dynakube.MetadataPrefix + notExpectedKey: "beep-boop",
			"test-label":                             "boom",
		}
		request.Pod.Annotations = map[string]string{
			"test-annotation": "boom",
		}

		for _, key := range expectedKeys {
			request.Pod.Annotations[dynakube.MetadataPrefix+key] = key + "-value"
		}

		addMetadataToInitEnv(request.Pod, request.InstallContainer)

		annotationsEnv := env.FindEnvVar(request.InstallContainer.Env, consts.EnrichmentWorkloadAnnotationsEnv)
		require.NotNil(t, annotationsEnv)

		propagatedAnnotations := map[string]string{}
		err := json.Unmarshal([]byte(annotationsEnv.Value), &propagatedAnnotations)
		require.NoError(t, err)

		for _, key := range expectedKeys {
			require.Contains(t, propagatedAnnotations, key)
			assert.Equal(t, key+"-value", propagatedAnnotations[key])
			assert.NotContains(t, propagatedAnnotations, notExpectedKey)
		}
	})
}
