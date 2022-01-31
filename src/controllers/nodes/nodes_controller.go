package nodes

import (
	"context"
	"os"
	"time"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube"
	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func Add(mgr manager.Manager, _ string) error {
	return NewReconciler(mgr).SetupWithManager(mgr)
}

func (r *ReconcileNode) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Node{}).
		WithEventFilter(nodeDeletionPredicate(r)).
		Complete(r)
}
func nodeDeletionPredicate(controller *ReconcileNode) predicate.Predicate {
	return predicate.Funcs{
		DeleteFunc: func(deleteEvent event.DeleteEvent) bool {
			controller.reconcileNodeDeletion(deleteEvent.Object.GetName())
			return false
		},
	}
}

// blank assignment to verify that ReconcileNode implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileNode{}

// NewReconciler returns a new ReconcileDynaKube
func NewReconciler(mgr manager.Manager) *ReconcileNode {
	return &ReconcileNode{
		client:       mgr.GetClient(),
		scheme:       mgr.GetScheme(),
		dtClientFunc: dynakube.BuildDynatraceClient,
		runLocal:     os.Getenv("RUN_LOCAL") == "true",
		podNamespace: os.Getenv("POD_NAMESPACE"),
	}
}

type ReconcileNode struct {
	client       client.Client
	scheme       *runtime.Scheme
	dtClientFunc dynakube.DynatraceClientFunc
	runLocal     bool
	podNamespace string
}

type CachedNodeInfo struct {
	cachedNode CacheEntry
	nodeCache  *Cache
	nodeName   string
}

func (r *ReconcileNode) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	nodeName := request.NamespacedName.Name
	dynakube, err := r.determineDynakubeForNode(nodeName)
	if err != nil {
		log.Error(err, "error while getting Dynakube for Node")
		return reconcile.Result{}, err
	}

	nodeCache, err := r.getCache()
	if err != nil {
		return reconcile.Result{}, err
	}

	var node corev1.Node
	if err := r.client.Get(ctx, client.ObjectKey{Name: nodeName}, &node); err != nil {
		// handle deletion of Node
		if k8serrors.IsNotFound(err) {
			log.Info("node was not found in cluster", "node", nodeName)
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	// Node is found in the cluster, add or update to cache
	if dynakube != nil {
		var ipAddress = dynakube.Status.OneAgent.Instances[nodeName].IPAddress
		cacheEntry := CacheEntry{
			Instance:  dynakube.Name,
			IPAddress: ipAddress,
			LastSeen:  time.Now().UTC(),
		}

		if cached, err := nodeCache.Get(nodeName); err == nil {
			cacheEntry.LastMarkedForTermination = cached.LastMarkedForTermination
		}

		if err := nodeCache.Set(nodeName, cacheEntry); err != nil {
			return reconcile.Result{}, err
		}

		//Handle unschedulable Nodes, if they have a OneAgent instance
		if r.isUnschedulable(&node) {
			cachedNodeData := CachedNodeInfo{
				cachedNode: cacheEntry,
				nodeCache:  nodeCache,
				nodeName:   nodeName,
			}

			if err := r.markForTermination(dynakube, cachedNodeData); err != nil {
				log.Error(err, "unschedulable node failed to mark for termination", "node", nodeName)
				return reconcile.Result{}, err
			}
		}
	}

	// check node cache for outdated nodes and remove them, to keep cache clean
	if nodeCache.IsCacheOutdated() {
		if err := r.handleOutdatedCache(nodeCache); err != nil {
			return reconcile.Result{}, err
		}
		nodeCache.UpdateTimestamp()
	}
	return reconcile.Result{}, r.updateCache(nodeCache, ctx)
}

func (r *ReconcileNode) reconcileNodeDeletion(nodeName string) error {
	nodeCache, err := r.getCache()
	if err != nil {
		return err
	}

	dynakube, err := r.determineDynakubeForNode(nodeName)
	if err != nil {
		log.Error(err, "error while getting Dynakube for Node on deletion")
	}

	cachedNodeInfo, err := nodeCache.Get(nodeName)
	if err != nil {
		if err == ErrNotFound {
			// uncached node -> igonoring
			log.Error(err, "ignoring uncached node on deletion", "node", nodeName)
			return nil
		}
		log.Error(err, "error while getting cachedNode on deletion")
		return err
	}

	// Node is found in the cluster and in cache, send mark for termination, not found node is handled in err check
	if dynakube != nil {
		cachedNodeData := CachedNodeInfo{
			cachedNode: cachedNodeInfo,
			nodeCache:  nodeCache,
			nodeName:   nodeName,
		}

		if err := r.markForTermination(dynakube, cachedNodeData); err != nil {
			log.Error(err, "error while sending mark for termination for node:", "node", nodeName)
			return err
		}
	}

	nodeCache.Delete(nodeName)
	if err := r.updateCache(nodeCache, context.TODO()); err != nil {
		log.Error(err, "error while updating node cache after deletion:", "node", nodeName)
		return err
	}
	return nil
}

func (r *ReconcileNode) getCache() (*Cache, error) {
	var cm corev1.ConfigMap

	err := r.client.Get(context.TODO(), client.ObjectKey{Name: cacheName, Namespace: r.podNamespace}, &cm)
	if err == nil {
		return &Cache{Obj: &cm}, nil
	}

	if k8serrors.IsNotFound(err) {
		cm := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      cacheName,
				Namespace: r.podNamespace,
			},
			Data: map[string]string{},
		}

		if !r.runLocal { // If running locally, don't set the controller.
			deploy, err := kubeobjects.GetDeployment(r.client, os.Getenv("POD_NAME"), r.podNamespace)
			if err != nil {
				return nil, err
			}

			if err = controllerutil.SetControllerReference(deploy, cm, r.scheme); err != nil {
				return nil, err
			}
		}

		return &Cache{Create: true, Obj: cm}, nil
	}

	return nil, err
}

