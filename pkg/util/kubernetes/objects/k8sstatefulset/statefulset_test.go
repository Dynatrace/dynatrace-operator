package k8sstatefulset

import (
	"context"
	"errors"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

const ns = "dynatrace"

func TestResolveReplicas(t *testing.T) {
	const name = "test-statefulset"

	objectKey := client.ObjectKey{Name: name, Namespace: ns}
	testErr := errors.New("kube api failure")

	tests := []struct {
		name            string
		reader          client.Reader
		defaultReplicas *int32
		expected        int32
		expectedErr     error
	}{
		{
			name:            "returns provided default replicas",
			reader:          fake.NewClient(&appsv1.StatefulSet{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns}, Spec: appsv1.StatefulSetSpec{Replicas: ptr.To(int32(7))}}),
			defaultReplicas: ptr.To(int32(3)),
			expected:        int32(3),
		},
		{
			name:     "returns statefulset replicas when found",
			reader:   fake.NewClient(&appsv1.StatefulSet{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns}, Spec: appsv1.StatefulSetSpec{Replicas: ptr.To(int32(5))}}),
			expected: int32(5),
		},
		{
			name:     "returns one when statefulset has nil replicas",
			reader:   fake.NewClient(&appsv1.StatefulSet{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns}}),
			expected: int32(1),
		},
		{
			name:     "returns one when statefulset is not found",
			reader:   fake.NewClient(),
			expected: int32(1),
		},
		{
			name: "returns error on kube reader failure",
			reader: fake.NewClientWithInterceptors(interceptor.Funcs{
				Get: func(context.Context, client.WithWatch, client.ObjectKey, client.Object, ...client.GetOption) error {
					return testErr
				},
			}),
			expectedErr: testErr,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			replicas, err := ResolveReplicas(t.Context(), tc.reader, objectKey, tc.defaultReplicas)

			if tc.expectedErr != nil {
				require.ErrorIs(t, err, tc.expectedErr)
				assert.Equal(t, int32(0), replicas)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, tc.expected, replicas)
		})
	}
}

func TestResolveAndSetReplicas(t *testing.T) {
	const name = "test-statefulset"

	testErr := errors.New("kube api failure")

	tests := []struct {
		name            string
		reader          client.Reader
		defaultReplicas *int32
		expected        *int32
		expectedErr     error
	}{
		{
			name: "sets replicas from resolved statefulset",
			reader: fake.NewClient(&appsv1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns},
				Spec:       appsv1.StatefulSetSpec{Replicas: ptr.To(int32(6))},
			}),
			expected: ptr.To(int32(6)),
		},
		{
			name:            "sets replicas from provided default",
			reader:          fake.NewClient(),
			defaultReplicas: ptr.To(int32(4)),
			expected:        ptr.To(int32(4)),
		},
		{
			name: "returns error and does not set replicas when reader fails",
			reader: fake.NewClientWithInterceptors(interceptor.Funcs{
				Get: func(context.Context, client.WithWatch, client.ObjectKey, client.Object, ...client.GetOption) error {
					return testErr
				},
			}),
			expectedErr: testErr,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			statefulSet := &appsv1.StatefulSet{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns}}

			err := ResolveAndSetReplicas(t.Context(), tc.reader, statefulSet, tc.defaultReplicas)
			if tc.expectedErr != nil {
				require.ErrorIs(t, err, tc.expectedErr)
				assert.Nil(t, statefulSet.Spec.Replicas)

				return
			}

			require.NoError(t, err)
			require.NotNil(t, statefulSet.Spec.Replicas)
			assert.Equal(t, *tc.expected, *statefulSet.Spec.Replicas)
		})
	}
}
