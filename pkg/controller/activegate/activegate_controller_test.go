package activegate

import (
	"context"
	"fmt"
	"testing"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-activegate-operator/pkg/apis/dynatrace/v1alpha1"
	"github.com/Dynatrace/dynatrace-activegate-operator/pkg/controller/builder"
	_const "github.com/Dynatrace/dynatrace-activegate-operator/pkg/controller/const"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type failUpdatePodsService struct {
	mockIsLatestUpdateService
}

const errorUpdatingPods = "error updating pods"

func (updateServer *failUpdatePodsService) UpdatePods(*ReconcileActiveGate, *corev1.Pod, *dynatracev1alpha1.ActiveGate, *corev1.Secret) (*reconcile.Result, error) {
	result := builder.ReconcileAfterFiveMinutes()
	return &result, fmt.Errorf(errorUpdatingPods)
}

func TestUpdateInstanceStatus(t *testing.T) {
	r, instance, err := setupReconciler(t, &mockIsLatestUpdateService{})
	assert.NotNil(t, r)
	assert.NoError(t, err)

	pods, err := r.findPods(instance)
	assert.NotEmpty(t, pods)
	assert.NoError(t, err)

	for _, pod := range pods {
		r.updateInstanceStatus(&pod, instance, nil)
	}
	assert.Equal(t, mockActivegateVersion, instance.Status.Version)
}

func TestGetTokenSecret(t *testing.T) {
	r, instance, err := setupReconciler(t, &mockIsLatestUpdateService{})
	assert.NotNil(t, r)
	assert.NoError(t, err)

	t.Run("GetTokenSecret", func(t *testing.T) {
		secret, err := r.getTokenSecret(instance)
		assert.NoError(t, err)
		assert.NotNil(t, secret)
		assert.Equal(t, _const.ActivegateName, secret.Name)
	})
	t.Run("GetTokenSecret missing secret", func(t *testing.T) {
		secret, err := r.getTokenSecret(&dynatracev1alpha1.ActiveGate{})
		assert.NoError(t, err)
		assert.Nil(t, secret)
	})
}

func TestReconcile(t *testing.T) {
	t.Run("Reconile instance not found", func(t *testing.T) {
		r, instance, err := setupReconciler(t, &mockIsLatestUpdateService{})
		assert.NotNil(t, r)
		assert.NotNil(t, instance)
		assert.NoError(t, err)

		result, err := r.Reconcile(reconcile.Request{})
		assert.NotNil(t, result)
		assert.NoError(t, err)
	})
	t.Run("Reconcile create pod", func(t *testing.T) {
		r, instance, err := setupReconciler(t, &mockIsLatestUpdateService{})
		assert.NotNil(t, r)
		assert.NotNil(t, instance)
		assert.NoError(t, err)

		result, err := r.Reconcile(reconcile.Request{
			NamespacedName: types.NamespacedName{
				Namespace: instance.Namespace,
				Name:      instance.Name,
			},
		})
		assert.NotNil(t, result)
		assert.NoError(t, err)
		assert.Equal(t, result, builder.ReconcileAfterFiveMinutes())

		secret, err := r.getTokenSecret(instance)
		assert.NoError(t, err)

		pod := r.newPodForCR(instance, secret)
		assert.NotNil(t, pod)

		found := &corev1.Pod{}
		err = r.client.Get(context.TODO(), types.NamespacedName{Name: pod.Name, Namespace: pod.Namespace}, found)
		assert.NotNil(t, found)
		assert.NoError(t, err)

		assert.Equal(t, pod.Name, found.Name)
		assert.Equal(t, pod.Namespace, found.Namespace)
	})
	t.Run("Reconcile missing secret", func(t *testing.T) {
		r, instance, err := setupReconciler(t, &mockIsLatestUpdateService{})
		assert.NotNil(t, r)
		assert.NotNil(t, instance)
		assert.NoError(t, err)

		// First run: Create pod
		result, err := r.Reconcile(reconcile.Request{
			NamespacedName: types.NamespacedName{
				Namespace: instance.Namespace,
				Name:      instance.Name,
			},
		})
		assert.NotNil(t, result)
		assert.NoError(t, err)

		secret, err := r.getTokenSecret(instance)
		assert.NoError(t, err)
		assert.NotNil(t, secret)

		err = r.client.Delete(context.TODO(), secret)
		assert.NoError(t, err)

		// Second run: Expected to reconcile immediately
		result, err = r.Reconcile(reconcile.Request{
			NamespacedName: types.NamespacedName{
				Namespace: instance.Namespace,
				Name:      instance.Name,
			},
		})
		assert.NotNil(t, result)
		assert.NoError(t, err)
		assert.Equal(t, result, builder.ReconcileAfter(0))
	})
	t.Run("Reconcile error updating pods", func(t *testing.T) {
		r, instance, err := setupReconciler(t, &failUpdatePodsService{})
		assert.NotNil(t, r)
		assert.NotNil(t, instance)
		assert.NoError(t, err)

		// First run: Create pod
		result, err := r.Reconcile(reconcile.Request{
			NamespacedName: types.NamespacedName{
				Namespace: instance.Namespace,
				Name:      instance.Name,
			},
		})
		assert.NotNil(t, result)
		assert.NoError(t, err)

		// Second run: Expected to reconcile after five minutes and return error returned by UpdatePods
		result, err = r.Reconcile(reconcile.Request{
			NamespacedName: types.NamespacedName{
				Namespace: instance.Namespace,
				Name:      instance.Name,
			},
		})
		assert.NotNil(t, result)
		assert.Error(t, err)
		assert.Equal(t, errorUpdatingPods, err.Error())
		assert.Equal(t, result, builder.ReconcileAfterFiveMinutes())
	})
	t.Run("Reconcile", func(t *testing.T) {
		r, instance, err := setupReconciler(t, &mockIsLatestUpdateService{})
		assert.NotNil(t, r)
		assert.NotNil(t, instance)
		assert.NoError(t, err)

		// First run: Create pod
		result, err := r.Reconcile(reconcile.Request{
			NamespacedName: types.NamespacedName{
				Namespace: instance.Namespace,
				Name:      instance.Name,
			},
		})
		assert.NotNil(t, result)
		assert.NoError(t, err)

		// Second run: Expected to have nothing to do
		result, err = r.Reconcile(reconcile.Request{
			NamespacedName: types.NamespacedName{
				Namespace: instance.Namespace,
				Name:      instance.Name,
			},
		})
		assert.NotNil(t, result)
		assert.NoError(t, err)
		assert.Equal(t, result, builder.ReconcileAfterFiveMinutes())
	})
}
