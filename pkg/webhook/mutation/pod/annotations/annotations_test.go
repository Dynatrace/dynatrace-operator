package annotations

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func newTestMutationRequest(t *testing.T) *mutator.MutationRequest {
	pod := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "pod", Namespace: "ns"}}
	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "ns"}}
	dk := &dynakube.DynaKube{ObjectMeta: metav1.ObjectMeta{Name: "dk", Namespace: "ns"}}

	return mutator.NewMutationRequest(t.Context(), *ns, nil, pod, *dk)
}

func TestSetDynatraceInjectedAnnotation_InitializesMapAndSetsFlags(t *testing.T) {
	req := newTestMutationRequest(t)
	// ensure nil map scenario
	req.Pod.Annotations = nil

	SetDynatraceInjectedAnnotation(req)

	require.NotNil(t, req.Pod.Annotations)
	assert.Equal(t, "true", req.Pod.Annotations[mutator.AnnotationDynatraceInjected])
	_, hasReason := req.Pod.Annotations[mutator.AnnotationDynatraceReason]
	assert.False(t, hasReason, "reason annotation should be removed when setting injected=true")
}

func TestSetDynatraceInjectedAnnotation_RemovesReasonIfPresent(t *testing.T) {
	req := newTestMutationRequest(t)
	req.Pod.Annotations = map[string]string{
		mutator.AnnotationDynatraceReason:   "some-reason",
		"other":                             "value",
		mutator.AnnotationDynatraceInjected: "false",
	}

	SetDynatraceInjectedAnnotation(req)

	assert.Equal(t, "true", req.Pod.Annotations[mutator.AnnotationDynatraceInjected])
	assert.Equal(t, "value", req.Pod.Annotations["other"], "unrelated annotation must be preserved")
	_, hasReason := req.Pod.Annotations[mutator.AnnotationDynatraceReason]
	assert.False(t, hasReason, "reason annotation should be deleted")
}

func TestSetNotInjectedAnnotations_InitializesMapAndSetsReason(t *testing.T) {
	req := newTestMutationRequest(t)
	req.Pod.Annotations = nil

	SetNotInjectedAnnotations(req, "missing-secret")

	require.NotNil(t, req.Pod.Annotations)
	assert.Equal(t, "false", req.Pod.Annotations[mutator.AnnotationDynatraceInjected])
	assert.Equal(t, "missing-secret", req.Pod.Annotations[mutator.AnnotationDynatraceReason])
}
