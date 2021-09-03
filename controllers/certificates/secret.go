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
	r.namespace = request.Namespace
	r.ctx = ctx
	r.logger.Info("reconciling mutating webhook certificates", "namespace", r.namespace, "name", request.Name)

	mutatingWebhook, err := r.getMutatingWebhookConfiguration(ctx)
	if reconcileRes, err := r.handleNotFoundErr(err); err != nil {
		return reconcileRes, err
	}

	validationWebhook, err := r.getValidationWebhookConfiguration(ctx)
	if reconcileRes, err := r.handleNotFoundErr(err); err != nil {
		return reconcileRes, err
	}

	secret, createSecret, err := r.validateAndGenerateSecretAndWebhookCA([]*admissionregistrationv1.WebhookClientConfig{&mutatingWebhook.Webhooks[0].ClientConfig, &validationWebhook.Webhooks[0].ClientConfig})
	if k8serrors.IsNotFound(errors.Cause(err)) {
		return reconcile.Result{RequeueAfter: SecretMissingDuration}, nil
	} else if err != nil {
		return reconcile.Result{}, err
	}

	if secret == nil {
		r.logger.Info("secret for certificates up to date, skipping update", "namespace", r.namespace)
		return reconcile.Result{}, nil
	}

	r.logger.Info("update mutating webhook configuration", "namespace", r.namespace)
	if err = r.client.Update(ctx, mutatingWebhook); err != nil {
		return reconcile.Result{}, errors.WithStack(err)
	}
	if err = r.client.Update(ctx, validationWebhook); err != nil {
		return reconcile.Result{}, errors.WithStack(err)
	}

	if createSecret {
		r.logger.Info("creating secret", "namespace", r.namespace)
		err = r.client.Create(ctx, secret)
		if err != nil {
			return reconcile.Result{}, errors.WithStack(err)
		}
	} else {
		r.logger.Info("updating secret", "namespace", r.namespace)
		err = r.client.Update(ctx, secret)
		if err != nil {
			return reconcile.Result{}, errors.WithStack(err)
		}
	}

	return reconcile.Result{RequeueAfter: SuccessDuration}, nil
}

func (r *ReconcileWebhookCertificates) handleNotFoundErr(err error) (reconcile.Result, error) {
	if k8serrors.IsNotFound(err) {
		r.logger.Info("unable to find webhook configuration", "namespace", r.namespace)
		return reconcile.Result{RequeueAfter: WebhookMissingDuration}, nil
	} else if err != nil {
		return reconcile.Result{}, errors.WithStack(err)
	}
	return reconcile.Result{}, nil
}

func (r *ReconcileWebhookCertificates) getMutatingWebhookConfiguration(ctx context.Context) (*admissionregistrationv1.MutatingWebhookConfiguration, error) {
	var mutatingWebhook admissionregistrationv1.MutatingWebhookConfiguration
	err := r.client.Get(ctx, client.ObjectKey{
		Name:      webhookDeploymentName,
		Namespace: r.namespace,
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
		Name:      webhookDeploymentName,
		Namespace: r.namespace,
	}, &mutatingWebhook)
	if err != nil {
		return nil, err
	}

	if len(mutatingWebhook.Webhooks) <= 0 {
		return nil, errors.New("mutating validation webhook configuration has no registered webhooks")
	}
	return &mutatingWebhook, nil
}

func (r *ReconcileWebhookCertificates) validateAndGenerateSecretAndWebhookCA(webhookConfiguration []*admissionregistrationv1.WebhookClientConfig) (*corev1.Secret, bool, error) {
	r.logger.Info("reconciling certificates")

	secret, createSecret, err := r.validateAndBuildDesiredSecret()
	if err != nil {
		return nil, false, errors.WithStack(err)
	}
	if secret == nil {
		return nil, false, nil
	}

	for _, conf := range webhookConfiguration {
		r.logger.Info("save CA into admission webhook configuration", "namespace", r.namespace)
		if err := r.updateConfiguration(conf, *secret); err != nil {
			return nil, false, errors.WithStack(err)
		}
	}

	return secret, createSecret, nil
}

func (r *ReconcileWebhookCertificates) validateAndBuildDesiredSecret() (*corev1.Secret, bool, error) {
	var data map[string][]byte
	create := true

	oldSecret, err := r.getSecret()
	if err != nil {
		return nil, false, nil
	}

	if oldSecret != nil {
		create = false
		data = oldSecret.Data
	}

	certs := Certs{
		Log:     r.logger,
		Domain:  r.getDomain(),
		SrcData: data,
		Now:     time.Now(),
	}
	if err = certs.ValidateCerts(); err != nil {
		return nil, false, errors.WithStack(err)
	}

	if create {
		return r.buildDesiredSecret(certs), create, nil
	}

	if !reflect.DeepEqual(certs.Data, oldSecret.Data) {
		return r.buildDesiredSecret(certs), create, nil
	}
	return nil, false, nil
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

func (r *ReconcileWebhookCertificates) buildDesiredSecret(certs Certs) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      r.buildSecretName(),
			Namespace: r.namespace,
		},
		Data: certs.Data,
	}
}

func (r *ReconcileWebhookCertificates) updateConfiguration(webhookConfiguration *admissionregistrationv1.WebhookClientConfig, secret corev1.Secret) error {
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
