package dtcsi

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	"github.com/Dynatrace/dynatrace-operator/controllers/kubeobjects"
	"github.com/Dynatrace/dynatrace-operator/logger"
	"github.com/Dynatrace/dynatrace-operator/scheme"
	"github.com/Dynatrace/dynatrace-operator/scheme/fake"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	testNamespace       = "test-namespace"
	testDynakube        = "test-dynakube"
	testOperatorPodName = "test-operator-pod"
	testOperatorImage   = "test-operator-image"
)

func TestReconcile_NoOperatorPod(t *testing.T) {
	log := logger.NewDTLogger()
	fakeClient := fake.NewClient()
	rec := NewReconciler(fakeClient, scheme.Scheme, log, nil, testOperatorPodName, testNamespace)

	result, err := rec.Reconcile()
	require.Error(t, err)
	require.False(t, result)
}

func TestReconcile_NoOperatorImage(t *testing.T) {
	log := logger.NewDTLogger()
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testOperatorPodName,
			Namespace: testNamespace,
		},
	}
	fakeClient := fake.NewClient(pod)
	rec := NewReconciler(fakeClient, scheme.Scheme, log, nil, testOperatorPodName, testNamespace)

	result, err := rec.Reconcile()
	require.Error(t, err)
	require.False(t, result)
}

func TestReconcile_CreateDaemonSet(t *testing.T) {
	log := logger.NewDTLogger()
	fakeClient := prepareFakeClient()
	dk := prepareDynakube(testDynakube)
	rec := NewReconciler(fakeClient, scheme.Scheme, log, dk, testOperatorPodName, testNamespace)

	result, err := rec.Reconcile()
	require.NoError(t, err)
	assert.True(t, result)

	createdDaemonSet := &appsv1.DaemonSet{}
	err = fakeClient.Get(context.TODO(), client.ObjectKey{
		Namespace: testNamespace,
		Name:      DaemonSetName,
	}, createdDaemonSet)
	require.NoError(t, err)

	assert.NotNil(t, createdDaemonSet.Annotations)
	assert.Contains(t, createdDaemonSet.Annotations, kubeobjects.AnnotationHash)
	assert.Len(t, createdDaemonSet.Spec.Template.Spec.Containers, 3)
	assert.Len(t, createdDaemonSet.Spec.Template.Spec.Volumes, 5)
}

func TestReconcile_UpdateDaemonSet(t *testing.T) {
	log := logger.NewDTLogger()
	ds := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      DaemonSetName,
			Namespace: testNamespace,
			Annotations: map[string]string{
				kubeobjects.AnnotationHash: "old",
			},
		},
	}
	fakeClient := prepareFakeClient(ds)

	dk := prepareDynakube(testDynakube)
	rec := NewReconciler(fakeClient, scheme.Scheme, log, dk, testOperatorPodName, testNamespace)
	result, err := rec.Reconcile()
	require.NoError(t, err)
	assert.True(t, result)

	updatedDaemonSet := &appsv1.DaemonSet{}
	err = fakeClient.Get(context.TODO(), client.ObjectKey{
		Namespace: testNamespace,
		Name:      DaemonSetName,
	}, updatedDaemonSet)
	require.NoError(t, err)

	assert.NotNil(t, updatedDaemonSet.Annotations)
	assert.Contains(t, updatedDaemonSet.Annotations, kubeobjects.AnnotationHash)
	assert.NotEqual(t, "old", updatedDaemonSet.Annotations[kubeobjects.AnnotationHash])
}

func prepareFakeClient(objs ...client.Object) client.Client {
	objs = append(objs,
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testOperatorPodName,
				Namespace: testNamespace,
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Image: testOperatorImage,
					},
				},
			},
		})
	return fake.NewClient(objs...)
}

func prepareDynakube(name string) *v1alpha1.DynaKube {
	return &v1alpha1.DynaKube{
		TypeMeta: metav1.TypeMeta{
			Kind:       "DynaKube",
			APIVersion: "dynatrace.com/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: testNamespace,
			UID:       types.UID(name),
		},
		Spec: v1alpha1.DynaKubeSpec{
			CodeModules: v1alpha1.CodeModulesSpec{
				ServiceAccountNameCSIDriver: "test",
			},
		},
	}
}
