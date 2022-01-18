package certificates

import (
	"context"
	"time"

	"github.com/Dynatrace/dynatrace-operator/src/eventfilter"
	"github.com/Dynatrace/dynatrace-operator/src/webhook"
	"github.com/pkg/errors"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apiv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
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

	mutatingWebhookConfiguration, err := controller.getMutatingWebhookConfiguration(ctx)
	if err != nil {
		// Generation must not be skipped because webhook startup routine listens for the secret
		// See cmd/operator/manager.go and cmd/operator/watcher.go
		log.Info("could not find mutating webhook configuration, this is normal when deployed using OLM")
	}

	validatingWebhookConfiguration, err := controller.getValidatingWebhookConfiguration(ctx)
	if err != nil {
		// Generation must not be skipped because webhook startup routine listens for the secret
		// See cmd/operator/manager.go and cmd/operator/watcher.go
		log.Info("could not find validating webhook configuration, this is normal when deployed using OLM")
	}

	certSecret := newCertificateSecret()

	err = certSecret.setSecretFromReader(controller.ctx, controller.apiReader, controller.namespace)
	if err != nil {
		return reconcile.Result{}, errors.WithStack(err)
	}

	err = certSecret.validateCertificates(controller.namespace)
	if err != nil {
		return reconcile.Result{}, errors.WithStack(err)
	}

	mutatingWebhookConfigs := getClientConfigsFromMutatingWebhook(mutatingWebhookConfiguration)
	validatingWebhookConfigs := getClientConfigsFromValidatingWebhook(validatingWebhookConfiguration)

	areMutatingWebhookConfigsValid := certSecret.areConfigsValid(mutatingWebhookConfigs)
	areValidatingWebhookConfigsValid := certSecret.areConfigsValid(validatingWebhookConfigs)

	if certSecret.isRecent() &&
		areMutatingWebhookConfigsValid &&
		areValidatingWebhookConfigsValid {
		log.Info("secret for certificates up to date, skipping update")
		controller.cancelMgr()
		return reconcile.Result{RequeueAfter: SuccessDuration}, nil
	}

	if err = certSecret.createOrUpdateIfNecessary(controller.ctx, controller.client); err != nil {
		return reconcile.Result{}, errors.WithStack(err)
	}

	if err = controller.updateCRDConfiguration(certSecret.secret); err != nil {
		return reconcile.Result{}, errors.WithStack(err)
	}

	err = certSecret.updateClientConfigurations(controller.ctx, controller.client, mutatingWebhookConfigs, mutatingWebhookConfiguration)
	if err != nil {
		return reconcile.Result{}, errors.WithStack(err)
	}

	err = certSecret.updateClientConfigurations(controller.ctx, controller.client, validatingWebhookConfigs, validatingWebhookConfiguration)
	if err != nil {
		return reconcile.Result{}, errors.WithStack(err)
	}

	controller.cancelMgr()
	return reconcile.Result{RequeueAfter: SuccessDuration}, nil
}

func (controller *WebhookCertController) cancelMgr() {
	if controller.cancelMgrFunc != nil {
		log.Info("stopping manager after certificates creation")
		controller.cancelMgrFunc()
	}
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

func (controller *WebhookCertController) updateCRDConfiguration(secret *corev1.Secret) error {

	var crd apiv1.CustomResourceDefinition
	if err := controller.apiReader.Get(controller.ctx, types.NamespacedName{Name: crdName}, &crd); err != nil {
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
	if err := controller.client.Update(controller.ctx, &crd); err != nil {
		return err
	}
	return nil
}

func hasConversionWebhook(crd apiv1.CustomResourceDefinition) bool {
	return crd.Spec.Conversion != nil && crd.Spec.Conversion.Webhook != nil && crd.Spec.Conversion.Webhook.ClientConfig != nil
}
