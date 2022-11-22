package certificates

import (
	"context"
	"reflect"
	"time"

	"github.com/Dynatrace/dynatrace-operator/src/eventfilter"
	"github.com/Dynatrace/dynatrace-operator/src/webhook"
	"github.com/pkg/errors"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
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
		Complete(newWebhookCertificateController(mgr, nil))
}

func AddBootstrap(mgr manager.Manager, ns string, cancelMgr context.CancelFunc) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&appsv1.Deployment{}).
		WithEventFilter(eventfilter.ForObjectNameAndNamespace(webhook.DeploymentName, ns)).
		Complete(newWebhookCertificateController(mgr, cancelMgr))
}

func newWebhookCertificateController(mgr manager.Manager, cancelMgr context.CancelFunc) *WebhookCertificateController {
	return &WebhookCertificateController{
		cancelMgrFunc: cancelMgr,
		client:        mgr.GetClient(),
		apiReader:     mgr.GetAPIReader(),
	}
}

type WebhookCertificateController struct {
	ctx           context.Context
	client        client.Client
	apiReader     client.Reader
	namespace     string
	cancelMgrFunc context.CancelFunc
}

func (controller *WebhookCertificateController) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log.Info("reconciling webhook certificates",
		"namespace", request.Namespace, "name", request.Name)
	controller.namespace = request.Namespace
	controller.ctx = ctx

	mutatingWebhookConfiguration, err := controller.getMutatingWebhookConfiguration()
	if err != nil {
		// Generation must not be skipped because webhook startup routine listens for the secret
		// See cmd/operator/manager.go and cmd/operator/watcher.go
		log.Info("could not find mutating webhook configuration, this is normal when deployed using OLM")
	}

	validatingWebhookConfiguration, err := controller.getValidatingWebhookConfiguration()
	if err != nil {
		// Generation must not be skipped because webhook startup routine listens for the secret
		// See cmd/operator/manager.go and cmd/operator/watcher.go
		log.Info("could not find validating webhook configuration, this is normal when deployed using OLM")
	}

	crd, err := controller.getCRDConfiguration()
	if err != nil {
		log.Info("could not find CRD configuration")
		return reconcile.Result{}, errors.WithStack(err)
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

	areMutatingWebhookConfigsValid := certSecret.areWebhookConfigsValid(mutatingWebhookConfigs)
	areValidatingWebhookConfigsValid := certSecret.areWebhookConfigsValid(validatingWebhookConfigs)
	isCRDConversionConfigValid := certSecret.isCRDConversionValid(crd.Spec.Conversion)

	if certSecret.isRecent() &&
		areMutatingWebhookConfigsValid &&
		areValidatingWebhookConfigsValid &&
		isCRDConversionConfigValid {
		log.Info("secret for certificates up to date, skipping update")
		controller.cancelMgr()
		return reconcile.Result{RequeueAfter: SuccessDuration}, nil
	}

	if err = certSecret.createOrUpdateIfNecessary(controller.ctx, controller.client); err != nil {
		return reconcile.Result{}, errors.WithStack(err)
	}

	bundle, err := certSecret.loadCombinedBundle()
	if err != nil {
		return reconcile.Result{}, errors.WithStack(err)
	}

	err = controller.updateClientConfigurations(bundle, mutatingWebhookConfigs, mutatingWebhookConfiguration)
	if err != nil {
		return reconcile.Result{}, errors.WithStack(err)
	}

	err = controller.updateClientConfigurations(bundle, validatingWebhookConfigs, validatingWebhookConfiguration)
	if err != nil {
		return reconcile.Result{}, errors.WithStack(err)
	}

	if err = controller.updateCRDConfiguration(bundle); err != nil {
		return reconcile.Result{}, errors.WithStack(err)
	}

	controller.cancelMgr()
	return reconcile.Result{RequeueAfter: SuccessDuration}, nil
}

func (controller *WebhookCertificateController) cancelMgr() {
	if controller.cancelMgrFunc != nil {
		log.Info("stopping manager after certificates creation")
		controller.cancelMgrFunc()
	}
}

func (controller *WebhookCertificateController) getMutatingWebhookConfiguration() (
	*admissionregistrationv1.MutatingWebhookConfiguration, error) {
	var mutatingWebhook admissionregistrationv1.MutatingWebhookConfiguration
	err := controller.apiReader.Get(controller.ctx, client.ObjectKey{
		Name: webhook.DeploymentName,
	}, &mutatingWebhook)
	if err != nil {
		return nil, err
	}

	if len(mutatingWebhook.Webhooks) == 0 {
		return nil, errors.New("mutating webhook configuration has no registered webhooks")
	}
	return &mutatingWebhook, nil
}

func (controller *WebhookCertificateController) getValidatingWebhookConfiguration() (
	*admissionregistrationv1.ValidatingWebhookConfiguration, error) {
	var mutatingWebhook admissionregistrationv1.ValidatingWebhookConfiguration
	err := controller.apiReader.Get(controller.ctx, client.ObjectKey{
		Name: webhook.DeploymentName,
	}, &mutatingWebhook)
	if err != nil {
		return nil, err
	}

	if len(mutatingWebhook.Webhooks) == 0 {
		return nil, errors.New("validating webhook configuration has no registered webhooks")
	}
	return &mutatingWebhook, nil
}

func (controller *WebhookCertificateController) getCRDConfiguration() (
	*apiv1.CustomResourceDefinition, error) {
	var crd apiv1.CustomResourceDefinition
	if err := controller.apiReader.Get(controller.ctx, types.NamespacedName{Name: crdName}, &crd); err != nil {
		return nil, err
	}

	return &crd, nil
}

func (controller *WebhookCertificateController) updateClientConfigurations(bundle []byte,
	webhookClientConfigs []*admissionregistrationv1.WebhookClientConfig, webhookConfig client.Object) error {
	if webhookConfig == nil || reflect.ValueOf(webhookConfig).IsNil() {
		return nil
	}

	for i := range webhookClientConfigs {
		webhookClientConfigs[i].CABundle = bundle
	}

	if err := controller.client.Update(controller.ctx, webhookConfig); err != nil {
		return err
	}
	return nil
}

func (controller *WebhookCertificateController) updateCRDConfiguration(bundle []byte) error {
	var crd apiv1.CustomResourceDefinition
	if err := controller.apiReader.Get(controller.ctx, types.NamespacedName{Name: crdName}, &crd); err != nil {
		return err
	}

	if !hasConversionWebhook(crd) {
		log.Info("no conversion webhook config, no cert will be provided")
		return nil
	}

	// update crd
	crd.Spec.Conversion.Webhook.ClientConfig.CABundle = bundle
	if err := controller.client.Update(controller.ctx, &crd); err != nil {
		return err
	}
	return nil
}

func hasConversionWebhook(crd apiv1.CustomResourceDefinition) bool {
	return crd.Spec.Conversion != nil && crd.Spec.Conversion.Webhook != nil && crd.Spec.Conversion.Webhook.ClientConfig != nil
}
