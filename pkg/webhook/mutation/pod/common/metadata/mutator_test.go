package metadata

import (
	"testing"

	maputils "github.com/Dynatrace/dynatrace-operator/pkg/util/map"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestSetInjectedAnnotation(t *testing.T) {
	t.Run("should add annotation to nil map", func(t *testing.T) {
		request := createTestMutationRequest(nil, nil)

		require.False(t, IsInjected(request.BaseRequest))
		SetInjectedAnnotation(request.Pod)
		require.Len(t, request.Pod.Annotations, 1)
		require.True(t, IsInjected(request.BaseRequest))
	})
}

func TestWorkloadAnnotations(t *testing.T) {
	workloadInfoName := "workload-name"
	workloadInfoKind := "workload-kind"

	t.Run("should add annotation to nil map", func(t *testing.T) {
		request := createTestMutationRequest(nil, nil)

		require.Equal(t, "not-found", maputils.GetField(request.Pod.Annotations, AnnotationWorkloadName, "not-found"))
		SetWorkloadAnnotations(request.Pod, &WorkloadInfo{Name: workloadInfoName, Kind: workloadInfoKind})
		require.Len(t, request.Pod.Annotations, 2)
		assert.Equal(t, workloadInfoName, maputils.GetField(request.Pod.Annotations, AnnotationWorkloadName, "not-found"))
		assert.Equal(t, workloadInfoKind, maputils.GetField(request.Pod.Annotations, AnnotationWorkloadKind, "not-found"))
	})
	t.Run("should lower case kind annotation", func(t *testing.T) {
		request := createTestMutationRequest(nil, nil)
		objectMeta := &metav1.PartialObjectMetadata{
			ObjectMeta: metav1.ObjectMeta{Name: workloadInfoName},
			TypeMeta:   metav1.TypeMeta{Kind: "SuperWorkload"},
		}

		SetWorkloadAnnotations(request.Pod, newWorkloadInfo(objectMeta))
		assert.Contains(t, request.Pod.Annotations, AnnotationWorkloadKind)
		assert.Equal(t, "superworkload", request.Pod.Annotations[AnnotationWorkloadKind])
	})
}
