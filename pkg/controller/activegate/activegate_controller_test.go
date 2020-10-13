package activegate

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/Dynatrace/dynatrace-activegate-operator/pkg/apis"
	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-activegate-operator/pkg/apis/dynatrace/v1alpha1"
	"github.com/Dynatrace/dynatrace-activegate-operator/pkg/controller/builder"
	_const "github.com/Dynatrace/dynatrace-activegate-operator/pkg/controller/const"
	"github.com/Dynatrace/dynatrace-activegate-operator/pkg/controller/factory"
	"github.com/Dynatrace/dynatrace-activegate-operator/pkg/dtclient"
	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type failUpdatePodsService struct {
	mockIsLatestUpdateService
}

const errorUpdatingPods = "error updating pods"

func init() {
	_ = apis.AddToScheme(scheme.Scheme) // Register OneAgent and Istio object schemas.
	_ = os.Setenv(k8sutil.WatchNamespaceEnvVar, _const.DynatraceNamespace)
}

func (updateServer *failUpdatePodsService) UpdatePods(*ReconcileActiveGate, *dynatracev1alpha1.ActiveGate) (*reconcile.Result, error) {
	result := builder.ReconcileAfterFiveMinutes()
	return &result, fmt.Errorf(errorUpdatingPods)
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
		assert.Equal(t, result, builder.ReconcileImmediately())

		sts, err := r.newStatefulSetForCR(instance, &dtclient.TenantInfo{}, factory.KubeSystemUID)
		assert.NoError(t, err)
		assert.NotNil(t, sts)

		found := &appsv1.StatefulSet{}
		err = r.client.Get(context.TODO(), types.NamespacedName{Name: sts.Name, Namespace: sts.Namespace}, found)
		assert.NotNil(t, found)
		assert.NoError(t, err)

		assert.Equal(t, sts.Name, found.Name)
		assert.Equal(t, sts.Namespace, found.Namespace)
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
	t.Run("Reconcile pod has uid env", func(t *testing.T) {
		r := &ReconcileActiveGate{
			client:        factory.CreateFakeClient(),
			dtcBuildFunc:  createFakeDTClient,
			scheme:        scheme.Scheme,
			updateService: &mockIsLatestUpdateService{},
		}
		request := reconcile.Request{
			NamespacedName: types.NamespacedName{
				Namespace: _const.DynatraceNamespace,
				Name:      _const.ActivegateName,
			},
		}

		result, err := r.Reconcile(request)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		statefulSet := &appsv1.StatefulSet{}

		err = r.client.Get(context.TODO(), client.ObjectKey{Name: "activegate", Namespace: _const.DynatraceNamespace}, statefulSet)
		assert.NoError(t, err)
		assert.NotNil(t, statefulSet)
		assert.NotNil(t, statefulSet.Spec)
		assert.NotNil(t, statefulSet.Spec.Template)

		pod := statefulSet.Spec.Template
		assert.NotNil(t, pod.Spec)
		assert.NotNil(t, pod.Spec.Containers)
		assert.LessOrEqual(t, 1, len(pod.Spec.Containers))

		for _, container := range pod.Spec.Containers {
			hasNamespace := false
			hasUID := false

			for _, envArg := range container.Env {
				if envArg.Name == builder.DtIdSeedNamespace {
					hasNamespace = true
					assert.Equal(t, _const.DynatraceNamespace, envArg.Value)
				} else if envArg.Name == builder.DtIdSeedClusterId {
					hasUID = true
					assert.Equal(t, factory.KubeSystemUID, envArg.Value)
				}
			}

			assert.True(t, hasNamespace)
			assert.True(t, hasUID)
		}
	})
	t.Run("Reconcile no kube-system namespace", func(t *testing.T) {
		r := &ReconcileActiveGate{
			client:        factory.CreateFakeClient(),
			dtcBuildFunc:  createFakeDTClient,
			scheme:        scheme.Scheme,
			updateService: &mockIsLatestUpdateService{},
		}
		request := reconcile.Request{
			NamespacedName: types.NamespacedName{
				Namespace: _const.DynatraceNamespace,
				Name:      _const.ActivegateName,
			},
		}

		kubeSystemNamespace := corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: _const.KubeSystemNamespace,
			},
		}
		err := r.client.Delete(context.TODO(), &kubeSystemNamespace)
		assert.NoError(t, err)

		result, err := r.Reconcile(request)
		assert.EqualError(t, err, "namespaces \"kube-system\" not found")
		assert.NotNil(t, result)
	})
}
