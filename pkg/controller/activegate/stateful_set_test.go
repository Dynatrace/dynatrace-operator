package activegate

import (
	"context"
	"testing"

	_const "github.com/Dynatrace/dynatrace-operator/pkg/controller/const"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestCreateDesiredStatefulSet(t *testing.T) {
	t.Run("CreateDesiredStatefulSet", func(t *testing.T) {
		r, instance, err := setupReconciler(t, &mockIsLatestUpdateService{})
		assert.NoError(t, err)

		desiredStatefulSet, err := r.createDesiredStatefulSet(instance)
		assert.NoError(t, err)
		assert.NotNil(t, desiredStatefulSet)
	})
	t.Run("CreateDesiredStatefulSet no kube-system", func(t *testing.T) {
		r, instance, err := setupReconciler(t, &mockIsLatestUpdateService{})
		assert.NoError(t, err)

		kubeSystemNamespace := &corev1.Namespace{}
		err = r.client.Get(context.TODO(), client.ObjectKey{Name: _const.KubeSystemNamespace}, kubeSystemNamespace)
		assert.NoError(t, err)

		err = r.client.Delete(context.TODO(), kubeSystemNamespace)
		assert.NoError(t, err)

		desiredStatefulSet, err := r.createDesiredStatefulSet(instance)
		assert.EqualError(t, err, "namespaces \"kube-system\" not found")
		assert.Nil(t, desiredStatefulSet)
	})
}
