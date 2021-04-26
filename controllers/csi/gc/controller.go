package csigc

import (
	"context"
	"fmt"
	"time"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	dtcsi "github.com/Dynatrace/dynatrace-operator/controllers/csi"
	"github.com/Dynatrace/dynatrace-operator/controllers/dynakube"
	"github.com/Dynatrace/dynatrace-operator/controllers/utils"
	"github.com/Dynatrace/dynatrace-operator/dtclient"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// CSIGarbageCollector reconciles a DynaKube object
type CSIGarbageCollector struct {
	client       client.Client
	scheme       *runtime.Scheme
	logger       logr.Logger
	namespace    string
	dtcBuildFunc dynakube.DynatraceClientFunc
}

func AddToManager(mgr manager.Manager, ns string) error {

	return add(mgr, &CSIGarbageCollector{
		client:       mgr.GetClient(),
		scheme:       mgr.GetScheme(),
		logger:       log.Log.WithName("csi.gc.controller"),
		namespace:    ns,
		dtcBuildFunc: dynakube.BuildDynatraceClient,
	})
}

func add(mgr manager.Manager, r *CSIGarbageCollector) error {
	// Create a new controller
	c, err := controller.New("csi-gc-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	ch := make(chan event.GenericEvent, 10)

	if err = c.Watch(&source.Channel{Source: ch}, &handler.EnqueueRequestForObject{}); err != nil {
		return err
	}

	// Create artificial requests
	go func() {
		// Because of https://github.com/kubernetes-sigs/controller-runtime/issues/942, waiting
		// some time before inserting an element so that the Channel has time to initialize.
		time.Sleep(10 * time.Second)

		ticker := time.NewTicker(15 * time.Minute)
		defer ticker.Stop()

		ch <- event.GenericEvent{
			Object: &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: dtcsi.DriverName, Namespace: r.namespace}},
		}

		for range ticker.C {
			ch <- event.GenericEvent{
				Object: &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: dtcsi.DriverName, Namespace: r.namespace}},
			}
		}
	}()

	if err = mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		r.logger.Error(err, "could not start health endpoint for operator")
	}

	if err = mgr.AddReadyzCheck("healthz", healthz.Ping); err != nil {
		r.logger.Error(err, "could not start ready endpoint for operator")
	}

	return nil
}

func (r *CSIGarbageCollector) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	r.logger.Info("reconciling csi driver", "namespace", request.Namespace, "name", request.Name)

	var dk dynatracev1alpha1.DynaKube
	if err := r.client.Get(ctx, request.NamespacedName, &dk); err != nil {
		if k8serrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	var tkns corev1.Secret
	if err := r.client.Get(ctx, client.ObjectKey{Name: utils.GetTokensName(&dk), Namespace: dk.Namespace}, &tkns); err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to query tokens: %w", err)
	}

	dtc, err := r.dtcBuildFunc(r.client, &dk, &tkns)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to create Dynatrace client: %w", err)
	}

	ci, err := dtc.GetConnectionInfo()
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to fetch connection info: %w", err)
	}

	ver, err := dtc.GetLatestAgentVersion(dtclient.OsUnix, dtclient.InstallerTypePaaS)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to query OneAgent version: %w", err)
	}

	if err := runBinaryGarbageCollection(r.logger, ci.TenantUUID, ver); err != nil {
		return reconcile.Result{}, fmt.Errorf("garbage collection failed with the following error: %w", err)
	}

	return reconcile.Result{}, nil
}
