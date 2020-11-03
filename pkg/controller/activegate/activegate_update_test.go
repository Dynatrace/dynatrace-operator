package activegate

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/apis"
	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-operator/pkg/apis/dynatrace/v1alpha1"
	"github.com/Dynatrace/dynatrace-operator/pkg/controller/builder"
	_const "github.com/Dynatrace/dynatrace-operator/pkg/controller/const"
	"github.com/Dynatrace/dynatrace-operator/pkg/controller/factory"
	"github.com/Dynatrace/dynatrace-operator/pkg/controller/version"
	"github.com/Dynatrace/dynatrace-operator/pkg/dtclient"
	"github.com/go-logr/logr"
	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/kubectl/pkg/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	mockActiveGateVersion = "1.200.0"
)

func init() {
	_ = apis.AddToScheme(scheme.Scheme) // Register OneAgent and Istio object schemas.
	_ = os.Setenv(k8sutil.WatchNamespaceEnvVar, _const.DynatraceNamespace)
}

type mockIsLatestUpdateService struct {
}

func (updateService *mockIsLatestUpdateService) FindOutdatedPods(r *ReconcileActiveGate,
	logger logr.Logger,
	instance *dynatracev1alpha1.DynaKube) ([]corev1.Pod, error) {
	return (&activeGateUpdateService{}).FindOutdatedPods(r, logger, instance)
}
func (updateService *mockIsLatestUpdateService) IsLatest(version.ReleaseValidator) (bool, error) {
	return false, nil
}
func (updateService *mockIsLatestUpdateService) UpdatePods(r *ReconcileActiveGate,
	instance *dynatracev1alpha1.DynaKube) (*reconcile.Result, error) {
	return (&activeGateUpdateService{}).UpdatePods(r, instance)
}

type failingIsLatestUpdateService struct {
	mockIsLatestUpdateService
}

func (updateService *failingIsLatestUpdateService) IsLatest(version.ReleaseValidator) (bool, error) {
	return false, fmt.Errorf("mocked error")
}

func TestFindOutdatedPods(t *testing.T) {
	t.Run("FindOutdatedPods", func(t *testing.T) {
		r, instance, err := setupReconciler(t, &mockIsLatestUpdateService{})
		assert.NotNil(t, r)
		assert.NoError(t, err)

		// Check if r is not nil so go linter does not complain
		if r != nil {
			instance.Spec.KubernetesMonitoringSpec.Image = "test-image"
			pod := &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      instance.Name,
					Namespace: _const.DynatraceNamespace,
					Labels:    builder.BuildLabelsForQuery(instance.Name),
				},
				Status: corev1.PodStatus{
					ContainerStatuses: []corev1.ContainerStatus{
						{
							Image:   "outdated",
							ImageID: "outdated",
						},
					},
				},
			}

			err = r.client.Create(context.TODO(), pod)
			assert.NoError(t, err)

			pods, err := r.updateService.FindOutdatedPods(r, log.WithName("TestUpdatePods"), instance)

			assert.NotNil(t, pods)
			assert.NotEmpty(t, pods)
			assert.Equal(t, 1, len(pods))
			assert.NoError(t, err)
		} else {
			assert.Fail(t, "r is nil")
		}
	})
	t.Run("FindOutdatedPods error during IsLatest", func(t *testing.T) {
		r, instance, err := setupReconciler(t, &failingIsLatestUpdateService{})
		assert.NotNil(t, r)
		assert.NoError(t, err)

		// Check if r is not nil so go linter does not complain
		if r != nil {
			pods, err := r.updateService.FindOutdatedPods(r, log.WithName("TestUpdatePods"), instance)

			assert.Nil(t, pods)
			assert.Empty(t, pods)
			assert.Equal(t, 0, len(pods))
			assert.NoError(t, err)
		} else {
			assert.Fail(t, "r is nil")
		}
	})
	t.Run("FindOutdatedPods instance has no image", func(t *testing.T) {
		r, instance, err := setupReconciler(t, &failingIsLatestUpdateService{})
		assert.NotNil(t, r)
		assert.NoError(t, err)

		// Check if r is not nil so go linter does not complain
		if r != nil {
			instance.Spec.KubernetesMonitoringSpec.Image = ""
			pods, err := r.updateService.FindOutdatedPods(r, log.WithName("TestUpdatePods"), instance)

			assert.Nil(t, pods)
			assert.Empty(t, pods)
			assert.Equal(t, 0, len(pods))
			assert.NoError(t, err)
		} else {
			assert.Fail(t, "r is nil")
		}
	})
}

