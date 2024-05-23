package metadata

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/address"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
						Controller: address.Of(true),
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
						Controller: address.Of(true),
					},
				},
				Name:      resourceName,
				Namespace: namespaceName,
			},
		}

		daemonSet := appsv1.DaemonSet{
			TypeMeta: metav1.TypeMeta{
				Kind: "DaemonSet",
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
		assert.Equal(t, resourceName, workloadInfo.name)
		assert.Equal(t, "DaemonSet", workloadInfo.kind)
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
		assert.Equal(t, resourceName, workloadInfo.name)
		assert.Equal(t, "Pod", workloadInfo.kind)
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
						Controller: address.Of(true),
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
		client := fake.NewClient(&pod, &namespace)
		workloadInfo, err := findRootOwnerOfPod(ctx, client, &pod, namespaceName)
		require.NoError(t, err)
		assert.Equal(t, resourceName, workloadInfo.name)
		assert.Equal(t, "Pod", workloadInfo.kind)
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
						Controller:         address.Of(false),
						BlockOwnerDeletion: address.Of(false),
					},
				},
				Name: resourceName,
			},
		}
		client := fake.NewClient(&pod)
		workloadInfo, err := findRootOwnerOfPod(ctx, client, &pod, namespaceName)
		require.NoError(t, err)
		assert.Equal(t, namespaceName, workloadInfo.name)
		assert.Equal(t, "Pod", workloadInfo.kind)
	})
	t.Run("should find the root owner of the pod if the root owner is unknown", func(t *testing.T) {
		pod := corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: "apps/v1",
						Kind:       "Deployment",
						Name:       "test",
						Controller: address.Of(true),
					},
				},
				Name:      resourceName,
				Namespace: namespaceName,
			},
		}

		deployment := appsv1.Deployment{
			TypeMeta: metav1.TypeMeta{
				Kind: "Deployment",
			},
			ObjectMeta: metav1.ObjectMeta{
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: "v1",
						Kind:       "Secret",
						Name:       "test",
						Controller: address.Of(true),
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
		assert.Equal(t, resourceName, workloadInfo.name)
		assert.Equal(t, "Deployment", workloadInfo.kind)
	})
}

func createTestWorkloadInfo() *workloadInfo {
	return &workloadInfo{
		kind: "test",
		name: "test",
	}
}