func (r *ReconcileNode) updateCache(nodeCache *Cache, ctx context.Context) error {
	if !nodeCache.Changed() {
		return nil
	}

	if nodeCache.Create {
		return r.client.Create(context.TODO(), nodeCache.Obj)
	}

	if err := r.client.Update(ctx, nodeCache.Obj); err != nil {
		return err
	}
	return nil
}

func (r *ReconcileNode) handleOutdatedCache(nodeCache *Cache) error {
	var nodeLst corev1.NodeList
	if err := r.client.List(context.TODO(), &nodeLst); err != nil {
		return err
	}

	for _, cachedNodeName := range nodeCache.Keys() {
		for _, clusterNode := range nodeLst.Items {
			if clusterNode.Name == cachedNodeName {
				cachedNodeInfo, err := nodeCache.Get(cachedNodeName)
				if err != nil {
					log.Error(err, "failed to get node", "node", cachedNodeName)
					return err
				}
				r.removeNodeFromCache(nodeCache, cachedNodeInfo, cachedNodeName)
				break
			}
		}
	}
	return nil
}

func (r *ReconcileNode) removeNodeFromCache(nodeCache *Cache, cachedNode CacheEntry, nodeName string) {
	if r.isNodeDeleteable(cachedNode) {
		nodeCache.Delete(nodeName)
	}
}

func (r *ReconcileNode) isNodeDeleteable(cachedNode CacheEntry) bool {
	if time.Now().UTC().Sub(cachedNode.LastSeen).Hours() > 1 {
		return true
	} else if cachedNode.IPAddress == "" {
		return true
	}
	return false
}

func (r *ReconcileNode) sendMarkedForTermination(dynakubeInstance *dynatracev1beta1.DynaKube, cachedNode CacheEntry) error {
	dtp, err := dynakube.NewDynatraceClientProperties(context.TODO(), r.client, *dynakubeInstance)
	if err != nil {
		log.Error(err, err.Error())
	}

	dtc, err := r.dtClientFunc(*dtp)
	if err != nil {
		return err
	}

	entityID, err := dtc.GetEntityIDForIP(cachedNode.IPAddress)
	if err != nil {
		log.Info("failed to send mark for termination event",
			"reason", "failed to determine entity id", "dynakube", dynakubeInstance.Name, "nodeIP", cachedNode.IPAddress, "cause", err)

		return err
	}

	ts := uint64(cachedNode.LastSeen.Add(-10*time.Minute).UnixNano()) / uint64(time.Millisecond)
	return dtc.SendEvent(&dtclient.EventData{
		EventType:     dtclient.MarkedForTerminationEvent,
		Source:        "Dynatrace Operator",
		Description:   "Kubernetes node cordoned. Node might be drained or terminated.",
		StartInMillis: ts,
		EndInMillis:   ts,
		AttachRules: dtclient.EventDataAttachRules{
			EntityIDs: []string{entityID},
		},
	})
}

func (r *ReconcileNode) markForTermination(dynakube *dynatracev1beta1.DynaKube, cachedNodeData CachedNodeInfo) error {
	if !r.isMarkableForTermination(&cachedNodeData.cachedNode) {
		return nil
	}

	if err := cachedNodeData.nodeCache.updateLastMarkedForTerminationTimestamp(cachedNodeData.cachedNode, cachedNodeData.nodeName); err != nil {
		return err
	}

	log.Info("sending mark for termination event to dynatrace server", "dynakube", dynakube.Name, "ip", cachedNodeData.cachedNode.IPAddress,
		"node", cachedNodeData.nodeName)

	return r.sendMarkedForTermination(dynakube, cachedNodeData.cachedNode)
}

func (r *ReconcileNode) isUnschedulable(node *corev1.Node) bool {
	return node.Spec.Unschedulable || r.hasUnschedulableTaint(node)
}

func (r *ReconcileNode) hasUnschedulableTaint(node *corev1.Node) bool {
	for _, taint := range node.Spec.Taints {
		for _, unschedulableTaint := range unschedulableTaints {
			if taint.Key == unschedulableTaint {
				return true
			}
		}
	}
	return false
}

// isMarkableForTermination checks if the timestamp from last mark is at least one hour old
func (r *ReconcileNode) isMarkableForTermination(nodeInfo *CacheEntry) bool {
	// If the last mark was an hour ago, mark again
	// Zero value for time.Time is 0001-01-01, so first mark is also executed
	lastMarked := nodeInfo.LastMarkedForTermination
	return lastMarked.UTC().Add(time.Hour).Before(time.Now().UTC())
}
