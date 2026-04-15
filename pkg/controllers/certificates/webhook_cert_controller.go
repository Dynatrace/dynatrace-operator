package certificates

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/eventfilter"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/objects/k8scrd"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

const (
	SuccessDuration = 3 * time.Hour

	initReconcileInterval = 5 * time.Second

	secretPostfix = "-certs"
)

var errCertificatesSecretEmpty = errors.New("certificates secret is empty")

func InitReconcile(ctx context.Context, clt client.Client, namespace string) error {
	controller := newWebhookCertificateController(clt, clt)
	request := ctrl.Request{NamespacedName: types.NamespacedName{Name: webhook.DeploymentName, Namespace: namespace}}

	return wait.PollUntilContextCancel(ctx, initReconcileInterval, true, func(ctx context.Context) (bool, error) {
		if _, err := controller.Reconcile(ctx, request); err != nil {
			log.Error(err, "failed init reconcile")

			return false, nil
		}

		return true, nil
	})
}

func Add(mgr manager.Manager, namespace string) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&appsv1.Deployment{}).
		Named("webhook-cert-controller").
		WithEventFilter(eventfilter.ForObjectNameAndNamespace(webhook.DeploymentName, namespace)).
		Complete(newWebhookCertificateController(mgr.GetClient(), mgr.GetAPIReader()))
}

func newWebhookCertificateController(clt client.Client, apiReader client.Reader) *WebhookCertificateController {
	return &WebhookCertificateController{
		client:    clt,
		apiReader: apiReader,
	}
}

type WebhookCertificateController struct {
	client    client.Client
	apiReader client.Reader
}

func (controller *WebhookCertificateController) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	log.Info("reconciling webhook certificates", "namespace", request.Namespace, "name", request.Name)

	webhookDeployment := &appsv1.Deployment{}

	if err := controller.apiReader.Get(ctx, types.NamespacedName{Name: webhook.DeploymentName, Namespace: request.Namespace}, webhookDeployment); err != nil {
		if k8serrors.IsNotFound(err) {
			log.Info("no webhook deployment found, skipping webhook certificate generation")

			return ctrl.Result{}, nil
		}

		return ctrl.Result{}, fmt.Errorf("get webhook deployment: %w", err)
	}

	mutatingWebhookConfiguration, validatingWebhookConfiguration := controller.getWebhooksConfigurations(ctx)

	crd, err := controller.getCRD(ctx)
	if err != nil {
		return ctrl.Result{}, err
	}

	certSecret := newCertificateSecret(webhookDeployment)
	if err := certSecret.setSecretFromReader(ctx, controller.apiReader, webhookDeployment.Namespace); err != nil {
		return ctrl.Result{}, err
	}

	if err := certSecret.validateCertificates(ctx, webhookDeployment.Namespace); err != nil {
		return ctrl.Result{}, err
	}

	mutatingWebhookClientConfigs := getClientConfigsFromMutatingWebhook(mutatingWebhookConfiguration)
	validatingWebhookConfigConfigs := getClientConfigsFromValidatingWebhook(validatingWebhookConfiguration)

	if controller.isUpToDate(certSecret, mutatingWebhookClientConfigs, validatingWebhookConfigConfigs, crd) {
		log.Info("secret for certificates up to date, skipping update")

		return ctrl.Result{RequeueAfter: SuccessDuration}, nil
	}

	if err = certSecret.createOrUpdateIfNecessary(ctx, controller.client); err != nil {
		return ctrl.Result{}, err
	}

	bundle, err := certSecret.loadCombinedBundle()
	if err != nil {
		return ctrl.Result{}, err
	}

	err = controller.updateClientConfigurations(ctx, bundle, mutatingWebhookClientConfigs, mutatingWebhookConfiguration)
	if err != nil {
		return ctrl.Result{}, err
	}

	err = controller.updateClientConfigurations(ctx, bundle, validatingWebhookConfigConfigs, validatingWebhookConfiguration)
	if err != nil {
		return ctrl.Result{}, err
	}

	if err = controller.updateCRDConfiguration(ctx, k8scrd.DynaKubeName, bundle); err != nil {
		return ctrl.Result{}, err
	}

	if err = controller.updateCRDConfiguration(ctx, k8scrd.EdgeConnectName, bundle); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{RequeueAfter: SuccessDuration}, nil
}

