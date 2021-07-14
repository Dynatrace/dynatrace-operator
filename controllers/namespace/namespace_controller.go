package namespace

import (
	"context"
	"time"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func Add(mgr manager.Manager, _ string) error {
	return NewReconciler(mgr).SetupWithManager(mgr)
}

func NewReconciler(mgr manager.Manager) *ReconcileNamespaces {
	return &ReconcileNamespaces{
		client:    mgr.GetClient(),
		apiReader: mgr.GetAPIReader(),
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
}

func (r *ReconcileNamespaces) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	targetNS := request.Name
	log := r.logger.WithValues("name", targetNS)
	log.Info("reconciling Namespace")

	var ns corev1.Namespace
	if err := r.client.Get(ctx, client.ObjectKey{Name: targetNS}, &ns); k8serrors.IsNotFound(err) {
		return reconcile.Result{}, nil
	} else if err != nil {
		return reconcile.Result{}, errors.WithMessage(err, "failed to query Namespace")
	}

	//ToDo implement mapping logic

	return reconcile.Result{RequeueAfter: 5 * time.Minute}, nil
}
