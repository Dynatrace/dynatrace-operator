package attributes

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

func TestGetWorkloadInfoAttributes(t *testing.T) {
	t.Run("sets workload kind and name from pod with no owner (pod is its own root owner)", func(t *testing.T) {
		ctx := t.Context()
		attrs := newTestPodAttributes()
		pod := corev1.Pod{
			TypeMeta:   metav1.TypeMeta{Kind: "Pod", APIVersion: "v1"},
			ObjectMeta: metav1.ObjectMeta{Name: "my-pod", Namespace: "my-ns"},
		}
		request := dtwebhook.BaseRequest{
			Pod:       &pod,
			Namespace: corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "my-ns"}},
		}

		err := attrs.readWorkloadInfoAttributes(ctx, request, fake.NewClient())

		require.NoError(t, err)
		assert.Equal(t, "pod", attrs.workloadInfo[K8sWorkloadKindAttr])
		assert.Equal(t, "my-pod", attrs.workloadInfo[K8sWorkloadNameAttr])
	})

	t.Run("propagates error when owner lookup fails", func(t *testing.T) {
		ctx := t.Context()
		attrs := newTestPodAttributes()
		pod := corev1.Pod{
			TypeMeta: metav1.TypeMeta{Kind: "Pod", APIVersion: "v1"},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "my-pod",
				Namespace: "my-ns",
				OwnerReferences: []metav1.OwnerReference{
					{APIVersion: "apps/v1", Kind: "Deployment", Name: "my-deploy", Controller: boolPtr(true)},
				},
			},
		}
		request := dtwebhook.BaseRequest{
			Pod:       &pod,
			Namespace: corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "my-ns"}},
		}
		failClient := fake.NewClientWithInterceptors(interceptor.Funcs{
			Get: func(ctx context.Context, c client.WithWatch, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
				return errors.New("boom")
			},
		})

		err := attrs.readWorkloadInfoAttributes(ctx, request, failClient)

		assert.Error(t, err)
	})
}

func boolPtr(b bool) *bool { return &b }
