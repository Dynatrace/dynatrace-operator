package dynakube

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	"github.com/Dynatrace/dynatrace-operator/controllers/capability/kubemon"
	"github.com/Dynatrace/dynatrace-operator/controllers/capability/routing"
	"github.com/Dynatrace/dynatrace-operator/controllers/dtpullsecret"
	"github.com/Dynatrace/dynatrace-operator/controllers/dtversion"
	"github.com/Dynatrace/dynatrace-operator/controllers/dynakube/updates"
	"github.com/Dynatrace/dynatrace-operator/controllers/istio"
	"github.com/Dynatrace/dynatrace-operator/controllers/oneagent"
	"github.com/Dynatrace/dynatrace-operator/controllers/utils"
	"github.com/Dynatrace/dynatrace-operator/dtclient"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	defaultUpdateInterval = 5 * time.Minute

	// EnvVarDisableActiveGateUpdates is an internal environment variable to disable ActiveGate updates, since there is
	// no documented way to do so otherwise.
	EnvVarDisableActiveGateUpdates = "OPERATOR_ACTIVEGATE_DISABLE_UPDATES"
)

var log = logf.Log.WithName("controller_dynakube")

func Add(mgr manager.Manager, _ string) error {
	return NewReconciler(mgr).SetupWithManager(mgr)
}

// NewReconciler returns a new ReconcileActiveGate
func NewReconciler(mgr manager.Manager) *ReconcileDynaKube {
	return &ReconcileDynaKube{
		client:        mgr.GetClient(),
		apiReader:     mgr.GetAPIReader(),
		scheme:        mgr.GetScheme(),
		dtcBuildFunc:  BuildDynatraceClient,
		config:        mgr.GetConfig(),
		enableUpdates: os.Getenv(EnvVarDisableActiveGateUpdates) != "false",
	}
}

func (r *ReconcileDynaKube) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&dynatracev1alpha1.DynaKube{}).
		Owns(&appsv1.StatefulSet{}).
		Owns(&appsv1.DaemonSet{}).
		Complete(r)
}

func NewDynaKubeReconciler(c client.Client, apiReader client.Reader, scheme *runtime.Scheme, dtcBuildFunc DynatraceClientFunc, logger logr.Logger, config *rest.Config) *ReconcileDynaKube {
	return &ReconcileDynaKube{
		client:       c,
		apiReader:    apiReader,
		scheme:       scheme,
		dtcBuildFunc: dtcBuildFunc,
		logger:       logger,
		config:       config,
	}
}

// blank assignment to verify that ReconcileActiveGate implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileDynaKube{}

// ReconcileActiveGate reconciles a DynaKube object
type ReconcileDynaKube struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client        client.Client
	apiReader     client.Reader
	scheme        *runtime.Scheme
	dtcBuildFunc  DynatraceClientFunc
	logger        logr.Logger
	config        *rest.Config
	enableUpdates bool
}

type DynatraceClientFunc func(rtc client.Client, instance *dynatracev1alpha1.DynaKube, secret *corev1.Secret) (dtclient.Client, error)

// Reconcile reads that state of the cluster for a DynaKube object and makes changes based on the state read
// and what is in the DynaKube.Spec
// a Pod as an example
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileDynaKube) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("namespace", request.Namespace, "name", request.Name)
	reqLogger.Info("Reconciling DynaKube")

	// Fetch the DynaKube instance
	instance := &dynatracev1alpha1.DynaKube{}
	err := r.client.Get(ctx, request.NamespacedName, instance)
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

	rec := utils.NewReconciliation(reqLogger, instance)
	r.reconcileImpl(ctx, rec)

	if rec.Err != nil {
		if rec.Updated || instance.Status.SetPhaseOnError(rec.Err) {
			if errClient := r.updateCR(ctx, reqLogger, instance); errClient != nil {
				return reconcile.Result{}, fmt.Errorf("failed to update CR after failure, original, %s, then: %w", rec.Err, errClient)
			}
		}

		var serr dtclient.ServerError
		if ok := errors.As(rec.Err, &serr); ok && serr.Code == http.StatusTooManyRequests {
			rec.Log.Info("Request limit for Dynatrace API reached! Next reconcile in one minute")
			return reconcile.Result{RequeueAfter: 1 * time.Minute}, nil
		}

		return reconcile.Result{}, rec.Err
	}

	if rec.Updated {
		if err := r.updateCR(ctx, reqLogger, instance); err != nil {
			return reconcile.Result{}, err
		}
	}

	return reconcile.Result{RequeueAfter: rec.RequeueAfter}, nil
}

