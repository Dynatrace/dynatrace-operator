package k8sdeployment

import (
	"context"
	"errors"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

const ns = "dynatrace"

var deploymentLog = logd.Get().WithName("test-deployment")

func createTestDeploymentWithMatchLabels(name, namespace string, annotations, matchLabels map[string]string) appsv1.Deployment {
	return appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   namespace,
			Annotations: annotations,
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: matchLabels,
			},
		},
	}
}

// GetDeployment returns the Deployment object who is the owner of this pod.
func TestGetDeployment(t *testing.T) {
	trueVar := true

	fakeClient := fake.NewClient(
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "mypod",
				Namespace: ns,
				OwnerReferences: []metav1.OwnerReference{
					{Kind: "ReplicaSet", Name: "myreplicaset", Controller: &trueVar},
				},
			},
		},
		&appsv1.ReplicaSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "myreplicaset",
				Namespace: ns,
				OwnerReferences: []metav1.OwnerReference{
					{Kind: "Deployment", Name: "mydeployment", Controller: &trueVar},
				},
			},
		},
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "mydeployment",
				Namespace: ns,
			},
		})

	deploy, err := GetDeployment(fakeClient, "mypod", "dynatrace")
	require.NoError(t, err)
	assert.Equal(t, "mydeployment", deploy.Name)
	assert.Equal(t, "dynatrace", deploy.Namespace)
}

func TestResolveReplicas(t *testing.T) {
	const name = "test-deployment"

	ctx := context.Background()
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
			reader:          fake.NewClient(&appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns}, Spec: appsv1.DeploymentSpec{Replicas: ptr.To(int32(7))}}),
			defaultReplicas: ptr.To(int32(3)),
			expected:        int32(3),
		},
		{
			name:     "returns deployment replicas when found",
			reader:   fake.NewClient(&appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns}, Spec: appsv1.DeploymentSpec{Replicas: ptr.To(int32(5))}}),
			expected: int32(5),
		},
		{
			name:     "returns one when deployment has nil replicas",
			reader:   fake.NewClient(&appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns}}),
			expected: int32(1),
		},
		{
			name:     "returns one when deployment is not found",
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
			replicas, err := ResolveReplicas(ctx, tc.reader, objectKey, tc.defaultReplicas)

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
	const name = "test-deployment"

	ctx := context.Background()
	testErr := errors.New("kube api failure")

	tests := []struct {
		name            string
		reader          client.Reader
		defaultReplicas *int32
		expected        *int32
		expectedErr     error
	}{
		{
			name: "sets replicas from resolved deployment",
			reader: fake.NewClient(&appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns},
				Spec:       appsv1.DeploymentSpec{Replicas: ptr.To(int32(6))},
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
			deployment := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns}}

			err := ResolveAndSetReplicas(ctx, tc.reader, deployment, tc.defaultReplicas)
			if tc.expectedErr != nil {
				require.ErrorIs(t, err, tc.expectedErr)
				assert.Nil(t, deployment.Spec.Replicas)

				return
			}

			require.NoError(t, err)
			require.NotNil(t, deployment.Spec.Replicas)
			assert.Equal(t, *tc.expected, *deployment.Spec.Replicas)
		})
	}
}
