package crdcleanup

import (
	"context"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/eventfilter"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8senv"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/objects/k8scrd"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
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
		Named("crd-cleanup-controller").
		WithEventFilter(eventfilter.ForObjectNameAndNamespace(webhook.DeploymentName, ns)).
		Complete(newCRDCleanupController(mgr, cancelMgr))
}

func newCRDCleanupController(mgr manager.Manager, cancelMgr context.CancelFunc) *CRDCleanupController {
	return &CRDCleanupController{
		cancelMgrFunc: cancelMgr,
		client:        mgr.GetClient(),
		apiReader:     mgr.GetAPIReader(),
	}
}

type CRDCleanupController struct {
	client        client.Client
	apiReader     client.Reader
	cancelMgrFunc context.CancelFunc
}

func (controller *CRDCleanupController) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log.Info("reconciling CRD storage version cleanup",
		"namespace", request.Namespace, "name", request.Name)

	// There is a dependency on the webhook being ready to perform conversions
	webhookDeployment := appsv1.Deployment{}
	err := controller.apiReader.Get(ctx, types.NamespacedName{Name: webhook.DeploymentName, Namespace: request.Namespace}, &webhookDeployment)
	if k8serrors.IsNotFound(err) {
		log.Info("no webhook deployment found, skipping CRD cleanup")
		return reconcile.Result{}, nil
	} else if err != nil {
		return reconcile.Result{}, errors.WithStack(err)
	}

	if !isWebhookReady(&webhookDeployment) {
		log.Info("webhook deployment not ready yet, retrying CRD cleanup later")
		return reconcile.Result{RequeueAfter: RetryDuration}, nil
	}

	cleanupNeeded, err := controller.performCRDStorageVersionsCleanup(ctx)
	if err != nil {
		return reconcile.Result{}, errors.WithStack(err)
	}

	if !cleanupNeeded {
		log.Info("CRD cleanup not needed or already completed")
	} else {
		log.Info("CRD cleanup completed successfully")
	}

	controller.cancelMgr()

	return reconcile.Result{}, nil
}

func (controller *CRDCleanupController) cancelMgr() {
	if controller.cancelMgrFunc != nil {
		controller.cancelMgrFunc()
	}
}

// performCRDStorageVersionsCleanup performs cleanup of CRD storage versions before operator startup.
// It checks if the DynaKube CRD has multiple storage versions and if so, reads and writes
// all DynaKube instances to migrate them to the current storage version.
// Returns true if cleanup was performed, false if it wasn't needed.
func (controller *CRDCleanupController) performCRDStorageVersionsCleanup(ctx context.Context) (bool, error) {
	log.Info("starting CRD storage version cleanup")

	var crd apiextensionsv1.CustomResourceDefinition

	err := controller.apiReader.Get(ctx, types.NamespacedName{Name: k8scrd.DynaKubeName}, &crd)
	if err != nil {
		log.Info("failed to get DynaKube CRD, skipping cleanup", "error", err)
		return false, nil
	}

	if len(crd.Status.StoredVersions) == 0 {
		log.Info("DynaKube CRD has no storage versions, skipping cleanup")
		return false, nil
	}

	if len(crd.Status.StoredVersions) == 1 && crd.Status.StoredVersions[0] == latest.GroupVersion.Version {
		log.Info("DynaKube CRD has single, up-to-date storage version, no cleanup needed",
			"storedVersions", crd.Status.StoredVersions)
		return false, nil
	}

	log.Info("DynaKube CRD has multiple storage versions, performing migration",
		"storedVersions", crd.Status.StoredVersions,
		"currentVersion", latest.GroupVersion.Version)

	// List all DynaKube instances
	var dynakubeList dynakube.DynaKubeList

	err = controller.apiReader.List(ctx, &dynakubeList, &client.ListOptions{
		Namespace: k8senv.DefaultNamespace(),
	})
	if err != nil {
		return false, errors.Wrap(err, "failed to list DynaKube instances")
	}

	log.Info("migrating DynaKube instances to current storage version",
		"count", len(dynakubeList.Items),
		"targetVersion", latest.GroupVersion.Version)

	for i := range dynakubeList.Items {
		dk := &dynakubeList.Items[i]
		log.Info("migrating DynaKube instance",
			"name", dk.Name,
			"namespace", dk.Namespace)

		err = controller.client.Update(ctx, dk)
		if err != nil {
			return false, errors.Wrapf(err, "failed to update DynaKube %s/%s", dk.Namespace, dk.Name)
		}
	}

	crd.Status.StoredVersions = []string{latest.GroupVersion.Version}

	err = controller.client.Status().Update(ctx, &crd)
	if err != nil {
		return false, errors.Wrap(err, "failed to update DynaKube CRD status")
	}

	log.Info("successfully migrated all DynaKube instances to current storage version")

	return true, nil
}

func isWebhookReady(deployment *appsv1.Deployment) bool {
	if deployment.Spec.Replicas == nil {
		return false
	}
	return deployment.Status.ReadyReplicas >= *deployment.Spec.Replicas && *deployment.Spec.Replicas > 0
}
