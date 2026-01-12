package crdstoragemigration

import (
	"context"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/util/eventfilter"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8senv"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	RetryDuration = 10 * time.Second
)

func AddInit(mgr manager.Manager, ns string, cancelMgr context.CancelFunc) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&appsv1.Deployment{}).
		Named("crd-storage-migration-controller").
		WithEventFilter(eventfilter.ForObjectNameAndNamespace(webhook.DeploymentName, ns)).
		Complete(newCRDStorageMigrationController(mgr, cancelMgr))
}

func newCRDStorageMigrationController(mgr manager.Manager, cancelMgr context.CancelFunc) *Controller {
	return &Controller{
		cancelMgrFunc: cancelMgr,
		client:        mgr.GetClient(),
		apiReader:     mgr.GetAPIReader(),
	}
}

type Controller struct {
	client        client.Client
	apiReader     client.Reader
	cancelMgrFunc context.CancelFunc
}

func (controller *Controller) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log.Info("reconciling CRD storage version migration",
		"namespace", request.Namespace, "name", request.Name)

	// There is a dependency on the webhook being ready to perform conversions
	webhookDeployment := appsv1.Deployment{}

	err := controller.apiReader.Get(ctx, types.NamespacedName{Name: webhook.DeploymentName, Namespace: request.Namespace}, &webhookDeployment)
	if k8serrors.IsNotFound(err) {
		log.Info("no webhook deployment found, skipping CRD storage version migration")

		return reconcile.Result{}, nil
	} else if err != nil {
		return reconcile.Result{}, errors.WithStack(err)
	}

	if !isWebhookReady(&webhookDeployment) {
		log.Info("webhook deployment not ready yet, retrying CRD storage version migration later")

		return reconcile.Result{RequeueAfter: RetryDuration}, nil
	}

	migrationNeeded, err := controller.performCRDStorageVersionMigration(ctx)
	if err != nil {
		return reconcile.Result{}, errors.WithStack(err)
	}

	if !migrationNeeded {
		log.Info("CRD storage version migration not needed or already completed")
	} else {
		log.Info("CRD storage version migration completed successfully")
	}

	controller.cancelMgr()

	return reconcile.Result{}, nil
}

func (controller *Controller) cancelMgr() {
	if controller.cancelMgrFunc != nil {
		controller.cancelMgrFunc()
	}
}

func (controller *Controller) performCRDStorageVersionMigration(ctx context.Context) (bool, error) {
	return PerformCRDStorageVersionMigration(ctx, controller.client, controller.apiReader, k8senv.DefaultNamespace())
}

func isWebhookReady(deployment *appsv1.Deployment) bool {
	if deployment.Spec.Replicas == nil {
		return false
	}

	return deployment.Status.ReadyReplicas >= *deployment.Spec.Replicas && *deployment.Spec.Replicas > 0
}
