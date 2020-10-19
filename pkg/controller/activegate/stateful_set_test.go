package activegate

import (
	"context"
	"fmt"
	"testing"

	_const "github.com/Dynatrace/dynatrace-operator/pkg/controller/const"
	"github.com/Dynatrace/dynatrace-operator/pkg/dtclient"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestCreateDesiredStatefulSet(t *testing.T) {
	t.Run("CreateDesiredStatefulSet", func(t *testing.T) {
		r, instance, err := setupReconciler(t, &mockIsLatestUpdateService{})
		assert.NoError(t, err)

		dtc, err := createFakeDTClient(nil, nil, nil)
		assert.NoError(t, err)

		desiredStatefulSet, err := r.createDesiredStatefulSet(instance, dtc)
		assert.NoError(t, err)
		assert.NotNil(t, desiredStatefulSet)
	})
	t.Run("CreateDesiredStatefulSet error getting tenant info", func(t *testing.T) {
		r, instance, err := setupReconciler(t, &mockIsLatestUpdateService{})
		assert.NoError(t, err)

		dtc := &dtclient.MockDynatraceClient{}
		dtc.
			On("GetTenantInfo").
			Return(&dtclient.TenantInfo{}, fmt.Errorf("could not retrieve tenant info"))

		desiredStatefulSet, err := r.createDesiredStatefulSet(instance, dtc)
		assert.EqualError(t, err, "could not retrieve tenant info")
		assert.Nil(t, desiredStatefulSet)
	})
	t.Run("CreateDesiredStatefulSet no kube-system", func(t *testing.T) {
		r, instance, err := setupReconciler(t, &mockIsLatestUpdateService{})
		assert.NoError(t, err)

		dtc, err := createFakeDTClient(nil, nil, nil)
		assert.NoError(t, err)

		kubeSystemNamespace := &corev1.Namespace{}
		err = r.client.Get(context.TODO(), client.ObjectKey{Name: _const.KubeSystemNamespace}, kubeSystemNamespace)
		assert.NoError(t, err)

		err = r.client.Delete(context.TODO(), kubeSystemNamespace)
		assert.NoError(t, err)

		desiredStatefulSet, err := r.createDesiredStatefulSet(instance, dtc)
		assert.EqualError(t, err, "namespaces \"kube-system\" not found")
		assert.Nil(t, desiredStatefulSet)
	})
}
