// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package dtprometheus

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme/fake"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/status"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha1/dtprometheus"
	"github.com/Dynatrace/dynatrace-operator/test/integrationtests"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// reconcileRecorder counts, per DTPrometheus key, how many times the
// reconciler's (fake) client Get was called, which happens exactly once per Reconcile.
type reconcileRecorder struct {
	mu    sync.Mutex
	calls map[client.ObjectKey]int
}

func newReconcileRecorder() *reconcileRecorder {
	return &reconcileRecorder{calls: map[client.ObjectKey]int{}}
}

func (r *reconcileRecorder) record(key client.ObjectKey) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls[key]++
}

func (r *reconcileRecorder) count(key client.ObjectKey) int {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.calls[key]
}

func (r *reconcileRecorder) reset(key client.ObjectKey) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.calls, key)
}

func TestSetupWithManager(t *testing.T) {
	recorder := newReconcileRecorder()
	r := &Reconciler{
		Client: fake.NewClientWithInterceptors(interceptor.Funcs{
			Get: func(_ context.Context, _ client.WithWatch, key client.ObjectKey, _ client.Object, _ ...client.GetOption) error {
				recorder.record(key)

				return k8serrors.NewNotFound(schema.GroupResource{}, key.Name)
			},
		}),
	}

	clt := integrationtests.SetupManagerTestEnvironment(t, func(mgr ctrl.Manager) error {
		return r.SetupWithManager(mgr)
	})

	const (
		eventuallyTimeout  = 10 * time.Second
		eventuallyInterval = 100 * time.Millisecond
		neverTimeout       = 2 * time.Second
	)

	// createDTPrometheus creates a DTPrometheus in the default namespace, then waits
	// for the initial reconcile triggered by the primary "For" watch. Waiting here
	// means later assertions can attribute any *additional* reconcile to the
	// specific watch under test, rather than to this setup step.
	createDTPrometheus := func(t *testing.T, name, dynaKubeName string) (*dtprometheus.DTPrometheus, client.ObjectKey) {
		t.Helper()

		dtprom := &dtprometheus.DTPrometheus{
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: metav1.NamespaceDefault},
			Spec:       dtprometheus.DTPrometheusSpec{DynaKubeName: dynaKubeName},
		}
		require.NoError(t, clt.Create(t.Context(), dtprom))

		key := client.ObjectKeyFromObject(dtprom)
		require.Eventually(t, func() bool { return recorder.count(key) > 0 }, eventuallyTimeout, eventuallyInterval)

		return dtprom, key
	}

	// assertReconcileTriggered resets the recorder for key, runs action, and asserts
	// that it caused a reconcile, isolating the effect of action from anything
	// counted against key beforehand (e.g. the initial reconcile from createDTPrometheus).
	assertReconcileTriggered := func(t *testing.T, key client.ObjectKey, action func()) {
		t.Helper()

		recorder.reset(key)
		action()

		require.Eventually(t, func() bool { return recorder.count(key) > 0 }, eventuallyTimeout, eventuallyInterval)
	}

	// assertReconcileNotTriggered resets the recorder for key, runs action, and asserts
	// that it did not cause a reconcile within neverTimeout.
	assertReconcileNotTriggered := func(t *testing.T, key client.ObjectKey, action func()) {
		t.Helper()

		recorder.reset(key)
		action()

		require.Never(t, func() bool { return recorder.count(key) > 0 }, neverTimeout, eventuallyInterval)
	}

	t.Run("update to an owned ConfigMap triggers reconcile", func(t *testing.T) {
		dtprom, key := createDTPrometheus(t, "dtprom-configmap", "dk")

		cm := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "owned-configmap", Namespace: dtprom.Namespace}}
		require.NoError(t, controllerutil.SetControllerReference(dtprom, cm, scheme.Scheme))

		assertReconcileTriggered(t, key, func() {
			require.NoError(t, clt.Create(t.Context(), cm))
		})
	})

	t.Run("update to an owned Deployment triggers reconcile", func(t *testing.T) {
		dtprom, key := createDTPrometheus(t, "dtprom-deployment", "dk")

		dep := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{Name: "owned-deployment", Namespace: dtprom.Namespace},
			Spec: appsv1.DeploymentSpec{
				Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "owned-deployment"}},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": "owned-deployment"}},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{{Name: "test", Image: "dummy-app-img:1.0.0"}},
					},
				},
			},
		}
		require.NoError(t, controllerutil.SetControllerReference(dtprom, dep, scheme.Scheme))

		assertReconcileTriggered(t, key, func() {
			require.NoError(t, clt.Create(t.Context(), dep))
		})
	})

	t.Run("update to an owned Service triggers reconcile", func(t *testing.T) {
		dtprom, key := createDTPrometheus(t, "dtprom-service", "dk")

		svc := &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{Name: "owned-service", Namespace: dtprom.Namespace},
			Spec:       corev1.ServiceSpec{Ports: []corev1.ServicePort{{Port: 80}}},
		}
		require.NoError(t, controllerutil.SetControllerReference(dtprom, svc, scheme.Scheme))

		assertReconcileTriggered(t, key, func() {
			require.NoError(t, clt.Create(t.Context(), svc))
		})
	})

	t.Run("DynaKube phase change triggers reconcile", func(t *testing.T) {
		dtprom, key := createDTPrometheus(t, "dtprom-dynakube", "dk")

		dk := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{Name: dtprom.Spec.DynaKubeName, Namespace: dtprom.Namespace},
			Spec:       dynakube.DynaKubeSpec{APIURL: "https://dummy.dynatrace.com/api"},
			Status:     dynakube.DynaKubeStatus{Phase: status.Running},
		}

		assertReconcileTriggered(t, key, func() {
			// Create leaves Status empty; UpdateStatus then performs the phase-changing
			// Update event that the watch predicate looks for.
			integrationtests.CreateDynakube(t, clt, dk)
		})
	})

	t.Run("DynaKube update without a phase change does not trigger reconcile", func(t *testing.T) {
		dtprom, key := createDTPrometheus(t, "dtprom-dynakube-nophase", "dk-nophase")

		dk := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{Name: dtprom.Spec.DynaKubeName, Namespace: dtprom.Namespace},
			Spec:       dynakube.DynaKubeSpec{APIURL: "https://dummy.dynatrace.com/api"},
			Status:     dynakube.DynaKubeStatus{Phase: status.Running},
		}
		assertReconcileTriggered(t, key, func() {
			integrationtests.CreateDynakube(t, clt, dk)
		})

		assertReconcileNotTriggered(t, key, func() {
			dk.Annotations = map[string]string{"foo": "bar"}
			require.NoError(t, clt.Update(t.Context(), dk))
		})
	})

	t.Run("DynaKube resource attributes change triggers reconcile", func(t *testing.T) {
		dtprom, key := createDTPrometheus(t, "dtprom-dynakube-resattrs", "dk-resattrs")

		dk := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{Name: dtprom.Spec.DynaKubeName, Namespace: dtprom.Namespace},
			Spec:       dynakube.DynaKubeSpec{APIURL: "https://dummy.dynatrace.com/api"},
			Status:     dynakube.DynaKubeStatus{Phase: status.Running},
		}
		assertReconcileTriggered(t, key, func() {
			integrationtests.CreateDynakube(t, clt, dk)
		})

		assertReconcileTriggered(t, key, func() {
			dk.Spec.ResourceAttributes = map[string]string{"region": "us-east"}
			require.NoError(t, clt.Update(t.Context(), dk))
		})
	})

	t.Run("phase change on a DynaKube not referenced by any DTPrometheus does not trigger reconcile", func(t *testing.T) {
		_, key := createDTPrometheus(t, "dtprom-dynakube-unreferenced", "dk-referenced")

		unreferencedDK := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{Name: "dk-unreferenced", Namespace: metav1.NamespaceDefault},
			Spec:       dynakube.DynaKubeSpec{APIURL: "https://dummy.dynatrace.com/api"},
			Status:     dynakube.DynaKubeStatus{Phase: status.Running},
		}

		assertReconcileNotTriggered(t, key, func() {
			integrationtests.CreateDynakube(t, clt, unreferencedDK)
		})
	})

	t.Run("DynaKube deletion triggers reconcile", func(t *testing.T) {
		dtprom, key := createDTPrometheus(t, "dtprom-dynakube-delete", "dk-delete")

		dk := &dynakube.DynaKube{
			ObjectMeta: metav1.ObjectMeta{Name: dtprom.Spec.DynaKubeName, Namespace: dtprom.Namespace},
			Spec:       dynakube.DynaKubeSpec{APIURL: "https://dummy.dynatrace.com/api"},
			Status:     dynakube.DynaKubeStatus{Phase: status.Running},
		}

		integrationtests.CreateDynakube(t, clt, dk)

		assertReconcileTriggered(t, key, func() {
			require.NoError(t, clt.Delete(t.Context(), dk))
		})
	})
}
