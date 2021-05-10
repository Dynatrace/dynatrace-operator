package csigc

import (
	"context"
	"time"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	dtcsi "github.com/Dynatrace/dynatrace-operator/controllers/csi"
	"github.com/Dynatrace/dynatrace-operator/controllers/dynakube"
	"github.com/Dynatrace/dynatrace-operator/dtclient"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
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
	ch := make(chan event.GenericEvent, 10)

	gcController, err := controller.New("garbage-collector", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	err = gcController.Watch(&source.Channel{Source: ch}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	go func() {
		ticker := time.NewTicker(r.opts.GCInterval)
		defer ticker.Stop()

		ch <- event.GenericEvent{
			Object: &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: dtcsi.DriverName}},
		}

		for range ticker.C {
			ch <- event.GenericEvent{
				Object: &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: dtcsi.DriverName}},
			}
		}
	}()

	return nil
}

var _ reconcile.Reconciler = &CSIGarbageCollector{}

func (r *CSIGarbageCollector) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	r.logger.Info("running OneAgent garbage collection", "namespace", request.Namespace, "name", request.Name)

	var dk dynatracev1alpha1.DynaKube
	if err := r.client.Get(ctx, request.NamespacedName, &dk); err != nil {
		if k8serrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}

		r.logger.Error(err, "failed to get DynaKube object")
		return reconcile.Result{}, nil
	}

	var tkns corev1.Secret
	if err := r.client.Get(ctx, client.ObjectKey{Name: dk.Tokens(), Namespace: dk.Namespace}, &tkns); err != nil {
		r.logger.Error(err, "failed to query tokens")
		return reconcile.Result{}, nil
	}

	dtc, err := r.dtcBuildFunc(r.client, &dk, &tkns)
	if err != nil {
		r.logger.Error(err, "failed to create Dynatrace client")
		return reconcile.Result{}, nil
	}

	ci, err := dtc.GetConnectionInfo()
	if err != nil {
		r.logger.Error(err, "failed to fetch connection info")
		return reconcile.Result{}, nil
	}

	ver, err := dtc.GetLatestAgentVersion(dtclient.OsUnix, dtclient.InstallerTypePaaS)
	if err != nil {
		r.logger.Error(err, "failed to query OneAgent version")
		return reconcile.Result{}, nil
	}

	r.logger.Info("running binary garbage collection")
	if err := runBinaryGarbageCollection(r.logger, ci.TenantUUID, ver, r.opts); err != nil {
		r.logger.Error(err, "garbage collection failed")
		return reconcile.Result{}, nil
	}

	return reconcile.Result{}, nil
}
