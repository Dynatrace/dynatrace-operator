package csigc

import (
	"context"
	"fmt"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	dtcsi "github.com/Dynatrace/dynatrace-operator/controllers/csi"
	"github.com/Dynatrace/dynatrace-operator/controllers/dynakube"
	"github.com/Dynatrace/dynatrace-operator/controllers/utils"
	"github.com/Dynatrace/dynatrace-operator/dtclient"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
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

	var dk dynatracev1alpha1.DynaKube
	if err := r.client.Get(ctx, request.NamespacedName, &dk); err != nil {
		if k8serrors.IsNotFound(err) {
			return r.newReconcileResult(), nil
		}
		return r.newReconcileResult(), errors.WithStack(err)
	}

	var tkns corev1.Secret
	if err := r.client.Get(ctx, client.ObjectKey{Name: utils.GetTokensName(&dk), Namespace: dk.Namespace}, &tkns); err != nil {
		return r.newReconcileResult(), fmt.Errorf("failed to query tokens: %w", err)
	}

	dtc, err := r.dtcBuildFunc(r.client, &dk, &tkns)
	if err != nil {
		return r.newReconcileResult(), fmt.Errorf("failed to create Dynatrace client: %w", err)
	}

	ci, err := dtc.GetConnectionInfo()
	if err != nil {
		return r.newReconcileResult(), fmt.Errorf("failed to fetch connection info: %w", err)
	}

	ver, err := dtc.GetLatestAgentVersion(dtclient.OsUnix, dtclient.InstallerTypePaaS)
	if err != nil {
		return r.newReconcileResult(), fmt.Errorf("failed to query OneAgent version: %w", err)
	}

	if err := runBinaryGarbageCollection(r.logger, ci.TenantUUID, ver, r.opts); err != nil {
		return r.newReconcileResult(), fmt.Errorf("garbage collection failed: %w", err)
	}

	return r.newReconcileResult(), nil
}

func (r *CSIGarbageCollector) newReconcileResult() reconcile.Result {
	return reconcile.Result{
		Requeue:      false,
		RequeueAfter: r.opts.GCInterval,
	}
}
