package validation

import (
	"context"
	"github.com/Dynatrace/dynatrace-operator/controllers/certificates"
	"github.com/Dynatrace/dynatrace-operator/eventfilter"
	"github.com/Dynatrace/dynatrace-operator/logger"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	admissionv1 "k8s.io/api/admissionregistration/v1"
	v1 "k8s.io/api/apps/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	SecretCertsName = "dynatrace-validation-certs"

	validationWebhookName = "dynatrace-webhook"
)

type webhookReconciler struct {
	logger logr.Logger
	clt    client.Client
}

func Add(mgr manager.Manager, namespace string) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1.Deployment{}).
		WithEventFilter(eventfilter.ForObjectNameAndNamespace(validationWebhookName, namespace)).
		Complete(newWebhookReconciler(mgr))
}

func newWebhookReconciler(mgr manager.Manager) *webhookReconciler {
	return &webhookReconciler{
		logger: logger.NewDTLogger(),
		clt:    mgr.GetClient(),
	}
}

func (r *webhookReconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	r.logger.Info("reconciling validation webhook certificates", "namespace", request.Namespace, "name", request.Name)

	var validationWebhook admissionv1.ValidatingWebhookConfiguration
	err := r.clt.Get(ctx, client.ObjectKey{Name: validationWebhookName}, &validationWebhook)
	if k8serrors.IsNotFound(err) {
		r.logger.Info("unable to find validation webhook configuration", "namespace", request.Namespace, "name", request.Name)
		return reconcile.Result{RequeueAfter: certificates.FiveMinutes}, nil
	} else if err != nil {
		return reconcile.Result{}, errors.WithStack(err)
	}

	if len(validationWebhook.Webhooks) <= 0 {
		return reconcile.Result{}, errors.New("validation webhook configuration has no registered webhooks")
	}

	err = certificates.NewCertificateReconciler(ctx, r.clt, validationWebhookName, request.Namespace, r.logger).
		ReconcileCertificateSecretForWebhook(&validationWebhook.Webhooks[0].ClientConfig)
	if k8serrors.IsNotFound(errors.Cause(err)) {
		return reconcile.Result{RequeueAfter: certificates.Tens}, nil
	} else if err != nil {
		return reconcile.Result{}, err
	}

	if err = r.clt.Update(ctx, &validationWebhook); err != nil {
		return reconcile.Result{}, errors.WithStack(err)
	}

	return reconcile.Result{RequeueAfter: certificates.ThreeHours}, nil
}
