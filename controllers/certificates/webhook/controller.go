package webhook

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/controllers/certificates"
	"github.com/Dynatrace/dynatrace-operator/eventfilter"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	v1 "k8s.io/api/apps/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	webhookName           = "dynatrace-webhook"
	validationWebhookName = "dynatrace-webhook"
)

func Add(mgr manager.Manager, ns string) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1.Deployment{}).
		WithEventFilter(eventfilter.ForObjectNameAndNamespace(webhookName, ns)).
		Complete(newWebhookReconciler(mgr))
}

func newWebhookReconciler(mgr manager.Manager) *ReconcileWebhookCertificates {
	return &ReconcileWebhookCertificates{
		client: mgr.GetClient(),
		logger: log.Log.WithName("operator.webhook-certificates"),
	}
}

// ReconcileWebhookCertificates updates certificates secret for the webhooks
type ReconcileWebhookCertificates struct {
	client client.Client
	logger logr.Logger
}

func (r *ReconcileWebhookCertificates) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	r.logger.Info("reconciling mutating webhook certificates", "namespace", request.Namespace, "name", request.Name)

	var mutatingWebhook admissionregistrationv1.MutatingWebhookConfiguration
	err := r.client.Get(ctx, client.ObjectKey{Name: webhookName}, &mutatingWebhook)
	if k8serrors.IsNotFound(err) {
		r.logger.Info("unable to find mutating webhook configuration", "namespace", request.Namespace, "name", request.Name)
		return reconcile.Result{RequeueAfter: certificates.FiveMinutes}, nil
	} else if err != nil {
		return reconcile.Result{}, errors.WithStack(err)
	}

	if len(mutatingWebhook.Webhooks) <= 0 {
		return reconcile.Result{}, errors.New("mutating webhook configuration has no registered webhooks")
	}

	var validationWebhook admissionregistrationv1.ValidatingWebhookConfiguration
	err = r.client.Get(ctx, client.ObjectKey{Name: validationWebhookName}, &validationWebhook)
	if k8serrors.IsNotFound(err) {
		r.logger.Info("unable to find validation webhook configuration", "namespace", request.Namespace, "name", request.Name)
		return reconcile.Result{RequeueAfter: certificates.FiveMinutes}, nil
	} else if err != nil {
		return reconcile.Result{}, errors.WithStack(err)
	}

	if len(validationWebhook.Webhooks) <= 0 {
		return reconcile.Result{}, errors.New("validation webhook configuration has no registered webhooks")
	}

	err = certificates.NewCertificateReconciler(ctx, r.client, webhookName, request.Namespace, r.logger).
		ReconcileCertificateSecretForWebhook([]*admissionregistrationv1.WebhookClientConfig{&mutatingWebhook.Webhooks[0].ClientConfig, &validationWebhook.Webhooks[0].ClientConfig})
	if k8serrors.IsNotFound(errors.Cause(err)) {
		return reconcile.Result{RequeueAfter: certificates.Tens}, nil
	} else if err != nil {
		return reconcile.Result{}, err
	}

	if err = r.client.Update(ctx, &mutatingWebhook); err != nil {
		return reconcile.Result{}, errors.WithStack(err)
	}
	if err = r.client.Update(ctx, &validationWebhook); err != nil {
		return reconcile.Result{}, errors.WithStack(err)
	}

	return reconcile.Result{RequeueAfter: certificates.ThreeHours}, nil
}
