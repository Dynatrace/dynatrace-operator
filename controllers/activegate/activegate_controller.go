package activegate

import (
	"context"
	"time"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	"github.com/Dynatrace/dynatrace-operator/controllers/dtversion"
	"github.com/Dynatrace/dynatrace-operator/controllers/kubemon"
	"github.com/Dynatrace/dynatrace-operator/dtclient"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var log = logf.Log.WithName("controller_dynakube")

// NewReconciler returns a new ReconcileActiveGate
func NewReconciler(mgr manager.Manager) *ReconcileActiveGate {
	return &ReconcileActiveGate{
		client:       mgr.GetClient(),
		apiReader:    mgr.GetAPIReader(),
		scheme:       mgr.GetScheme(),
		dtcBuildFunc: BuildDynatraceClient,
	}
}

func (r *ReconcileActiveGate) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&dynatracev1alpha1.DynaKube{}).
		Owns(&appsv1.StatefulSet{}).
		Complete(r)
}

// blank assignment to verify that ReconcileActiveGate implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileActiveGate{}

// ReconcileActiveGate reconciles a DynaKube object
type ReconcileActiveGate struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client       client.Client
	apiReader    client.Reader
	scheme       *runtime.Scheme
	dtcBuildFunc DynatraceClientFunc
}

type DynatraceClientFunc func(rtc client.Client, instance *dynatracev1alpha1.DynaKube, secret *corev1.Secret) (dtclient.Client, error)

// Reconcile reads that state of the cluster for a DynaKube object and makes changes based on the state read
// and what is in the DynaKube.Spec
// a Pod as an example
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileActiveGate) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling DynaKube")

	// Fetch the DynaKube instance
	instance := &dynatracev1alpha1.DynaKube{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	// Fetch api token secret
	secret, err := r.getTokenSecret(instance)
	if err != nil {
		reqLogger.Error(err, "could not find token secret")
		return reconcile.Result{}, err
	}

	dtc, err := r.dtcBuildFunc(r.client, instance, secret)
	if err != nil {
		return reconcile.Result{}, err
	}

	if instance.Spec.KubernetesMonitoringSpec.Enabled {
		result, err := kubemon.NewReconciler(
			r.client, r.apiReader, r.scheme, dtc, reqLogger, secret, instance, dtversion.GetImageVersion,
		).Reconcile(request)
		if err != nil {
			reqLogger.Error(err, "could not reconcile kubernetes monitoring")
			return result, err
		}
	}

	reqLogger.Info("Nothing to do: Instance is ready", "Namespace", instance.Namespace, "Name", instance.Name)
	return reconcile.Result{RequeueAfter: 5 * time.Minute}, nil
}

func (r *ReconcileActiveGate) getTokenSecret(instance *dynatracev1alpha1.DynaKube) (*corev1.Secret, error) {
	var secret corev1.Secret
	err := r.client.Get(context.TODO(), client.ObjectKey{Name: GetTokensName(instance), Namespace: instance.Namespace}, &secret)
	return &secret, err
}
