package nodes

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func (r *ReconcileNodeDeletion) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Node{}).
		WithEventFilter(nodeDeletionPredicate()).
		Complete(r)
}

func nodeDeletionPredicate() predicate.Predicate {
	return predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			// Ignore updates to CR status in which case metadata.Generation does not change
			return false
		},
		CreateFunc: func(createEvent event.CreateEvent) bool {
			return false
		},
		GenericFunc: func(e event.GenericEvent) bool {
			return false
		},
	}
}

// blank assignment to verify that ReconcileNode implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileNodeDeletion{}

// NewDeletionReconciler  returns a new ReconcileDynaKube
func NewDeletionReconciler(mgr manager.Manager) *ReconcileNodeDeletion {
	return &ReconcileNodeDeletion{
		client:       mgr.GetClient(),
		scheme:       mgr.GetScheme(),
		logger:       log,
		dtClientFunc: dynakube.BuildDynatraceClient,
	}
}

type ReconcileNodeDeletion struct {
	client       client.Client
	scheme       *runtime.Scheme
	logger       logr.Logger
	dtClientFunc dynakube.DynatraceClientFunc
}

func (r *ReconcileNodeDeletion) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	nodeCache, err := getCache(r.client, r.scheme)
	if err != nil {
		return reconcile.Result{}, err
	}

	var nodeName = request.NamespacedName.Name
	dk, err := determineDynakubeForNode(nodeName, r.client)
	if err != nil {
		r.logger.Error(err, "error while getting Dynakube for Node")
		return reconcile.Result{}, err
	}

	cachedNodeInfo, err := nodeCache.Get(nodeName)
	if err != nil {
		if err == ErrNotFound {
			// uncached node -> igonoring
			return reconcile.Result{}, nil
		}

		r.logger.Error(err, "error while getting cachedNode on deletion")
		return reconcile.Result{}, err
	}

	// Node is found in the cluster and in cache, send mark for termination, not found node is handled in err check
	if dk != nil {
		err = markForTermination(MarkForTerminationOptions{
			nodeCache:    nodeCache,
			nodeName:     nodeName,
			cachedNode:   cachedNodeInfo,
			dynakube:     dk,
			client:       r.client,
			dtClientFunc: r.dtClientFunc,
		})

		if err != nil {
			return reconcile.Result{}, err
		}
	}

	nodeCache.Delete(nodeName)
	return reconcile.Result{}, nodeCache.updateCache(r.client, ctx)
}
