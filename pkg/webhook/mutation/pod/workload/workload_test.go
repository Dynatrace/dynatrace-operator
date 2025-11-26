package workload

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	maputils "github.com/Dynatrace/dynatrace-operator/pkg/util/map"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator"
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

var testLogger = logd.Get()

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

		request := mutator.BaseRequest{Pod: &pod, Namespace: namespace}

		client := fake.NewClient(&pod, &deployment, &daemonSet, &namespace)

		workloadInfo, err := FindRootOwnerOfPod(ctx, client, request, testLogger)
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

		namespace := corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: namespaceName,
			},
		}

		request := mutator.BaseRequest{Pod: &pod, Namespace: namespace}

		client := fake.NewClient(&pod)
		workloadInfo, err := FindRootOwnerOfPod(ctx, client, request, testLogger)
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

		namespace := corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: namespaceName,
			},
		}

		request := mutator.BaseRequest{Pod: &pod, Namespace: namespace}

		client := fake.NewClient(&pod, &secret)
		workloadInfo, err := FindRootOwnerOfPod(ctx, client, request, testLogger)
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

		namespace := corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: namespaceName,
			},
		}

		request := mutator.BaseRequest{Pod: &pod, Namespace: namespace}

		client := fake.NewClient(&pod)
		workloadInfo, err := FindRootOwnerOfPod(ctx, client, request, testLogger)
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

		request := mutator.BaseRequest{Pod: &pod, Namespace: namespace}

		client := fake.NewClient(&pod, &deployment, &secret, &namespace)

		workloadInfo, err := FindRootOwnerOfPod(ctx, client, request, testLogger)
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

		namespace := corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: namespaceName,
			},
		}

		request := mutator.BaseRequest{Pod: &pod, Namespace: namespace}

		client := createFailK8sClient(t)

		workloadInfo, err := FindRootOwnerOfPod(ctx, client, request, testLogger)
		require.NoError(t, err)
		assert.Equal(t, resourceName, workloadInfo.Name)
	})

	t.Run("should add annotation if owner lookup failed", func(t *testing.T) {
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

		namespace := corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: namespaceName,
			},
		}

		request := mutator.BaseRequest{Pod: &pod, Namespace: namespace}

		client := createFailK8sClient(t)
		_, err := FindRootOwnerOfPod(ctx, client, request, testLogger)
		require.Error(t, err)
	})
}

func TestWorkloadAnnotations(t *testing.T) {
	workloadInfoName := "workload-name"
	workloadInfoKind := "workload-kind"

	t.Run("should add annotation to nil map", func(t *testing.T) {
		pod := corev1.Pod{}

		require.Equal(t, "not-found", maputils.GetField(pod.Annotations, AnnotationWorkloadName, "not-found"))
		SetWorkloadAnnotations(&pod, &Info{Name: workloadInfoName, Kind: workloadInfoKind})
		require.Len(t, pod.Annotations, 2)
		assert.Equal(t, workloadInfoName, maputils.GetField(pod.Annotations, AnnotationWorkloadName, "not-found"))
		assert.Equal(t, workloadInfoKind, maputils.GetField(pod.Annotations, AnnotationWorkloadKind, "not-found"))
	})
	t.Run("should lower case kind annotation", func(t *testing.T) {
		pod := corev1.Pod{}
		objectMeta := &metav1.PartialObjectMetadata{
			ObjectMeta: metav1.ObjectMeta{Name: workloadInfoName},
			TypeMeta:   metav1.TypeMeta{Kind: "SuperWorkload"},
		}

		SetWorkloadAnnotations(&pod, NewInfo(objectMeta))
		assert.Contains(t, pod.Annotations, AnnotationWorkloadKind)
		assert.Equal(t, "superworkload", pod.Annotations[AnnotationWorkloadKind])
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
