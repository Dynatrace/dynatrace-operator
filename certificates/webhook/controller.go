package webhook

import (
	"context"
	"time"

	"github.com/Dynatrace/dynatrace-operator/certificates"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	v1 "k8s.io/api/apps/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	webhookName = "dynatrace-webhook"
)

func Add(mgr manager.Manager, ns string) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1.Deployment{}).
		WithEventFilter(filterForMutatingDeployment(ns)).
		Complete(newWebhookReconciler(mgr, ns))
}

func newWebhookReconciler(mgr manager.Manager, ns string) *ReconcileWebhookCertificates {
	return &ReconcileWebhookCertificates{
		client:    mgr.GetClient(),
		scheme:    mgr.GetScheme(),
		namespace: ns,
		logger:    log.Log.WithName("operator.webhook-certificates"),
	}
}

// ReconcileWebhookCertificates updates certificates secret for the webhooks
type ReconcileWebhookCertificates struct {
	client    client.Client
	scheme    *runtime.Scheme
	logger    logr.Logger
	namespace string
}

func (r *ReconcileWebhookCertificates) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	r.logger.Info("reconciling mutating webhook certificates", "namespace", request.Namespace, "name", request.Name)

	var mutatingWebhook admissionregistrationv1.MutatingWebhookConfiguration
	err := r.client.Get(ctx, client.ObjectKey{Name: webhookName}, &mutatingWebhook)
	if k8serrors.IsNotFound(err) {
		r.logger.Info("unable to find mutating webhook configuration", "namespace", request.Namespace, "name", request.Name)
		return reconcile.Result{RequeueAfter: 5 * time.Minute}, nil
	} else if err != nil {
		return reconcile.Result{}, errors.WithStack(err)
	}

	if len(mutatingWebhook.Webhooks) <= 0 {
		return reconcile.Result{}, errors.New("mutating webhook configuration has no registered webhooks")
	}

	err = certificates.NewCertificateReconciler(ctx, r.client, webhookName, request.Namespace, r.logger).
		ReconcileCertificateSecretForWebhook(&mutatingWebhook.Webhooks[0].ClientConfig)
	if err != nil {
		return reconcile.Result{}, errors.WithStack(err)
	}

	if err = r.client.Update(ctx, &mutatingWebhook); err != nil {
		return reconcile.Result{}, errors.WithStack(err)
	}

	return reconcile.Result{RequeueAfter: 3 * time.Hour}, nil
}
