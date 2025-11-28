package attributes

import (
	"testing"

	maputils "github.com/Dynatrace/dynatrace-operator/pkg/util/map"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/workload"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestWorkloadAnnotations(t *testing.T) {
	workloadInfoName := "workload-name"
	workloadInfoKind := "workload-kind"

	t.Run("should add annotation to nil map", func(t *testing.T) {
		request := createTestMutationRequest(nil, nil)

		require.Equal(t, "not-found", maputils.GetField(request.Pod.Annotations, AnnotationWorkloadName, "not-found"))
		setWorkloadAnnotations(request.Pod, &workload.Info{Name: workloadInfoName, Kind: workloadInfoKind})
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

		setWorkloadAnnotations(request.Pod, workload.NewInfo(objectMeta))
		assert.Contains(t, request.Pod.Annotations, AnnotationWorkloadKind)
		assert.Equal(t, "superworkload", request.Pod.Annotations[AnnotationWorkloadKind])
	})
}

/*
func TestGetWorkloadInfo(t *testing.T) {
	t.Run("should return not found for missing owner references", func(t *testing.T) {
		request := createTestMutationRequest(nil, nil)
		attrs := podattr.Attributes{}
		info, err := GetWorkloadInfoAttributes(attrs, t.Context(), request.BaseRequest, nil)
		require.NoError(t, err)
		assert.Equal(t, workload.Info{Name: workload.NotFound, Kind: workload.NotFound}, info.WorkloadInfo)
	})

	t.Run("should return workload info from owner references", func(t *testing.T) {
		ownerName := "owner-name"
		ownerKind := "owner-kind"
		ownerRefs := []metav1.OwnerReference{
			{
				Kind: ownerKind,
				Name: ownerName,
			},
		}
		request := createTestMutationRequest(nil, ownerRefs)

		info, err := GetWorkloadInfoAttributes(workload.Attributes{}, nil, request, nil)
		require.NoError(t, err)
		assert.Equal(t, workload.Info{Name: ownerName, Kind: ownerKind}, info.WorkloadInfo)
	}
}*/
