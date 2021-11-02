package certificates

import (
	"bytes"
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/Dynatrace/dynatrace-operator/eventfilter"
	"github.com/Dynatrace/dynatrace-operator/webhook"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apiv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	SuccessDuration = 3 * time.Hour

	crdName                      = "dynakubes.dynatrace.com"
	secretPostfix                = "-certs"
	errorCertificatesSecretEmpty = "certificates secret is empty"
)

func Add(mgr manager.Manager, ns string) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&appsv1.Deployment{}).
		WithEventFilter(eventfilter.ForObjectNameAndNamespace(webhook.DeploymentName, ns)).
		Complete(newWebhookReconciler(mgr, func() {}))
}

func AddBootstrap(mgr manager.Manager, ns string, cancelMgr context.CancelFunc) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&appsv1.Deployment{}).
		WithEventFilter(eventfilter.ForObjectNameAndNamespace(webhook.DeploymentName, ns)).
		Complete(newWebhookReconciler(mgr, cancelMgr))
}

func newWebhookReconciler(mgr manager.Manager, cancelMgr context.CancelFunc) *ReconcileWebhookCertificates {
	return &ReconcileWebhookCertificates{
		cancelBootstrapper: cancelMgr,
		client:             mgr.GetClient(),
		logger:             log.Log.WithName("operator.webhook-certificates"),
	}
}

type ReconcileWebhookCertificates struct {
	ctx                context.Context
	client             client.Client
	namespace          string
	logger             logr.Logger
	cancelBootstrapper context.CancelFunc
}

func (r *ReconcileWebhookCertificates) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	r.logger.Info("reconciling webhook certificates",
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

	certs := Certs{
		Log:     r.logger,
		Domain:  r.getDomain(),
		SrcData: secret.Data,
		Now:     time.Now(),
	}
	if err = certs.ValidateCerts(); err != nil {
		return reconcile.Result{}, err
	}

	mutatingWebhookConfiguration, err := r.getMutatingWebhookConfiguration(ctx)
	if err != nil {
		return reconcile.Result{}, err
	}
	validatingWebhookConfiguration, err := r.getValidatingWebhookConfiguration(ctx)
	if err != nil {
		return reconcile.Result{}, err
	}

	isWebhookCertificateValid := r.checkMutatingWebhookConfigurations(
		mutatingWebhookConfiguration, validatingWebhookConfiguration, certs.Data[RootCert])

	isSecretOutdated := false
	if !reflect.DeepEqual(certs.Data, secret.Data) {
		// certificate needs to be updated
		secret.Data = certs.Data
		isSecretOutdated = true
	} else if isWebhookCertificateValid {
		r.logger.Info("secret for certificates up to date, skipping update")
		r.cancelBootstrapper()
		return reconcile.Result{RequeueAfter: SuccessDuration}, nil
	}

	if isSecretOutdated {
		err = r.createOrUpdateSecret(ctx, secret, createSecret)
		if err != nil {
			return reconcile.Result{}, err
		}
	}
	err = r.updateWebhookConfigurations(ctx, secret, mutatingWebhookConfiguration, validatingWebhookConfiguration)
	if err != nil {
		return reconcile.Result{}, err
	}

	r.cancelBootstrapper()
	return reconcile.Result{RequeueAfter: SuccessDuration}, nil
}

func (r *ReconcileWebhookCertificates) createOrUpdateSecret(ctx context.Context, secret *corev1.Secret, createSecret bool) error {
	if createSecret {
		r.logger.Info("creating certificates secret")
		err := r.client.Create(ctx, secret)
		if err != nil {
			return err
		}
		r.logger.Info("created certificates secret")
	} else {
		r.logger.Info("updating certificates secret")
		err := r.client.Update(ctx, secret)
		if err != nil {
			return err
		}
		r.logger.Info("updated certificates secret")
	}
	return nil
}

func (r *ReconcileWebhookCertificates) updateWebhookConfigurations(ctx context.Context, secret *corev1.Secret,
	mutatingWebhookConfiguration *admissionregistrationv1.MutatingWebhookConfiguration,
	validatingWebhookConfiguration *admissionregistrationv1.ValidatingWebhookConfiguration) error {

	// update certificates for webhook configurations
	r.logger.Info("saving certificates into webhook configurations")
	for i := range mutatingWebhookConfiguration.Webhooks {
		if err := r.updateConfiguration(&mutatingWebhookConfiguration.Webhooks[i].ClientConfig, secret); err != nil {
			return err
		}
	}
	for i := range validatingWebhookConfiguration.Webhooks {
		if err := r.updateConfiguration(&validatingWebhookConfiguration.Webhooks[i].ClientConfig, secret); err != nil {
			return err
		}
	}

	if err := r.updateCRDConfiguration(ctx, secret); err != nil {
		return err
	}
	if err := r.client.Update(ctx, mutatingWebhookConfiguration); err != nil {
		return err
	}
	if err := r.client.Update(ctx, validatingWebhookConfiguration); err != nil {
		return err
	}
	r.logger.Info("saved certificates into webhook configurations")
	return nil
}