func (controller *WebhookCertificateController) isUpToDate(certSecret *certificateSecret, mutatingWebhookClientConfigs []*admissionregistrationv1.WebhookClientConfig, validatingWebhookConfigConfigs []*admissionregistrationv1.WebhookClientConfig, crd *apiextensionsv1.CustomResourceDefinition) bool {
	areMutatingWebhookConfigsValid := certSecret.areWebhookConfigsValid(mutatingWebhookClientConfigs)
	areValidatingWebhookConfigsValid := certSecret.areWebhookConfigsValid(validatingWebhookConfigConfigs)
	isCRDConversionConfigValid := certSecret.isCRDConversionValid(crd)

	isUpToDate := certSecret.isRecent() &&
		areMutatingWebhookConfigsValid &&
		areValidatingWebhookConfigsValid &&
		isCRDConversionConfigValid

	return isUpToDate
}

func (controller *WebhookCertificateController) getWebhooksConfigurations(ctx context.Context) (*admissionregistrationv1.MutatingWebhookConfiguration, *admissionregistrationv1.ValidatingWebhookConfiguration) {
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

	return mutatingWebhookConfiguration, validatingWebhookConfiguration
}

func (controller *WebhookCertificateController) getMutatingWebhookConfiguration(ctx context.Context) (*admissionregistrationv1.MutatingWebhookConfiguration, error) {
	var mutatingWebhook admissionregistrationv1.MutatingWebhookConfiguration

	err := controller.apiReader.Get(ctx, client.ObjectKey{
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

func (controller *WebhookCertificateController) getValidatingWebhookConfiguration(ctx context.Context) (*admissionregistrationv1.ValidatingWebhookConfiguration, error) {
	var mutatingWebhook admissionregistrationv1.ValidatingWebhookConfiguration

	err := controller.apiReader.Get(ctx, client.ObjectKey{
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

func (controller *WebhookCertificateController) getCRD(ctx context.Context) (*apiextensionsv1.CustomResourceDefinition, error) {
	var crd apiextensionsv1.CustomResourceDefinition
	if err := controller.apiReader.Get(ctx, types.NamespacedName{Name: k8scrd.DynaKubeName}, &crd); err != nil {
		return nil, fmt.Errorf("get CRD: %w", err)
	}

	return &crd, nil
}

func (controller *WebhookCertificateController) updateClientConfigurations(ctx context.Context, bundle []byte,
	webhookClientConfigs []*admissionregistrationv1.WebhookClientConfig, webhookConfig client.Object,
) error {
	if webhookConfig == nil || reflect.ValueOf(webhookConfig).IsNil() {
		return nil
	}

	for i := range webhookClientConfigs {
		webhookClientConfigs[i].CABundle = bundle
	}

	if err := controller.client.Update(ctx, webhookConfig); err != nil {
		return fmt.Errorf("update webhook configuration %s: %w", webhookConfig.GetName(), err)
	}

	return nil
}

func (controller *WebhookCertificateController) updateCRDConfiguration(ctx context.Context, crdName string, bundle []byte) error {
	var crd apiextensionsv1.CustomResourceDefinition
	if err := controller.apiReader.Get(ctx, types.NamespacedName{Name: crdName}, &crd); err != nil {
		return fmt.Errorf("get CRD %s: %w", crdName, err)
	}

	if !hasConversionWebhook(crd) {
		log.Info("no conversion webhook config, no cert will be provided")

		return nil
	}

	crd.Spec.Conversion.Webhook.ClientConfig.CABundle = bundle
	if err := controller.client.Update(ctx, &crd); err != nil {
		return fmt.Errorf("update CRD %s: %w", crdName, err)
	}

	return nil
}

func hasConversionWebhook(crd apiextensionsv1.CustomResourceDefinition) bool {
	return crd.Spec.Conversion != nil && crd.Spec.Conversion.Webhook != nil && crd.Spec.Conversion.Webhook.ClientConfig != nil
}