func (r *ReconcileDynaKube) reconcileImpl(ctx context.Context, rec *utils.Reconciliation) {
	dtcReconciler := DynatraceClientReconciler{
		Client:              r.client,
		DynatraceClientFunc: r.dtcBuildFunc,
		UpdateAPIToken:      true,
		UpdatePaaSToken:     true,
	}
	dtc, upd, err := dtcReconciler.Reconcile(ctx, rec.Instance)
	rec.Update(upd, defaultUpdateInterval, "Token conditions updated")
	if rec.Error(err) {
		return
	}

	// Fetch api token secret
	secret, err := r.getTokenSecret(ctx, rec.Instance)
	if err != nil {
		rec.Log.Error(err, "could not find token secret")
		return
	}

	if rec.Instance.Spec.EnableIstio {
		if upd, err := istio.NewController(r.config, r.scheme).ReconcileIstio(rec.Instance, dtc); err != nil {
			// If there are errors log them, but move on.
			rec.Log.Info("Istio: failed to reconcile objects", "error", err)
		} else if upd {
			rec.Update(true, 30*time.Second, "Istio: objects updated")
		}
	}

	err = dtpullsecret.
		NewReconciler(r.client, r.apiReader, r.scheme, rec.Instance, dtc, rec.Log, secret).
		Reconcile()
	if rec.Error(err) {
		rec.Log.Error(err, "could not reconcile Dynatrace pull secret")
		return
	}

	upd, err = updates.ReconcileImageVersions(ctx, rec, r.client, r.enableUpdates, dtversion.GetImageVersion)
	rec.Update(upd, defaultUpdateInterval, "Found image updates")
	rec.Error(err)

	if rec.Instance.Spec.KubernetesMonitoringSpec.Enabled {
		upd, err := kubemon.NewReconciler(
			r.client, r.apiReader, r.scheme, dtc, rec.Log, rec.Instance, dtversion.GetImageVersion, r.enableUpdates,
		).Reconcile()
		if rec.Error(err) || rec.Update(upd, defaultUpdateInterval, "kubemon reconciled") {
			return
		}
	} else {
		sts := appsv1.StatefulSet{ObjectMeta: metav1.ObjectMeta{Name: rec.Instance.Name + "-kubemon", Namespace: rec.Instance.Namespace}}
		if err := r.ensureDeleted(&sts); rec.Error(err) {
			return
		}
	}

	if !r.reconcileRouting(rec, dtc) {
		return
	}

	if rec.Instance.Spec.InfraMonitoring.Enabled {
		upd, err := oneagent.NewOneAgentReconciler(
			r.client, r.apiReader, r.scheme, rec.Log, dtc, rec.Instance, &rec.Instance.Spec.InfraMonitoring, oneagent.InframonFeature,
		).Reconcile(ctx, rec)
		if rec.Error(err) || rec.Update(upd, defaultUpdateInterval, "infra monitoring reconciled") {
			return
		}
	} else {
		ds := appsv1.DaemonSet{ObjectMeta: metav1.ObjectMeta{Name: rec.Instance.Name + "-inframon", Namespace: rec.Instance.Namespace}}
		if err := r.ensureDeleted(&ds); rec.Error(err) {
			return
		}
	}

	if rec.Instance.Spec.ClassicFullStack.Enabled {
		upd, err := oneagent.NewOneAgentReconciler(
			r.client, r.apiReader, r.scheme, rec.Log, dtc, rec.Instance, &rec.Instance.Spec.ClassicFullStack, oneagent.ClassicFeature,
		).Reconcile(ctx, rec)
		if rec.Error(err) || rec.Update(upd, defaultUpdateInterval, "classic fullstack reconciled") {
			return
		}
	} else {
		ds := appsv1.DaemonSet{ObjectMeta: metav1.ObjectMeta{Name: rec.Instance.Name + "-classic", Namespace: rec.Instance.Namespace}}
		if err := r.ensureDeleted(&ds); rec.Error(err) {
			return
		}
	}
}

func (r *ReconcileDynaKube) ensureDeleted(obj client.Object) error {
	if err := r.client.Delete(context.TODO(), obj); err != nil && !k8serrors.IsNotFound(err) {
		return err
	}
	return nil
}

func (r *ReconcileDynaKube) reconcileRouting(rec *utils.Reconciliation, dtc dtclient.Client) bool {
	if rec.Instance.Spec.RoutingSpec.Enabled {
		upd, err := routing.NewReconciler(
			r.client, r.apiReader, r.scheme, dtc, rec.Log, rec.Instance, dtversion.GetImageVersion, r.enableUpdates,
		).Reconcile()
		if rec.Error(err) || rec.Update(upd, defaultUpdateInterval, "routing reconciled") {
			return false
		}
	} else {
		sts := appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      rec.Instance.Name + routing.StatefulSetSuffix,
				Namespace: rec.Instance.Namespace,
			},
		}
		if err := r.ensureDeleted(&sts); rec.Error(err) {
			return false
		}
	}
	return true
}

func (r *ReconcileDynaKube) getTokenSecret(ctx context.Context, instance *dynatracev1alpha1.DynaKube) (*corev1.Secret, error) {
	var secret corev1.Secret
	err := r.client.Get(ctx, client.ObjectKey{Name: utils.GetTokensName(instance), Namespace: instance.Namespace}, &secret)
	return &secret, errors.WithStack(err)
}

func (r *ReconcileDynaKube) updateCR(ctx context.Context, log logr.Logger, instance *dynatracev1alpha1.DynaKube) error {
	instance.Status.UpdatedTimestamp = metav1.Now()
	err := r.client.Status().Update(ctx, instance)
	if err != nil && k8serrors.IsConflict(err) {
		// OneAgent reconciler already updates instance which leads to conflict here
		// Only print info in that event
		log.Info("could not update instance due to conflict")
		return nil
	}
	return errors.WithStack(err)
}