func (r *ReconcileWebhookCertificates) getMutatingWebhookConfiguration(ctx context.Context) (
	*admissionregistrationv1.MutatingWebhookConfiguration, error) {
	var mutatingWebhook admissionregistrationv1.MutatingWebhookConfiguration
	err := r.client.Get(ctx, client.ObjectKey{
		Name: webhook.DeploymentName,
	}, &mutatingWebhook)
	if err != nil {
		return nil, err
	}

	if len(mutatingWebhook.Webhooks) <= 0 {
		return nil, errors.New("mutating webhook configuration has no registered webhooks")
	}
	return &mutatingWebhook, nil
}

func (r *ReconcileWebhookCertificates) getValidatingWebhookConfiguration(ctx context.Context) (
	*admissionregistrationv1.ValidatingWebhookConfiguration, error) {
	var mutatingWebhook admissionregistrationv1.ValidatingWebhookConfiguration
	err := r.client.Get(ctx, client.ObjectKey{
		Name: webhook.DeploymentName,
	}, &mutatingWebhook)
	if err != nil {
		return nil, err
	}

	if len(mutatingWebhook.Webhooks) <= 0 {
		return nil, errors.New("validating webhook configuration has no registered webhooks")
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
	return fmt.Sprintf("%s%s", webhook.DeploymentName, secretPostfix)
}

func (r *ReconcileWebhookCertificates) getDomain() string {
	return fmt.Sprintf("%s.%s.svc", webhook.DeploymentName, r.namespace)
}

// checkMutatingWebhookConfigurations checks certificates exist and are valid
func (r *ReconcileWebhookCertificates) checkMutatingWebhookConfigurations(
	mutatingWebhookConfiguration *admissionregistrationv1.MutatingWebhookConfiguration,
	validatingWebhookConfiguration *admissionregistrationv1.ValidatingWebhookConfiguration, expectedCert []byte) bool {

	for _, mutatingWebhook := range mutatingWebhookConfiguration.Webhooks {
		webhookCert := mutatingWebhook.ClientConfig.CABundle
		if len(webhookCert) == 0 || !bytes.Equal(webhookCert, expectedCert) {
			return false
		}
	}

	for _, validatingWebhook := range validatingWebhookConfiguration.Webhooks {
		webhookCert := validatingWebhook.ClientConfig.CABundle
		if len(webhookCert) == 0 || !bytes.Equal(webhookCert, expectedCert) {
			return false
		}
	}
	return true
}

func (r *ReconcileWebhookCertificates) updateConfiguration(
	webhookConfiguration *admissionregistrationv1.WebhookClientConfig, secret *corev1.Secret) error {
	data, hasData := secret.Data[RootCert]
	if !hasData {
		return errors.New(errorCertificatesSecretEmpty)
	}

	if oldData, hasOldData := secret.Data[RootCertOld]; hasOldData {
		data = append(data, oldData...)
	}

	if webhookConfiguration != nil {
		webhookConfiguration.CABundle = data
	}
	return nil
}

func (r *ReconcileWebhookCertificates) updateCRDConfiguration(ctx context.Context, secret *corev1.Secret) error {

	var crd apiv1.CustomResourceDefinition
	if err := r.client.Get(ctx, types.NamespacedName{Name: crdName}, &crd); err != nil {
		return err
	}

	if !hasConversionWebhook(crd) {
		r.logger.Info("No conversion webhook config, no cert will be provided")
		return nil
	}

	data, hasData := secret.Data[RootCert]
	if !hasData {
		return errors.New(errorCertificatesSecretEmpty)
	}

	if oldData, hasOldData := secret.Data[RootCertOld]; hasOldData {
		data = append(data, oldData...)
	}

	// update crd
	crd.Spec.Conversion.Webhook.ClientConfig.CABundle = data
	if err := r.client.Update(ctx, &crd); err != nil {
		return err
	}
	return nil
}

func hasConversionWebhook(crd apiv1.CustomResourceDefinition) bool {
	return crd.Spec.Conversion != nil && crd.Spec.Conversion.Webhook != nil && crd.Spec.Conversion.Webhook.ClientConfig != nil
}
