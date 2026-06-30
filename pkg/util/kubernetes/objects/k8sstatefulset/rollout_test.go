package k8sstatefulset

import (
	"testing"

	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestIsRolloutComplete(t *testing.T) {
	tests := []struct {
		name        string
		statefulSet *appsv1.StatefulSet
		expected    bool
	}{
		{
			name:     "returns false for nil statefulset",
			expected: false,
		},
		{
			name: "returns false when generation is not observed yet",
			statefulSet: &appsv1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{Generation: 2},
				Spec:       appsv1.StatefulSetSpec{Replicas: new(int32(1))},
				Status:     appsv1.StatefulSetStatus{ObservedGeneration: 1, ReadyReplicas: 1},
			},
			expected: false,
		},
		{
			name: "returns false when ready replicas are below desired",
			statefulSet: &appsv1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{Generation: 2},
				Spec:       appsv1.StatefulSetSpec{Replicas: new(int32(2))},
				Status:     appsv1.StatefulSetStatus{ObservedGeneration: 2, ReadyReplicas: 1},
			},
			expected: false,
		},
		{
			name: "returns true when generation observed and all replicas ready",
			statefulSet: &appsv1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{Generation: 2},
				Spec:       appsv1.StatefulSetSpec{Replicas: new(int32(2))},
				Status:     appsv1.StatefulSetStatus{ObservedGeneration: 2, ReadyReplicas: 2},
			},
			expected: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.expected, IsRolloutComplete(test.statefulSet))
		})
	}
}
