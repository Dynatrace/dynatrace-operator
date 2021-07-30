package webhookcerts

import (
	"bytes"
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/Dynatrace/dynatrace-operator/webhook"
	"github.com/go-logr/logr"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	webhookName = "dynatrace-webhook"
)

func Add(mgr manager.Manager, ns string) error {
	return add(mgr, &ReconcileWebhookCertificates{
		client:    mgr.GetClient(),
		scheme:    mgr.GetScheme(),
		namespace: ns,
		logger:    log.Log.WithName("operator.webhook-certificates"),
	})
}

func add(mgr manager.Manager, r *ReconcileWebhookCertificates) error {
	c, err := controller.New("webhook-certificates-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	ch := make(chan event.GenericEvent, 10)

	if err = c.Watch(&source.Channel{Source: ch}, &handler.EnqueueRequestForObject{}); err != nil {
		return err
	}

	// Create artificial requests
	go func() {
		// Because of https://github.com/kubernetes-sigs/controller-runtime/issues/942, waiting
		// some time before inserting an element so that the Channel has time to initialize.
		time.Sleep(10 * time.Second)

		ticker := time.NewTicker(3 * time.Hour)
		defer ticker.Stop()

		ch <- event.GenericEvent{
			Object: &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: webhookName, Namespace: r.namespace}},
		}

		for range ticker.C {
			ch <- event.GenericEvent{
				Object: &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: webhookName, Namespace: r.namespace}},
			}
		}
	}()

	return nil
}

// ReconcileWebhookCertificates updates certificates secret for the webhooks
type ReconcileWebhookCertificates struct {
	client    client.Client
	scheme    *runtime.Scheme
	logger    logr.Logger
	namespace string
	now       time.Time
}

func (r *ReconcileWebhookCertificates) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	r.logger.Info("reconciling webhook certificates", "namespace", request.Namespace, "name", request.Name)

	rootCerts, err := r.reconcileCerts(ctx, r.logger)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to reconcile certificates: %w", err)
	}

	if err := r.reconcileService(ctx, r.logger); err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to reconcile service: %w", err)
	}

	if err := r.reconcileWebhookConfig(ctx, r.logger, rootCerts); err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to reconcile webhook configuration: %w", err)
	}

	return reconcile.Result{}, nil
}

func (r *ReconcileWebhookCertificates) reconcileService(ctx context.Context, log logr.Logger) error {
	log.Info("Reconciling Service...")

	expected := corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      webhookName,
			Namespace: r.namespace,
			Labels: map[string]string{
				"dynatrace.com/operator":           "oneagent",
				"internal.dynatrace.com/component": "webhook",
			},
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"internal.dynatrace.com/component": "webhook",
				"internal.dynatrace.com/app":       "webhook",
			},
			Ports: []corev1.ServicePort{{
				Protocol:   corev1.ProtocolTCP,
				Port:       443,
				TargetPort: intstr.FromString("server-port"),
			}},
		},
	}

	var svc corev1.Service

	err := r.client.Get(context.TODO(), client.ObjectKey{Name: webhookName, Namespace: r.namespace}, &svc)
	if k8serrors.IsNotFound(err) {
		log.Info("Service doesn't exist, creating...")
		if err = r.client.Create(ctx, &expected); err != nil {
			return err
		}
		return nil
	}

	return err
}

func (r *ReconcileWebhookCertificates) reconcileCerts(ctx context.Context, log logr.Logger) ([]byte, error) {
	log.Info("Reconciling certificates...")

	var newSecret bool
	var secret corev1.Secret

	err := r.client.Get(ctx, client.ObjectKey{Name: webhook.SecretCertsName, Namespace: r.namespace}, &secret)
	if k8serrors.IsNotFound(err) {
		newSecret = true
	} else if err != nil {
		return nil, err
	}

	cs := Certs{
		Log:     log,
		Domain:  fmt.Sprintf("%s.%s.svc", webhookName, r.namespace),
		SrcData: secret.Data,
		Now:     r.now,
	}

	if err := cs.ValidateCerts(); err != nil {
		return nil, err
	}

	if newSecret {
		log.Info("Creating certificates secret...")
		secret = corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: webhook.SecretCertsName, Namespace: r.namespace},
			Data:       cs.Data,
		}
		err = r.client.Create(ctx, &secret)
	} else if !reflect.DeepEqual(cs.Data, secret.Data) {
		log.Info("Updating certificates secret...")
		secret.Data = cs.Data
		err = r.client.Update(ctx, &secret)
	}
	if err != nil {
		return nil, err
	}
	return append(cs.Data["ca.crt"], cs.Data["ca.crt.old"]...), nil
}

func (r *ReconcileWebhookCertificates) reconcileWebhookConfig(ctx context.Context, log logr.Logger, rootCerts []byte) error {
	log.Info("Reconciling MutatingWebhookConfiguration...")

	path := "/inject"
	scope := admissionregistrationv1.NamespacedScope
	sideEffects := admissionregistrationv1.SideEffectClassNone
	webhookConfiguration := &admissionregistrationv1.MutatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name: webhookName,
			Labels: map[string]string{
				"dynatrace.com/operator":           "oneagent",
				"internal.dynatrace.com/component": "webhook",
			},
		},
		Webhooks: []admissionregistrationv1.MutatingWebhook{{
			Name:                    "webhook.dynatrace.com",
			AdmissionReviewVersions: []string{"v1"},
			Rules: []admissionregistrationv1.RuleWithOperations{{
				Operations: []admissionregistrationv1.OperationType{admissionregistrationv1.Create},
				Rule: admissionregistrationv1.Rule{
					APIGroups:   []string{""},
					APIVersions: []string{"v1"},
					Resources:   []string{"pods"},
					Scope:       &scope,
				},
			}},
			NamespaceSelector: &metav1.LabelSelector{
				MatchExpressions: []metav1.LabelSelectorRequirement{{
					Key:      webhook.LabelInstance,
					Operator: metav1.LabelSelectorOpExists,
				}},
			},
			ClientConfig: admissionregistrationv1.WebhookClientConfig{
				Service: &admissionregistrationv1.ServiceReference{
					Name:      webhookName,
					Namespace: r.namespace,
					Path:      &path,
				},
				CABundle: rootCerts,
			},
			SideEffects: &sideEffects,
		}},
	}

	var cfg admissionregistrationv1.MutatingWebhookConfiguration
	err := r.client.Get(context.TODO(), client.ObjectKey{Name: webhookName}, &cfg)
	if k8serrors.IsNotFound(err) {
		log.Info("MutatingWebhookConfiguration doesn't exist, creating...")

		if err = r.client.Create(ctx, webhookConfiguration); err != nil {
			return err
		}
		return nil
	}

	if err != nil {
		return err
	}

	if len(cfg.Webhooks) == 1 && bytes.Equal(cfg.Webhooks[0].ClientConfig.CABundle, rootCerts) {
		return nil
	}

	log.Info("MutatingWebhookConfiguration is outdated, updating...")
	cfg.Webhooks = webhookConfiguration.Webhooks
	return r.client.Update(ctx, &cfg)
}
