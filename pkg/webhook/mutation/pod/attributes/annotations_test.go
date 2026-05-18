package attributes

import (
	"encoding/json"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/metadataenrichment"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestSetPodMetadataJSONAnnotation(t *testing.T) {
	parseJSON := func(t *testing.T, pod *corev1.Pod) map[string]string {
		t.Helper()
		jsonVal, ok := pod.Annotations[metadataenrichment.Annotation]
		require.True(t, ok, "expected JSON annotation to be set")
		var parsed map[string]string
		require.NoError(t, json.Unmarshal([]byte(jsonVal), &parsed))

		return parsed
	}

	t.Run("rules override namespaceAnnotations for shared key", func(t *testing.T) {
		attrs := newTestPodAttributes()
		attrs.namespaceAnnotations["shared.key"] = "from-ns"
		attrs.rules["shared.key"] = "from-rules"
		pod := &corev1.Pod{}

		require.NoError(t, attrs.setPodMetadataJSONAnnotation(pod))
		assert.Equal(t, "from-rules", parseJSON(t, pod)["shared.key"])
	})

	t.Run("rulesPropagate overrides rules for shared key", func(t *testing.T) {
		attrs := newTestPodAttributes()
		attrs.rules["shared.key"] = "from-rules"
		attrs.rulesPropagate["shared.key"] = "from-rulesPropagate"
		pod := &corev1.Pod{}

		require.NoError(t, attrs.setPodMetadataJSONAnnotation(pod))
		assert.Equal(t, "from-rulesPropagate", parseJSON(t, pod)["shared.key"])
	})

	t.Run("podAnnotations overrides rulesPropagate for shared key", func(t *testing.T) {
		attrs := newTestPodAttributes()
		attrs.rulesPropagate["shared.key"] = "from-rulesPropagate"
		attrs.podAnnotations["shared.key"] = "from-pod"
		pod := &corev1.Pod{}

		require.NoError(t, attrs.setPodMetadataJSONAnnotation(pod))
		assert.Equal(t, "from-pod", parseJSON(t, pod)["shared.key"])
	})

	t.Run("all sources merged with correct precedence", func(t *testing.T) {
		attrs := newTestPodAttributes()
		attrs.namespaceAnnotations["ns-only"] = "ns-val"
		attrs.namespaceAnnotations["shared.key"] = "from-ns"
		attrs.rules["rules-only"] = "rules-val"
		attrs.rules["shared.key"] = "from-rules"
		attrs.rulesPropagate["propagate-only"] = "propagate-val"
		attrs.rulesPropagate["shared.key"] = "from-rulesPropagate"
		attrs.podAnnotations["pod-only"] = "pod-val"
		attrs.podAnnotations["shared.key"] = "from-pod"
		pod := &corev1.Pod{}

		require.NoError(t, attrs.setPodMetadataJSONAnnotation(pod))
		parsed := parseJSON(t, pod)
		assert.Equal(t, "from-pod", parsed["shared.key"])
		assert.Equal(t, "ns-val", parsed["ns-only"])
		assert.Equal(t, "rules-val", parsed["rules-only"])
		assert.Equal(t, "propagate-val", parsed["propagate-only"])
		assert.Equal(t, "pod-val", parsed["pod-only"])
	})

	t.Run("does not overwrite existing JSON annotation", func(t *testing.T) {
		existingJSON := `{"existing":"value"}`
		pod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{metadataenrichment.Annotation: existingJSON},
			},
		}
		attrs := newTestPodAttributes()
		attrs.rulesPropagate["custom-annotation1"] = "foobar"
		err := attrs.setPodMetadataJSONAnnotation(pod)

		require.NoError(t, err)
		assert.JSONEq(t, existingJSON, pod.Annotations[metadataenrichment.Annotation])
	})
}

func TestApplyAnnotationsToPod(t *testing.T) {
	t.Run("sets individual annotations for each merged source", func(t *testing.T) {
		attrs := newTestPodAttributes()
		attrs.namespaceAnnotations["ns-attr"] = "from-ns"
		attrs.rulesPropagate["rule-attr"] = "from-rule"
		attrs.workloadInfo[K8sWorkloadKindAttr] = "deployment"
		pod := &corev1.Pod{}

		err := attrs.ApplyAnnotationsToPod(pod)

		require.NoError(t, err)
		assert.Equal(t, "deployment", pod.Annotations[metadataenrichment.Prefix+K8sWorkloadKindAttr])
		assert.Equal(t, "from-ns", pod.Annotations[metadataenrichment.Prefix+"ns-attr"])
		assert.Equal(t, "from-rule", pod.Annotations[metadataenrichment.Prefix+"rule-attr"])
	})

	t.Run("namespaceAnnotations overrides workloadInfo in the JSON annotation", func(t *testing.T) {
		attrs := newTestPodAttributes()
		attrs.workloadInfo["shared.key"] = "from-workload"
		attrs.namespaceAnnotations["shared.key"] = "from-ns"
		pod := &corev1.Pod{}

		err := attrs.ApplyAnnotationsToPod(pod)

		require.NoError(t, err)
		var parsed map[string]string
		require.NoError(t, json.Unmarshal([]byte(pod.Annotations[metadataenrichment.Annotation]), &parsed))
		assert.Equal(t, "from-ns", parsed["shared.key"])
	})

	t.Run("namespaceAnnotations overrides rulesPropagate in the JSON annotation", func(t *testing.T) {
		attrs := newTestPodAttributes()
		attrs.namespaceAnnotations["shared.key"] = "from-ns"
		attrs.rulesPropagate["shared.key"] = "from-rules"
		pod := &corev1.Pod{}

		err := attrs.ApplyAnnotationsToPod(pod)

		require.NoError(t, err)
		var parsed map[string]string
		require.NoError(t, json.Unmarshal([]byte(pod.Annotations[metadataenrichment.Annotation]), &parsed))
		assert.Equal(t, "from-rules", parsed["shared.key"])
	})

	t.Run("does not overwrite existing individual annotations on pod", func(t *testing.T) {
		attrs := newTestPodAttributes()
		attrs.namespaceAnnotations["existing-key"] = "new-value"
		pod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					metadataenrichment.Prefix + "existing-key": "original-value",
				},
			},
		}

		err := attrs.ApplyAnnotationsToPod(pod)

		require.NoError(t, err)
		assert.Equal(t, "original-value", pod.Annotations[metadataenrichment.Prefix+"existing-key"])
	})
}