func TestUpdatePods(t *testing.T) {
	t.Run("UpdatePods", func(t *testing.T) {
		r, instance, err := setupReconciler(t, &mockIsLatestUpdateService{})
		assert.NotNil(t, r)
		assert.NoError(t, err)

		// Check if r is not nil so go linter does not complain
		if r != nil {
			result, err := r.updateService.UpdatePods(r, instance)

			assert.Nil(t, result)
			assert.NoError(t, err)

			pods, err := r.updateService.FindOutdatedPods(r, log.WithName("TestUpdatePods"), instance)

			assert.Nil(t, pods)
			assert.Empty(t, pods)
			assert.Equal(t, 0, len(pods))
			assert.NoError(t, err)
		} else {
			assert.Fail(t, "r is nil")
		}
	})
	t.Run("UpdatePods instance is nil", func(t *testing.T) {
		r, _, err := setupReconciler(t, &mockIsLatestUpdateService{})
		assert.NotNil(t, r)
		assert.NoError(t, err)

		// Check if r is not nil so go linter does not complain
		if r != nil {
			result, err := r.updateService.UpdatePods(r, nil)

			assert.Nil(t, result)
			assert.Error(t, err)
		} else {
			assert.Fail(t, "r is nil")
		}
	})
	t.Run("UpdatePods auto update disabled", func(t *testing.T) {
		r, instance, err := setupReconciler(t, &mockIsLatestUpdateService{})
		assert.NotNil(t, r)
		assert.NoError(t, err)

		// Check if r is not nil so go linter does not complain
		if r != nil {
			instance.Spec.KubernetesMonitoringSpec.DisableActivegateUpdate = true
			instance.Spec.KubernetesMonitoringSpec.Image = "test-image"

			dummy := corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      instance.Name,
					Namespace: _const.DynatraceNamespace,
					Labels:    builder.BuildLabelsForQuery(instance.Name),
				},
				Status: corev1.PodStatus{
					ContainerStatuses: []corev1.ContainerStatus{
						{
							Image:   "outdated",
							ImageID: "outdated",
						},
					},
				},
			}
			err = r.client.Create(context.TODO(), &dummy)
			assert.NoError(t, err)

			result, err := r.updateService.UpdatePods(r, instance)

			assert.Nil(t, result)
			assert.NoError(t, err)

			pods, err := r.updateService.FindOutdatedPods(r, log.WithName("TestUpdatePods"), instance)

			// Since DisableActivegateUpdate is true, UpdatePods should not have deleted outdated pods
			assert.NotNil(t, pods)
			assert.NotEmpty(t, pods)
			assert.Equal(t, 1, len(pods))
			assert.NoError(t, err)
		} else {
			assert.Fail(t, "r is nil")
		}
	})
}

func setupReconciler(t *testing.T, updateService updateService) (*ReconcileActiveGate, *dynatracev1alpha1.DynaKube, error) {
	fakeClient := factory.CreateFakeClient()
	r := &ReconcileActiveGate{
		client:        fakeClient,
		dtcBuildFunc:  createFakeDTClient,
		scheme:        scheme.Scheme,
		updateService: updateService,
	}
	request := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Namespace: _const.DynatraceNamespace,
			Name:      _const.ActivegateName,
		},
	}

	instance := &dynatracev1alpha1.DynaKube{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	assert.NoError(t, err)

	var pod1 corev1.Pod
	pod1.Name = "activegate-pod-1"
	pod1.Status.ContainerStatuses = []corev1.ContainerStatus{
		{
			Image:   "latest",
			ImageID: "latest",
		},
	}

	var pod2 corev1.Pod
	pod2.Name = "activegate-pod-2"
	pod2.Status.ContainerStatuses = []corev1.ContainerStatus{
		{
			Image:   "outdated",
			ImageID: "outdated",
		},
	}

	err = fakeClient.Create(context.TODO(), &pod1)
	assert.NoError(t, err)

	err = fakeClient.Create(context.TODO(), &pod2)
	assert.NoError(t, err)
	return r, instance, err
}

func createFakeDTClient(client.Client, *dynatracev1alpha1.DynaKube, *corev1.Secret) (dtclient.Client, error) {
	dtMockClient := &dtclient.MockDynatraceClient{}
	dtMockClient.On("GetTenantInfo").Return(&dtclient.TenantInfo{}, nil)
	dtMockClient.On("GetConnectionInfo").Return(dtclient.ConnectionInfo{TenantUUID: "abc123456"}, nil)
	dtMockClient.On("reconcilePullSecret").Return(nil)
	dtMockClient.
		On("QueryActiveGates", &dtclient.ActiveGateQuery{Hostname: "", NetworkAddress: "", NetworkZone: "default", UpdateStatus: ""}).
		Return([]dtclient.ActiveGate{
			{Version: mockActiveGateVersion},
		}, nil)
	return dtMockClient, nil
}
