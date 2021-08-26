package namespace

import (
	"context"
	"time"

	mapper "github.com/Dynatrace/dynatrace-operator/namespacemapper"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func Add(mgr manager.Manager, ns string) error {
	return NewReconciler(mgr, ns).SetupWithManager(mgr)
}

func NewReconciler(mgr manager.Manager, ns string) *ReconcileNamespaces {
	return &ReconcileNamespaces{
		client:    mgr.GetClient(),
		apiReader: mgr.GetAPIReader(),
		logger:    log.Log.WithName("namespace.controller"),
		namespace: ns,
	}
}

func (r *ReconcileNamespaces) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Namespace{}).
		Complete(r)
}

type ReconcileNamespaces struct {
	client    client.Client
	apiReader client.Reader
	logger    logr.Logger
	namespace string
}

func (r *ReconcileNamespaces) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	targetNS := request.Name
	logger := r.logger.WithValues("name", targetNS)
	logger.Info("reconciling Namespace")

	var ns corev1.Namespace
	if err := r.client.Get(ctx, client.ObjectKey{Name: targetNS}, &ns); k8serrors.IsNotFound(err) {
		if err := mapper.UnmapFromNamespace(ctx, r.client, r.namespace, targetNS); err != nil {
			return reconcile.Result{}, err
		}
		return reconcile.Result{}, nil
	} else if err != nil {
		return reconcile.Result{}, errors.WithMessage(err, "failed to query Namespace")
	}

	if err := mapper.MapFromNamespace(ctx, r.client, r.namespace, ns); err != nil {
		return reconcile.Result{}, err
	}

	return reconcile.Result{RequeueAfter: 5 * time.Minute}, nil
}
