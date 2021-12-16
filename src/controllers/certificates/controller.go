package certificates

import (
	"bytes"
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/Dynatrace/dynatrace-operator/src/eventfilter"
	"github.com/Dynatrace/dynatrace-operator/src/webhook"
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
		Complete(newWebhookCertController(mgr, nil))
}

func AddBootstrap(mgr manager.Manager, ns string, cancelMgr context.CancelFunc) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&appsv1.Deployment{}).
		WithEventFilter(eventfilter.ForObjectNameAndNamespace(webhook.DeploymentName, ns)).
		Complete(newWebhookCertController(mgr, cancelMgr))
}

func newWebhookCertController(mgr manager.Manager, cancelMgr context.CancelFunc) *WebhookCertController {
	return &WebhookCertController{
		cancelMgrFunc: cancelMgr,
		client:        mgr.GetClient(),
		apiReader:     mgr.GetAPIReader(),
	}
}

type WebhookCertController struct {
	ctx           context.Context
	client        client.Client
	apiReader     client.Reader
	namespace     string
	cancelMgrFunc context.CancelFunc
}

func (controller *WebhookCertController) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log.Info("reconciling webhook certificates",
		"namespace", request.Namespace, "name", request.Name)
	controller.namespace = request.Namespace
	controller.ctx = ctx

	secret, err := controller.getSecret()
	if err != nil {
		return reconcile.Result{}, err
	}

	createSecret := false
	if secret == nil {
		createSecret = true
		secret = &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      controller.buildSecretName(),
				Namespace: controller.namespace,
			},
			Data: map[string][]byte{},
		}
	}

	certs := Certs{
		Domain:  controller.getDomain(),
		SrcData: secret.Data,
		Now:     time.Now(),
	}
	if err = certs.ValidateCerts(); err != nil {
		return reconcile.Result{}, err
	}

	mutatingWebhookConfiguration, err := controller.getMutatingWebhookConfiguration(ctx)
	if err != nil {
		return reconcile.Result{}, err
	}
	validatingWebhookConfiguration, err := controller.getValidatingWebhookConfiguration(ctx)
	if err != nil {
		return reconcile.Result{}, err
	}

	isWebhookCertificateValid := controller.checkMutatingWebhookConfigurations(
		mutatingWebhookConfiguration, validatingWebhookConfiguration, certs.Data[RootCert])

	isSecretOutdated := false
	if !reflect.DeepEqual(certs.Data, secret.Data) {
		// certificate needs to be updated
		secret.Data = certs.Data
		isSecretOutdated = true
	} else if isWebhookCertificateValid {
		log.Info("secret for certificates up to date, skipping update")
		controller.cancelMgr()
		return reconcile.Result{RequeueAfter: SuccessDuration}, nil
	}

	if isSecretOutdated {
		err = controller.createOrUpdateSecret(ctx, secret, createSecret)
		if err != nil {
			return reconcile.Result{}, err
		}
	}
	err = controller.updateWebhookConfigurations(ctx, secret, mutatingWebhookConfiguration, validatingWebhookConfiguration)
	if err != nil {
		return reconcile.Result{}, err
	}

	controller.cancelMgr()
	return reconcile.Result{RequeueAfter: SuccessDuration}, nil
}

func (controller *WebhookCertController) cancelMgr() {
	if controller.cancelMgrFunc != nil {
		log.Info("stopping manager after certificate creation")
		controller.cancelMgrFunc()
	}
}

func (controller *WebhookCertController) createOrUpdateSecret(ctx context.Context, secret *corev1.Secret, createSecret bool) error {
	if createSecret {
		log.Info("creating certificates secret")
		err := controller.client.Create(ctx, secret)
		if err != nil {
			return err
		}
		log.Info("created certificates secret")
	} else {
		log.Info("updating certificates secret")
		err := controller.client.Update(ctx, secret)
		if err != nil {
			return err
		}
		log.Info("updated certificates secret")
	}
	return nil
}

func (controller *WebhookCertController) updateWebhookConfigurations(ctx context.Context, secret *corev1.Secret,
	mutatingWebhookConfiguration *admissionregistrationv1.MutatingWebhookConfiguration,
	validatingWebhookConfiguration *admissionregistrationv1.ValidatingWebhookConfiguration) error {

	// update certificates for webhook configurations
	log.Info("saving certificates into webhook configurations")
	for i := range mutatingWebhookConfiguration.Webhooks {
		if err := controller.updateConfiguration(&mutatingWebhookConfiguration.Webhooks[i].ClientConfig, secret); err != nil {
			return err
		}
	}
	for i := range validatingWebhookConfiguration.Webhooks {
		if err := controller.updateConfiguration(&validatingWebhookConfiguration.Webhooks[i].ClientConfig, secret); err != nil {
			return err
		}
	}

	if err := controller.updateCRDConfiguration(ctx, secret); err != nil {
		return err
	}
	if err := controller.client.Update(ctx, mutatingWebhookConfiguration); err != nil {
		return err
	}
	if err := controller.client.Update(ctx, validatingWebhookConfiguration); err != nil {
		return err
	}
	log.Info("saved certificates into webhook configurations")
	return nil
}

func (controller *WebhookCertController) getMutatingWebhookConfiguration(ctx context.Context) (
	*admissionregistrationv1.MutatingWebhookConfiguration, error) {
	var mutatingWebhook admissionregistrationv1.MutatingWebhookConfiguration
	err := controller.apiReader.Get(ctx, client.ObjectKey{
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

func (controller *WebhookCertController) getValidatingWebhookConfiguration(ctx context.Context) (
	*admissionregistrationv1.ValidatingWebhookConfiguration, error) {
	var mutatingWebhook admissionregistrationv1.ValidatingWebhookConfiguration
	err := controller.apiReader.Get(ctx, client.ObjectKey{
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

func (controller *WebhookCertController) getSecret() (*corev1.Secret, error) {
	var oldSecret corev1.Secret
	err := controller.apiReader.Get(controller.ctx, client.ObjectKey{Name: controller.buildSecretName(), Namespace: controller.namespace}, &oldSecret)
	if k8serrors.IsNotFound(err) {
		return nil, nil
	}
	return &oldSecret, errors.WithStack(err)
}

func (controller *WebhookCertController) buildSecretName() string {
	return fmt.Sprintf("%s%s", webhook.DeploymentName, secretPostfix)
}

func (controller *WebhookCertController) getDomain() string {
	return fmt.Sprintf("%s.%s.svc", webhook.DeploymentName, controller.namespace)
}

// checkMutatingWebhookConfigurations checks certificates exist and are valid
func (controller *WebhookCertController) checkMutatingWebhookConfigurations(
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

func (controller *WebhookCertController) updateConfiguration(
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

func (controller *WebhookCertController) updateCRDConfiguration(ctx context.Context, secret *corev1.Secret) error {

	var crd apiv1.CustomResourceDefinition
	if err := controller.apiReader.Get(ctx, types.NamespacedName{Name: crdName}, &crd); err != nil {
		return err
	}

	if !hasConversionWebhook(crd) {
		log.Info("no conversion webhook config, no cert will be provided")
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
	if err := controller.client.Update(ctx, &crd); err != nil {
		return err
	}
	return nil
}

func hasConversionWebhook(crd apiv1.CustomResourceDefinition) bool {
	return crd.Spec.Conversion != nil && crd.Spec.Conversion.Webhook != nil && crd.Spec.Conversion.Webhook.ClientConfig != nil
}
