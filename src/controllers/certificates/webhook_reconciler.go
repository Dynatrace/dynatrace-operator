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
		Complete(newWebhookReconciler(mgr, nil))
}

func AddBootstrap(mgr manager.Manager, ns string, cancelMgr context.CancelFunc) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&appsv1.Deployment{}).
		WithEventFilter(eventfilter.ForObjectNameAndNamespace(webhook.DeploymentName, ns)).
		Complete(newWebhookReconciler(mgr, cancelMgr))
}

func newWebhookReconciler(mgr manager.Manager, cancelMgr context.CancelFunc) *ReconcileWebhookCertificates {
	return &ReconcileWebhookCertificates{
		cancelMgrFunc: cancelMgr,
		client:        mgr.GetClient(),
		apiReader:     mgr.GetAPIReader(),
	}
}

type ReconcileWebhookCertificates struct {
	ctx           context.Context
	client        client.Client
	apiReader     client.Reader
	namespace     string
	cancelMgrFunc context.CancelFunc
}

func (r *ReconcileWebhookCertificates) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log.Info("reconciling webhook certificates",
		"namespace", request.Namespace, "name", request.Name)
	r.namespace = request.Namespace
	r.ctx = ctx

	mutatingWebhookConfiguration, err := r.getMutatingWebhookConfiguration(ctx)
	if err != nil {
		log.Info("could not find mutating webhook configuration, this is normal when deployed using OLM")
	}

	validatingWebhookConfiguration, err := r.getValidatingWebhookConfiguration(ctx)
	if err != nil {
		log.Info("could not find validating webhook configuration, this is normal when deployed using OLM")
	}

	if mutatingWebhookConfiguration == nil && validatingWebhookConfiguration == nil {
		return reconcile.Result{RequeueAfter: SuccessDuration}, nil
	}

	certSecret := newCertificateSecret()

	err = certSecret.setSecretFromReader(r.ctx, r.apiReader, r.namespace)
	if err != nil {
		return reconcile.Result{}, errors.WithStack(err)
	}

	err = certSecret.validateCertificates(r.namespace)
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
		r.cancelMgr()
		return reconcile.Result{RequeueAfter: SuccessDuration}, nil
	}

	if err = certSecret.createOrUpdateIfNecessary(r.ctx, r.client); err != nil {
		return reconcile.Result{}, errors.WithStack(err)
	}

	if err = r.updateCRDConfiguration(certSecret.secret); err != nil {
		return reconcile.Result{}, errors.WithStack(err)
	}

	err = certSecret.updateClientConfigurations(r.ctx, r.client, mutatingWebhookConfigs, mutatingWebhookConfiguration)
	if err != nil {
		return reconcile.Result{}, errors.WithStack(err)
	}

	err = certSecret.updateClientConfigurations(r.ctx, r.client, validatingWebhookConfigs, validatingWebhookConfiguration)
	if err != nil {
		return reconcile.Result{}, errors.WithStack(err)
	}

	r.cancelMgr()
	return reconcile.Result{RequeueAfter: SuccessDuration}, nil
}

func (r *ReconcileWebhookCertificates) cancelMgr() {
	if r.cancelMgrFunc != nil {
		log.Info("stopping manager after certificates creation")
		r.cancelMgrFunc()
	}
}

func (r *ReconcileWebhookCertificates) getMutatingWebhookConfiguration(ctx context.Context) (
	*admissionregistrationv1.MutatingWebhookConfiguration, error) {
	var mutatingWebhook admissionregistrationv1.MutatingWebhookConfiguration
	err := r.apiReader.Get(ctx, client.ObjectKey{
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
	err := r.apiReader.Get(ctx, client.ObjectKey{
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

func (r *ReconcileWebhookCertificates) updateCRDConfiguration(secret *corev1.Secret) error {

	var crd apiv1.CustomResourceDefinition
	if err := r.apiReader.Get(r.ctx, types.NamespacedName{Name: crdName}, &crd); err != nil {
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
	if err := r.client.Update(r.ctx, &crd); err != nil {
		return err
	}
	return nil
}

func hasConversionWebhook(crd apiv1.CustomResourceDefinition) bool {
	return crd.Spec.Conversion != nil && crd.Spec.Conversion.Webhook != nil && crd.Spec.Conversion.Webhook.ClientConfig != nil
}
