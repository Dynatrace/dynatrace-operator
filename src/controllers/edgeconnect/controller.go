package edgeconnect

import (
	"context"
	"os"
	"time"

	edgeconnectv1alpha1 "github.com/Dynatrace/dynatrace-operator/src/api/v1alpha1/edgeconnect"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/version"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	"github.com/Dynatrace/dynatrace-operator/src/kubesystem"
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
	defaultUpdateInterval = 30 * time.Minute
)

// Controller reconciles an EdgeConnect object
type Controller struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the api-server
	client            client.Client
	apiReader         client.Reader
	scheme            *runtime.Scheme
	config            *rest.Config
	operatorNamespace string
	clusterID         string
	versionProvider   version.ImageVersionFunc
}

func Add(mgr manager.Manager, _ string) error {
	kubeSysUID, err := kubesystem.GetUID(mgr.GetAPIReader())
	if err != nil {
		return errors.WithStack(err)
	}
	return NewController(mgr, string(kubeSysUID)).SetupWithManager(mgr)
}

// NewController returns a new ReconcileDynaKube
func NewController(mgr manager.Manager, clusterID string) *Controller {
	return NewEdgeConnectController(mgr.GetClient(), mgr.GetAPIReader(), mgr.GetScheme(), mgr.GetConfig(), clusterID)
}

func NewEdgeConnectController(kubeClient client.Client, apiReader client.Reader, scheme *runtime.Scheme, config *rest.Config, clusterID string) *Controller { //nolint:revive
	return &Controller{
		client:            kubeClient,
		apiReader:         apiReader,
		scheme:            scheme,
		config:            config,
		operatorNamespace: os.Getenv(kubeobjects.EnvPodNamespace),
		clusterID:         clusterID,
		versionProvider:   version.GetImageVersion,
	}
}

func (controller *Controller) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&edgeconnectv1alpha1.EdgeConnect{}).
		Owns(&appsv1.Deployment{}).
		Complete(controller)
}

func (controller *Controller) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	requeueAfter := defaultUpdateInterval

	log.Info("reconciling EdgeConnect", "name", request.Name, "namespace", request.Namespace)

	edgeConnect, err := controller.getEdgeConnect(ctx, request.Name, request.Namespace)
	if err != nil {
		log.Error(errors.WithStack(err), "reconciliation of edgeconnect failed", "name", request.Name, "namespace", request.Namespace)
		return reconcile.Result{}, err
	} else if edgeConnect == nil {
		return reconcile.Result{}, nil
	}

	controller.updateEdgeConnectStatus(ctx, edgeConnect)

	log.Info("reconciling EdgeConnect done", "name", request.Name, "namespace", request.Namespace)

	return reconcile.Result{RequeueAfter: requeueAfter}, nil
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
	log.Info("updateEdgeConnectStatus", "name", edgeConnect.Name, "namespace", edgeConnect.Namespace)
	edgeConnect.Status.UpdatedTimestamp = metav1.Now()

	err := controller.client.Status().Update(ctx, edgeConnect)
	if k8serrors.IsConflict(err) {
		log.Info("could not update edgeconnect due to conflict", "name", edgeConnect.Name)
		return errors.WithStack(err)
	} else if err != nil {
		log.Error(err, "updateEdgeConnectStatus", "name", edgeConnect.Name, "namespace", edgeConnect.Namespace)
		return errors.WithStack(err)
	}
	log.Info("edgeconnect status updated", "name", edgeConnect.Name, "timestamp", edgeConnect.Status.UpdatedTimestamp)
	return nil
}
