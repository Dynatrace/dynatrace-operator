package certificates

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/Dynatrace/dynatrace-operator/eventfilter"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	secretPostfix  = "-certs"
	certificate    = "ca.crt"
	oldCertificate = "ca.crt.old"

	errorCertificatesSecretEmpty = "certificates secret is empty"
)

const (
	webhookDeploymentName = "dynatrace-webhook"
)

func Add(mgr manager.Manager, ns string) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1.Deployment{}).
		WithEventFilter(eventfilter.ForObjectNameAndNamespace(webhookDeploymentName, ns)).
		Complete(newWebhookReconciler(mgr))
}

func newWebhookReconciler(mgr manager.Manager) *ReconcileWebhookCertificates {
	return &ReconcileWebhookCertificates{
		client: mgr.GetClient(),
		logger: log.Log.WithName("operator.webhook-certificates"),
	}
}

type ReconcileWebhookCertificates struct {
	ctx       context.Context
	client    client.Client
	namespace string
	logger    logr.Logger
}

func (r *ReconcileWebhookCertificates) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	r.logger.Info("reconciling mutating webhook certificates",
		"namespace", request.Namespace, "name", request.Name)
	r.namespace = request.Namespace
	r.ctx = ctx

	secret, err := r.getSecret()
	if err != nil {
		return reconcile.Result{}, err
	}

	createSecret := false
	if secret == nil {
		createSecret = true
		secret = &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      r.buildSecretName(),
				Namespace: r.namespace,
			},
			Data: map[string][]byte{},
		}
	}

	// todo: validate certs if secret exists (split certs logic) -> skip update if still valid
	certs := Certs{
		Log:     r.logger,
		Domain:  r.getDomain(),
		SrcData: secret.Data,
		Now:     time.Now(),
	}
	if err = certs.ValidateCerts(); err != nil {
		return reconcile.Result{}, err
	}

	if !reflect.DeepEqual(certs.Data, secret.Data) {
		// use generated certificate
		secret.Data = certs.Data
	} else {
		r.logger.Info("secret for certificates up to date, skipping update")
		return reconcile.Result{RequeueAfter: SuccessDuration}, nil
	}

	if createSecret {
		r.logger.Info("creating certificates secret")
		err = r.client.Create(ctx, secret)
		if err != nil {
			return reconcile.Result{}, err
		}
	} else {
		r.logger.Info("updating certificates secret")
		err = r.client.Update(ctx, secret)
		if err != nil {
			return reconcile.Result{}, err
		}
	}

	// load webhook configurations that need certificates
	mutatingWebhook, err := r.getMutatingWebhookConfiguration(ctx)
	if err != nil {
		return reconcile.Result{}, err
	}
	validationWebhook, err := r.getValidationWebhookConfiguration(ctx)
	if err != nil {
		return reconcile.Result{}, err
	}

	// update certificates for webhook configurations
	r.logger.Info("save certificates into webhook configurations")
	webhookConfigurations := []*admissionregistrationv1.WebhookClientConfig{
		&mutatingWebhook.Webhooks[0].ClientConfig,
		&validationWebhook.Webhooks[0].ClientConfig,
	}
	for _, conf := range webhookConfigurations {
		if err := r.updateConfiguration(conf, secret); err != nil {
			return reconcile.Result{}, err
		}
	}

	if err = r.client.Update(ctx, mutatingWebhook); err != nil {
		return reconcile.Result{}, errors.WithStack(err)
	}
	if err = r.client.Update(ctx, validationWebhook); err != nil {
		return reconcile.Result{}, errors.WithStack(err)
	}

	return reconcile.Result{RequeueAfter: SuccessDuration}, nil
}

func (r *ReconcileWebhookCertificates) getMutatingWebhookConfiguration(ctx context.Context) (*admissionregistrationv1.MutatingWebhookConfiguration, error) {
	var mutatingWebhook admissionregistrationv1.MutatingWebhookConfiguration
	err := r.client.Get(ctx, client.ObjectKey{
		Name: webhookDeploymentName,
	}, &mutatingWebhook)
	if err != nil {
		return nil, err
	}

	if len(mutatingWebhook.Webhooks) <= 0 {
		return nil, errors.New("mutating admission webhook configuration has no registered webhooks")
	}
	return &mutatingWebhook, nil
}

func (r *ReconcileWebhookCertificates) getValidationWebhookConfiguration(ctx context.Context) (*admissionregistrationv1.ValidatingWebhookConfiguration, error) {
	var mutatingWebhook admissionregistrationv1.ValidatingWebhookConfiguration
	err := r.client.Get(ctx, client.ObjectKey{
		Name: webhookDeploymentName,
	}, &mutatingWebhook)
	if err != nil {
		return nil, err
	}

	if len(mutatingWebhook.Webhooks) <= 0 {
		return nil, errors.New("mutating validation webhook configuration has no registered webhooks")
	}
	return &mutatingWebhook, nil
}

func (r *ReconcileWebhookCertificates) getSecret() (*corev1.Secret, error) {
	var oldSecret corev1.Secret
	err := r.client.Get(r.ctx, client.ObjectKey{Name: r.buildSecretName(), Namespace: r.namespace}, &oldSecret)
	if k8serrors.IsNotFound(err) {
		return nil, nil
	}
	return &oldSecret, errors.WithStack(err)
}

func (r *ReconcileWebhookCertificates) buildSecretName() string {
	return fmt.Sprintf("%s%s", webhookDeploymentName, secretPostfix)
}

func (r *ReconcileWebhookCertificates) getDomain() string {
	return fmt.Sprintf("%s.%s.svc", webhookDeploymentName, r.namespace)
}

func (r *ReconcileWebhookCertificates) updateConfiguration(
	webhookConfiguration *admissionregistrationv1.WebhookClientConfig, secret *corev1.Secret) error {
	data, hasData := secret.Data[certificate]
	if !hasData {
		err := errors.New(errorCertificatesSecretEmpty)
		r.logger.Error(err, errorCertificatesSecretEmpty)
		return errors.WithStack(err)
	}

	if oldData, hasOldData := secret.Data[oldCertificate]; hasOldData {
		data = append(data, oldData...)
	}

	if webhookConfiguration != nil {
		webhookConfiguration.CABundle = data
	}
	return nil
}
