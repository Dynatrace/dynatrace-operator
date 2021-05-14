package csigc

import (
	"context"
	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	dtcsi "github.com/Dynatrace/dynatrace-operator/controllers/csi"
	"github.com/Dynatrace/dynatrace-operator/controllers/dynakube"
	"github.com/Dynatrace/dynatrace-operator/dtclient"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// CSIGarbageCollector removes unused and outdated agent versions
type CSIGarbageCollector struct {
	client       client.Client
	logger       logr.Logger
	opts         dtcsi.CSIOptions
	dtcBuildFunc dynakube.DynatraceClientFunc
}

// NewReconciler returns a new CSIGarbageCollector
func NewReconciler(client client.Client, opts dtcsi.CSIOptions) *CSIGarbageCollector {
	return &CSIGarbageCollector{
		client:       client,
		logger:       log.Log.WithName("csi.gc.controller"),
		opts:         opts,
		dtcBuildFunc: dynakube.BuildDynatraceClient,
	}
}

func (r *CSIGarbageCollector) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&dynatracev1alpha1.DynaKube{}).
		Complete(r)
}

var _ reconcile.Reconciler = &CSIGarbageCollector{}

func (r *CSIGarbageCollector) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	r.logger.Info("running OneAgent garbage collection", "namespace", request.Namespace, "name", request.Name)
	reconcileResult := reconcile.Result{RequeueAfter: r.opts.GCInterval}

	var dk dynatracev1alpha1.DynaKube
	if err := r.client.Get(ctx, request.NamespacedName, &dk); err != nil {
		if k8serrors.IsNotFound(err) {
			r.logger.Error(err, "given DynaKube object not found")
			return reconcileResult, nil
		}

		r.logger.Error(err, "failed to get DynaKube object")
		return reconcileResult, nil
	}

	var tkns corev1.Secret
	if err := r.client.Get(ctx, client.ObjectKey{Name: dk.Tokens(), Namespace: dk.Namespace}, &tkns); err != nil {
		r.logger.Error(err, "failed to query tokens")
		return reconcileResult, nil
	}

	dtc, err := r.dtcBuildFunc(r.client, &dk, &tkns)
	if err != nil {
		r.logger.Error(err, "failed to create Dynatrace client")
		return reconcileResult, nil
	}

	ci, err := dtc.GetConnectionInfo()
	if err != nil {
		r.logger.Error(err, "failed to fetch connection info")
		return reconcileResult, nil
	}

	ver, err := dtc.GetLatestAgentVersion(dtclient.OsUnix, dtclient.InstallerTypePaaS)
	if err != nil {
		r.logger.Error(err, "failed to query OneAgent version")
		return reconcileResult, nil
	}

	r.logger.Info("running binary garbage collection")
	if err := runBinaryGarbageCollection(r.logger, ci.TenantUUID, ver, r.opts); err != nil {
		r.logger.Error(err, "garbage collection failed")
		return reconcileResult, nil
	}

	return reconcileResult, nil
}
