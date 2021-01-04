package bootstrapper

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"time"

	"github.com/Dynatrace/dynatrace-operator/webhook"
	"github.com/go-logr/logr"
	admissionregistrationv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// time between consecutive queries for a new pod to get ready
const (
	webhookName = "dynatrace-oneagent-webhook"
	certsDir    = "/mnt/webhook-certs"
)

// AddToManager creates a new OneAgent Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func AddToManager(mgr manager.Manager, ns string) error {

	return add(mgr, &ReconcileWebhook{
		client:    mgr.GetClient(),
		scheme:    mgr.GetScheme(),
		namespace: ns,
		logger:    log.Log.WithName("webhook.controller"),
		certsDir:  certsDir,
	})
}

// add adds a new OneAgentController to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r *ReconcileWebhook) error {
	// Create a new controller
	c, err := controller.New("webhook-bootstrapper-controller", mgr, controller.Options{Reconciler: r})
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

		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()

		var svc corev1.Service
		err := r.client.Get(context.TODO(), client.ObjectKey{Name: webhookName, Namespace: r.namespace}, &svc)
		if err != nil {
			r.logger.Error(err, "Could not get webhook service")
		}

		ch <- event.GenericEvent{
			Object: &svc,
		}

		for range ticker.C {
			err = r.client.Get(context.TODO(), client.ObjectKey{Name: webhookName, Namespace: r.namespace}, &svc)
			if err != nil {
				r.logger.Error(err, "Could not get webhook service")
			}
			ch <- event.GenericEvent{
				Object: &svc,
			}
		}
	}()

	if err = mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		r.logger.Error(err, "could not start health endpoint for operator")
	}

	if err = mgr.AddReadyzCheck("healthz", healthz.Ping); err != nil {
		r.logger.Error(err, "could not start ready endpoint for operator")
	}

	return nil
}

// ReconcileWebhook reconciles the webhook
type ReconcileWebhook struct {
	client    client.Client
	scheme    *runtime.Scheme
	logger    logr.Logger
	namespace string
	certsDir  string
	now       time.Time
}

// Reconcile reads that state of the cluster for a OneAgent object and makes changes based on the state read
// and what is in the OneAgent.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileWebhook) Reconcile(context context.Context, request reconcile.Request) (reconcile.Result, error) {
	r.logger.Info("reconciling webhook", "namespace", request.Namespace, "name", request.Name)

	rootCerts, err := r.reconcileCerts(context, r.logger)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to reconcile certificates: %w", err)
	}

	if err := r.reconcileService(context, r.logger); err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to reconcile service: %w", err)
	}

	if err := r.reconcileWebhookConfig(context, r.logger, rootCerts); err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to reconcile webhook configuration: %w", err)
	}

	return reconcile.Result{}, nil
}

func (r *ReconcileWebhook) reconcileService(ctx context.Context, log logr.Logger) error {
	log.Info("Reconciling Service...")

	expected := corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      webhookName,
			Namespace: r.namespace,
			Labels: map[string]string{
				"dynatrace.com/operator":                    "oneagent",
				"internal.oneagent.dynatrace.com/component": "webhook",
			},
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"internal.oneagent.dynatrace.com/component": "webhook",
				"internal.oneagent.dynatrace.com/app":       "webhook",
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

func (r *ReconcileWebhook) reconcileCerts(ctx context.Context, log logr.Logger) ([]byte, error) {
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
		now:     r.now,
	}

	if err := cs.ValidateCerts(); err != nil {
		return nil, err
	}

	if newSecret {
		log.Info("Creating certificates secret...")
		err = r.client.Create(ctx, &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: webhook.SecretCertsName, Namespace: r.namespace},
			Data:       cs.Data,
		})
	} else if !reflect.DeepEqual(cs.Data, secret.Data) {
		log.Info("Updating certificates secret...")
		secret.Data = cs.Data
		err = r.client.Update(ctx, &secret)
	}

	if err != nil {
		return nil, err
	}

	for _, key := range []string{"tls.crt", "tls.key"} {
		f := filepath.Join(r.certsDir, key)

		data, err := ioutil.ReadFile(f)
		if err != nil && !os.IsNotExist(err) {
			return nil, err
		}

		if os.IsNotExist(err) || !bytes.Equal(data, cs.Data[key]) {
			if err := ioutil.WriteFile(f, cs.Data[key], 0666); err != nil {
				return nil, err
			}
		}
	}

	return cs.Data["ca.crt"], nil
}

func (r *ReconcileWebhook) reconcileWebhookConfig(ctx context.Context, log logr.Logger, rootCerts []byte) error {
	log.Info("Reconciling MutatingWebhookConfiguration...")

	scope := admissionregistrationv1beta1.NamespacedScope
	path := "/inject"
	webhookConfiguration := &admissionregistrationv1beta1.MutatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name: webhookName,
			Labels: map[string]string{
				"dynatrace.com/operator":                    "oneagent",
				"internal.oneagent.dynatrace.com/component": "webhook",
			},
		},
		Webhooks: []admissionregistrationv1beta1.MutatingWebhook{{
			Name:                    "webhook.oneagent.dynatrace.com",
			AdmissionReviewVersions: []string{"v1beta1"},
			Rules: []admissionregistrationv1beta1.RuleWithOperations{{
				Operations: []admissionregistrationv1beta1.OperationType{admissionregistrationv1beta1.Create},
				Rule: admissionregistrationv1beta1.Rule{
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
			ClientConfig: admissionregistrationv1beta1.WebhookClientConfig{
				Service: &admissionregistrationv1beta1.ServiceReference{
					Name:      webhookName,
					Namespace: r.namespace,
					Path:      &path,
				},
				CABundle: rootCerts,
			},
		}},
	}

	var cfg admissionregistrationv1beta1.MutatingWebhookConfiguration
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
