package metadata

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

func TestFindRootOwnerOfPod(t *testing.T) {
	ctx := context.Background()
	resourceName := "test"
	namespaceName := "test"

	t.Run("should find the root owner of the pod", func(t *testing.T) {
		pod := corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: "apps/v1",
						Kind:       "Deployment",
						Name:       "test",
						Controller: ptr.To(true),
					},
				},
				Name:      resourceName,
				Namespace: namespaceName,
			},
		}

		deployment := appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: "apps/v1",
						Kind:       "DaemonSet",
						Name:       "test",
						Controller: ptr.To(true),
					},
				},
				Name:      resourceName,
				Namespace: namespaceName,
			},
		}

		daemonSet := appsv1.DaemonSet{
			TypeMeta: metav1.TypeMeta{
				Kind:       "DaemonSet",
				APIVersion: "apps/v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      resourceName,
				Namespace: namespaceName,
			},
		}

		namespace := corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: namespaceName,
			},
		}

		client := fake.NewClient(&pod, &deployment, &daemonSet, &namespace)

		workloadInfo, err := findRootOwnerOfPod(ctx, client, &pod, namespaceName)
		require.NoError(t, err)
		assert.Equal(t, resourceName, workloadInfo.Name)
		assert.Equal(t, "daemonset", workloadInfo.Kind)
	})

	t.Run("should return Pod if owner references are empty", func(t *testing.T) {
		pod := corev1.Pod{
			TypeMeta: metav1.TypeMeta{
				Kind: "Pod",
			},
			ObjectMeta: metav1.ObjectMeta{
				OwnerReferences: []metav1.OwnerReference{},
				Name:            resourceName,
			},
		}
		client := fake.NewClient(&pod)
		workloadInfo, err := findRootOwnerOfPod(ctx, client, &pod, namespaceName)
		require.NoError(t, err)
		assert.Equal(t, resourceName, workloadInfo.Name)
		assert.Equal(t, "pod", workloadInfo.Kind)
	})

	t.Run("should be pod if owner is not well known", func(t *testing.T) {
		pod := corev1.Pod{
			TypeMeta: metav1.TypeMeta{
				Kind: "Pod",
			},
			ObjectMeta: metav1.ObjectMeta{
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: "v1",
						Kind:       "Secret",
						Name:       "test",
						Controller: ptr.To(true),
					},
				},
				Name: resourceName,
			},
		}
		secret := corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      resourceName,
				Namespace: namespaceName,
			},
		}
		client := fake.NewClient(&pod, &secret)
		workloadInfo, err := findRootOwnerOfPod(ctx, client, &pod, namespaceName)
		require.NoError(t, err)
		assert.Equal(t, resourceName, workloadInfo.Name)
		assert.Equal(t, "pod", workloadInfo.Kind)
	})

	t.Run("should be pod if no controller is the owner", func(t *testing.T) {
		pod := corev1.Pod{
			TypeMeta: metav1.TypeMeta{
				Kind: "Pod",
			},
			ObjectMeta: metav1.ObjectMeta{
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion:         "some.unknown.kind.com/v1alpha1",
						Kind:               "SomeUnknownKind",
						Name:               "some-owner",
						Controller:         ptr.To(false),
						BlockOwnerDeletion: ptr.To(false),
					},
				},
				Name: resourceName,
			},
		}
		client := fake.NewClient(&pod)
		workloadInfo, err := findRootOwnerOfPod(ctx, client, &pod, namespaceName)
		require.NoError(t, err)
		assert.Equal(t, namespaceName, workloadInfo.Name)
		assert.Equal(t, "pod", workloadInfo.Kind)
	})
	t.Run("should find the root owner of the pod if the root owner is unknown", func(t *testing.T) {
		pod := corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: "apps/v1",
						Kind:       "Deployment",
						Name:       "test",
						Controller: ptr.To(true),
					},
				},
				Name:      resourceName,
				Namespace: namespaceName,
			},
		}

		deployment := appsv1.Deployment{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Deployment",
				APIVersion: "apps/v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: "v1",
						Kind:       "Secret",
						Name:       "test",
						Controller: ptr.To(true),
					},
				},
				Name:      resourceName,
				Namespace: namespaceName,
			},
		}

		secret := corev1.Secret{
			TypeMeta: metav1.TypeMeta{
				Kind: "Secret",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      resourceName,
				Namespace: namespaceName,
			},
		}

		namespace := corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: namespaceName,
			},
		}

		client := fake.NewClient(&pod, &deployment, &secret, &namespace)

		workloadInfo, err := findRootOwnerOfPod(ctx, client, &pod, namespaceName)
		require.NoError(t, err)
		assert.Equal(t, resourceName, workloadInfo.Name)
		assert.Equal(t, "deployment", workloadInfo.Kind)
	})
	t.Run("should not make an api-call if workload is not well known", func(t *testing.T) {
		pod := corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: "some.unknown.kind.com/v1alpha1",
						Kind:       "SomeUnknownKind",
						Name:       "some-owner",
						Controller: ptr.To(true),
					},
				},
				Name:      resourceName,
				Namespace: namespaceName,
			},
		}

		client := createFailK8sClient(t)

		workloadInfo, err := findRootOwnerOfPod(ctx, client, &pod, namespaceName)
		require.NoError(t, err)
		assert.Equal(t, resourceName, workloadInfo.Name)
	})
}

func createFailK8sClient(t *testing.T) client.Client {
	t.Helper()

	boomClient := fake.NewClientWithInterceptors(interceptor.Funcs{
		Get: func(ctx context.Context, client client.WithWatch, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
			return errors.New("BOOM")
		},
	})

	return boomClient
}
