package edgeconnect

import (
	"context"
	"time"

	"github.com/Dynatrace/dynatrace-operator/src/api/status"
	edgeconnectv1alpha1 "github.com/Dynatrace/dynatrace-operator/src/api/v1alpha1/edgeconnect"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/edgeconnect/deployment"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/edgeconnect/version"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	"github.com/Dynatrace/dynatrace-operator/src/timeprovider"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	errorUpdateInterval   = 1 * time.Minute
	defaultUpdateInterval = 30 * time.Minute
)

// Controller reconciles an EdgeConnect object
type Controller struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the api-server
	client       client.Client
	apiReader    client.Reader
	scheme       *runtime.Scheme
	config       *rest.Config
	timeProvider *timeprovider.Provider
}

func Add(mgr manager.Manager, _ string) error {
	return NewController(mgr).SetupWithManager(mgr)
}

func NewController(mgr manager.Manager) *Controller {
	return &Controller{
		client:       mgr.GetClient(),
		apiReader:    mgr.GetAPIReader(),
		scheme:       mgr.GetScheme(),
		config:       mgr.GetConfig(),
		timeProvider: timeprovider.New(),
	}
}

func (controller *Controller) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&edgeconnectv1alpha1.EdgeConnect{}).
		Owns(&appsv1.Deployment{}).
		Complete(controller)
}

func (controller *Controller) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log.Info("reconciling EdgeConnect", "name", request.Name, "namespace", request.Namespace)

	edgeConnect, err := controller.getEdgeConnect(ctx, request.Name, request.Namespace)
	if err != nil {
		log.Error(errors.WithStack(err), "reconciliation of EdgeConnect failed", "name", request.Name, "namespace", request.Namespace)
		return reconcile.Result{}, err
	} else if edgeConnect == nil {
		return reconcile.Result{}, nil
	}

	log.Info("updating version info", "name", request.Name, "namespace", request.Namespace)
	versionReconciler := version.NewReconciler(edgeConnect, controller.apiReader, timeprovider.New())
	if err = versionReconciler.Reconcile(ctx); err != nil {
		log.Error(err, "reconciliation of EdgeConnect failed", "name", request.Name, "namespace", request.Namespace)
		return reconcile.Result{RequeueAfter: errorUpdateInterval}, nil
	}

	oldStatus := *edgeConnect.Status.DeepCopy()

	err = controller.reconcileEdgeConnect(edgeConnect)

	if err != nil {
		edgeConnect.Status.SetPhase(status.Error)
		log.Error(err, "error reconciling EdgeConnect", "namespace", edgeConnect.Namespace, "name", edgeConnect.Name)
	} else {
		edgeConnect.Status.SetPhase(status.Running)
	}
	err = controller.updateEdgeConnectStatus(ctx, edgeConnect)

	if isDifferentStatus, err := kubeobjects.IsDifferent(oldStatus, edgeConnect.Status); err != nil {
		log.Error(errors.WithStack(err), "failed to generate hash for the status section")
	} else if isDifferentStatus {
		log.Info("status changed, updating DynaKube")
		if errClient := controller.updateEdgeConnectStatus(ctx, edgeConnect); errClient != nil {
			return reconcile.Result{RequeueAfter: errorUpdateInterval}, errors.WithMessagef(errClient, "failed to update EdgeConnect after failure, original error: %s", err)
		}
	}

	log.Info("reconciling EdgeConnect done", "name", request.Name, "namespace", request.Namespace)

	return reconcile.Result{RequeueAfter: defaultUpdateInterval}, err
}

func (controller *Controller) getEdgeConnect(ctx context.Context, name, namespace string) (*edgeconnectv1alpha1.EdgeConnect, error) {
	edgeConnect := &edgeconnectv1alpha1.EdgeConnect{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
	err := controller.apiReader.Get(ctx, client.ObjectKey{Name: edgeConnect.Name, Namespace: edgeConnect.Namespace}, edgeConnect)

	if k8serrors.IsNotFound(err) {
		return nil, nil
	} else if err != nil {
		return nil, errors.WithStack(err)
	}
	return edgeConnect, nil
}

func (controller *Controller) updateEdgeConnectStatus(ctx context.Context, edgeConnect *edgeconnectv1alpha1.EdgeConnect) error {
	edgeConnect.Status.UpdatedTimestamp = *controller.timeProvider.Now()

	err := controller.client.Status().Update(ctx, edgeConnect)
	if k8serrors.IsConflict(err) {
		log.Info("could not update EdgeConnect status due to conflict", "name", edgeConnect.Name)
		return errors.WithStack(err)
	} else if err != nil {
		return errors.WithStack(err)
	}
	log.Info("EdgeConnect status updated", "name", edgeConnect.Name, "timestamp", edgeConnect.Status.UpdatedTimestamp)
	return nil
}

func (controller *Controller) reconcileEdgeConnect(edgeConnect *edgeconnectv1alpha1.EdgeConnect) error {
	desiredDeployment := deployment.New(edgeConnect)

	if err := controllerutil.SetControllerReference(edgeConnect, desiredDeployment, controller.scheme); err != nil {
		return errors.WithStack(err)
	}

	ddHash, err := kubeobjects.GenerateHash(desiredDeployment)
	if err != nil {
		return err
	}
	desiredDeployment.Annotations[kubeobjects.AnnotationHash] = ddHash

	_, err = kubeobjects.CreateOrUpdateDeployment(controller.client, log, desiredDeployment)

	if err != nil {
		log.Info("could not create or update deployment for EdgeConnect", "name", desiredDeployment.Name)
		return err
	}
	return nil
}
