package namespace

import (
	"context"
	_ "embed"
	"time"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	"github.com/Dynatrace/dynatrace-operator/controllers/utils"
	"github.com/Dynatrace/dynatrace-operator/webhook"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

func Add(mgr manager.Manager, ns string) error {
	logger := log.Log.WithName("namespaces.controller")
	apmExists, err := utils.CheckIfOneAgentAPMExists(mgr.GetConfig())
	if err != nil {
		return err
	}
	if apmExists {
		logger.Info("OneAgentAPM object detected - Namespace reconciler disabled until the OneAgent Operator has been uninstalled")
		return nil
	}
	return add(mgr, &ReconcileNamespaces{
		client:    mgr.GetClient(),
		apiReader: mgr.GetAPIReader(),
		namespace: ns,
		logger:    logger,
	})
}

func add(mgr manager.Manager, r *ReconcileNamespaces) error {
	// Create a new controller
	c, err := controller.New("namespace-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource Namespaces
	err = c.Watch(&source.Kind{Type: &corev1.Namespace{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	return nil
}

type ReconcileNamespaces struct {
	client    client.Client
	apiReader client.Reader
	logger    logr.Logger
	namespace string
}

func (r *ReconcileNamespaces) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	targetNS := request.Name
	log := r.logger.WithValues("name", targetNS)
	log.Info("reconciling Namespace")

	var ns corev1.Namespace
	if err := r.client.Get(ctx, client.ObjectKey{Name: targetNS}, &ns); k8serrors.IsNotFound(err) {
		return reconcile.Result{}, nil
	} else if err != nil {
		return reconcile.Result{}, errors.WithMessage(err, "failed to query Namespace")
	}

	if ns.Labels == nil {
		return reconcile.Result{}, nil
	}

	dkName := ns.Labels[webhook.LabelInstance]
	if dkName == "" {
		return reconcile.Result{}, nil
	}

	var dk dynatracev1alpha1.DynaKube
	if err := r.client.Get(ctx, client.ObjectKey{Name: dkName, Namespace: r.namespace}, &dk); err != nil {
		return reconcile.Result{}, errors.WithMessage(err, "failed to query DynaKubes")
	}

	script, err := newScript(ctx, r.client, dk, ns)
	if err != nil {
		return reconcile.Result{}, errors.WithMessage(err, "failed to generate init script")
	}

	data, err := script.generate()
	if err != nil {
		return reconcile.Result{}, errors.WithMessage(err, "failed to generate script")
	}

	// The default cache-based Client doesn't support cross-namespace queries, unless configured to do so in Manager
	// Options. However, this is our only use-case for it, so using the non-cached Client instead.
	err = utils.CreateOrUpdateSecretIfNotExists(r.client, r.apiReader, webhook.SecretConfigName, targetNS, data, corev1.SecretTypeOpaque, log)
	if err != nil {
		return reconcile.Result{}, errors.WithStack(err)
	}

	return reconcile.Result{RequeueAfter: 5 * time.Minute}, nil
}
