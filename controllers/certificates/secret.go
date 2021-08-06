package certificates

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	admissionv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	secretPostfix  = "-certs"
	certificate    = "ca.crt"
	oldCertificate = "ca.crt.old"

	errorCertificatesSecretEmpty = "certificates secret is empty"
)

type CertificateReconciler struct {
	ctx         context.Context
	clt         client.Client
	webhookName string
	namespace   string
	logger      logr.Logger
}

func NewCertificateReconciler(ctx context.Context, clt client.Client, webhookName string, namespace string, logger logr.Logger) *CertificateReconciler {
	return &CertificateReconciler{
		ctx:         ctx,
		clt:         clt,
		webhookName: webhookName,
		namespace:   namespace,
		logger:      logger,
	}
}

func (r *CertificateReconciler) ReconcileCertificateSecretForWebhook(webhookConfiguration *admissionv1.WebhookClientConfig) error {
	r.logger.Info("reconciling certificates", "webhook", r.webhookName)

	if err := r.createOrUpdateSecret(); err != nil {
		return errors.WithStack(err)
	}

	if err := r.updateConfiguration(webhookConfiguration); err != nil {
		return errors.WithStack(err)
	}
	return nil
}

func (r *CertificateReconciler) createOrUpdateSecret() error {
	var data map[string][]byte
	create := true

	oldSecret, err := r.getSecret()
	if err != nil {
		return nil
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
		return errors.WithStack(err)
	}

	if create {
		err = r.createSecret(certs)
	} else {
		err = r.updateSecret(certs, oldSecret)
	}
	return errors.WithStack(err)
}

func (r *CertificateReconciler) getSecret() (*corev1.Secret, error) {
	var oldSecret corev1.Secret
	err := r.clt.Get(r.ctx, client.ObjectKey{Name: r.buildSecretName(), Namespace: r.namespace}, &oldSecret)
	if k8serrors.IsNotFound(err) {
		return nil, nil
	}
	return &oldSecret, errors.WithStack(err)
}

func (r *CertificateReconciler) buildSecretName() string {
	return fmt.Sprintf("%s%s", r.webhookName, secretPostfix)
}

func (r *CertificateReconciler) getDomain() string {
	return fmt.Sprintf("%s.%s.svc", r.webhookName, r.namespace)
}

func (r *CertificateReconciler) createSecret(certs Certs) error {
	r.logger.Info("creating secret for certificates", "webhook", r.webhookName, "namespace", r.namespace)
	return r.clt.Create(r.ctx, r.buildDesiredSecret(certs))
}

func (r *CertificateReconciler) updateSecret(certs Certs, oldSecret *corev1.Secret) error {
	if !reflect.DeepEqual(certs.Data, oldSecret.Data) {
		r.logger.Info("updating secret for certificates", "webhook", r.webhookName, "namespace", r.namespace)
		return r.clt.Update(r.ctx, r.buildDesiredSecret(certs))
	}
	r.logger.Info("secret for certificates up to date, skipping update", "webhook", r.webhookName, "namespace", r.namespace)
	return nil
}

func (r *CertificateReconciler) buildDesiredSecret(certs Certs) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      r.buildSecretName(),
			Namespace: r.namespace,
		},
		Data: certs.Data,
	}
}

func (r *CertificateReconciler) updateConfiguration(webhookConfiguration *admissionv1.WebhookClientConfig) error {
	secret, err := r.getSecret()
	if err != nil {
		return errors.WithStack(err)
	}
	if secret == nil {
		return errors.Errorf("secret '%s' does not exist", r.buildSecretName())
	}

	data, hasData := secret.Data[certificate]
	if !hasData {
		err = errors.New(errorCertificatesSecretEmpty)
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
