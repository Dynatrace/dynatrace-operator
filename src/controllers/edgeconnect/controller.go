package edgeconnect

import (
	"context"
	"time"

	edgeconnectv1alpha1 "github.com/Dynatrace/dynatrace-operator/src/api/v1alpha1/edgeconnect"
	"github.com/Dynatrace/dynatrace-operator/src/timeprovider"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
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
		return reconcile.Result{RequeueAfter: errorUpdateInterval}, err
	} else if edgeConnect == nil {
		return reconcile.Result{RequeueAfter: errorUpdateInterval}, nil
	}

	err = controller.updateEdgeConnectStatus(ctx, edgeConnect)
	if err != nil {
		log.Error(errors.WithStack(err), "updating status of EdgeConnect failed", "name", request.Name, "namespace", request.Namespace)
		return reconcile.Result{RequeueAfter: errorUpdateInterval}, err
	}

	log.Info("reconciling EdgeConnect done", "name", request.Name, "namespace", request.Namespace)

	return reconcile.Result{RequeueAfter: defaultUpdateInterval}, nil
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
