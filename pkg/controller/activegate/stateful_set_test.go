package activegate

import (
	"context"
	"fmt"
	"testing"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-operator/pkg/apis/dynatrace/v1alpha1"
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

		desiredStatefulSet, err := r.createDesiredStatefulSet(instance, &corev1.Secret{})
		assert.NoError(t, err)
		assert.NotNil(t, desiredStatefulSet)
	})
	t.Run("CreateDesiredStatefulSet error creating dynatrace client", func(t *testing.T) {
		r, instance, err := setupReconciler(t, &mockIsLatestUpdateService{})
		assert.NoError(t, err)

		r.dtcBuildFunc = func(rtc client.Client, instance *dynatracev1alpha1.ActiveGate, secret *corev1.Secret) (dtclient.Client, error) {
			return nil, fmt.Errorf("could not create dynatrace client")
		}
		desiredStatefulSet, err := r.createDesiredStatefulSet(instance, &corev1.Secret{})
		assert.EqualError(t, err, "could not create dynatrace client")
		assert.Nil(t, desiredStatefulSet)
	})
	t.Run("CreateDesiredStatefulSet error getting tenant info", func(t *testing.T) {
		r, instance, err := setupReconciler(t, &mockIsLatestUpdateService{})
		assert.NoError(t, err)

		r.dtcBuildFunc = func(rtc client.Client, instance *dynatracev1alpha1.ActiveGate, secret *corev1.Secret) (dtclient.Client, error) {
			mockClient := &dtclient.MockDynatraceClient{}
			mockClient.
				On("GetTenantInfo").
				Return(&dtclient.TenantInfo{}, fmt.Errorf("could not retrieve tenant info"))
			return mockClient, nil
		}
		desiredStatefulSet, err := r.createDesiredStatefulSet(instance, &corev1.Secret{})
		assert.EqualError(t, err, "could not retrieve tenant info")
		assert.Nil(t, desiredStatefulSet)
	})
	t.Run("CreateDesiredStatefulSet no kube-system", func(t *testing.T) {
		r, instance, err := setupReconciler(t, &mockIsLatestUpdateService{})
		assert.NoError(t, err)

		kubeSystemNamespace := &corev1.Namespace{}
		err = r.client.Get(context.TODO(), client.ObjectKey{Name: _const.KubeSystemNamespace}, kubeSystemNamespace)
		assert.NoError(t, err)

		err = r.client.Delete(context.TODO(), kubeSystemNamespace)
		assert.NoError(t, err)

		desiredStatefulSet, err := r.createDesiredStatefulSet(instance, &corev1.Secret{})
		assert.EqualError(t, err, "namespaces \"kube-system\" not found")
		assert.Nil(t, desiredStatefulSet)
	})
}
