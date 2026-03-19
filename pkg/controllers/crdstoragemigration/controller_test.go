package crdstoragemigration

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

func TestInitReconcile(t *testing.T) {
	setupOneTimeClient := func(deploy *appsv1.Deployment) (context.Context, client.Client) {
		ctx, cancel := context.WithCancel(t.Context())

		deploy.Name = webhook.DeploymentName
		deploy.Namespace = testNamespace

		fakeClient := fake.NewClientWithInterceptors(interceptor.Funcs{
			Get: func(_ context.Context, _ client.WithWatch, _ client.ObjectKey, obj client.Object, _ ...client.GetOption) error {
				deploy.DeepCopyInto(obj.(*appsv1.Deployment))
				cancel()

				return nil
			},
		})

		return ctx, fakeClient
	}

	t.Cleanup(func() { run = Run })

	t.Run("deployment not found", func(t *testing.T) {
		run = nil
		err := InitReconcile(t.Context(), fake.NewClient(), testNamespace)
		assert.NoError(t, err)
	})

	t.Run("deployment not ready", func(t *testing.T) {
		ctx, clt := setupOneTimeClient(&appsv1.Deployment{
			Spec: appsv1.DeploymentSpec{Replicas: ptr.To(int32(1))},
		})

		run = nil
		err := InitReconcile(ctx, clt, testNamespace)
		assert.ErrorIs(t, err, context.Canceled)
	})

	t.Run("run error", func(t *testing.T) {
		ctx, clt := setupOneTimeClient(&appsv1.Deployment{
			Spec:   appsv1.DeploymentSpec{Replicas: ptr.To(int32(3))},
			Status: appsv1.DeploymentStatus{ReadyReplicas: 3},
		})

		called := false
		run = func(context.Context, client.Client, string) error {
			called = true

			return errors.New("retry")
		}

		err := InitReconcile(ctx, clt, testNamespace)
		assert.ErrorIs(t, err, context.Canceled) //nolint:testifylint
		assert.True(t, called)
	})

	t.Run("run ok", func(t *testing.T) {
		ctx, clt := setupOneTimeClient(&appsv1.Deployment{
			Spec:   appsv1.DeploymentSpec{Replicas: ptr.To(int32(3))},
			Status: appsv1.DeploymentStatus{ReadyReplicas: 3},
		})

		run = func(context.Context, client.Client, string) error {
			return nil
		}

		err := InitReconcile(ctx, clt, testNamespace)
		assert.NoError(t, err)
	})
}

func Test_isDeploymentReady(t *testing.T) {
	tests := []struct {
		name     string
		deploy   *appsv1.Deployment
		expected bool
	}{
		{
			name: "ready: generation synced and all replicas ready",
			deploy: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{Generation: 2},
				Spec:       appsv1.DeploymentSpec{Replicas: ptr.To(int32(3))},
				Status:     appsv1.DeploymentStatus{ObservedGeneration: 2, ReadyReplicas: 3},
			},
			expected: true,
		},
		{
			name: "not ready: generation synced but not all replicas ready",
			deploy: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{Generation: 2},
				Spec:       appsv1.DeploymentSpec{Replicas: ptr.To(int32(3))},
				Status:     appsv1.DeploymentStatus{ObservedGeneration: 2, ReadyReplicas: 2},
			},
			expected: false,
		},
		{
			name: "not ready: generation not yet observed by controller",
			deploy: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{Generation: 3},
				Spec:       appsv1.DeploymentSpec{Replicas: ptr.To(int32(3))},
				Status:     appsv1.DeploymentStatus{ObservedGeneration: 2, ReadyReplicas: 3},
			},
			expected: false,
		},
		{
			name: "not ready: scaled down deployment",
			deploy: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{Generation: 1},
				Spec:       appsv1.DeploymentSpec{Replicas: ptr.To(int32(0))},
				Status:     appsv1.DeploymentStatus{ObservedGeneration: 1, ReadyReplicas: 0},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, isDeploymentReady(tt.deploy))
		})
	}
}
