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

type NodesController struct {
	client       client.Client
	apiReader    client.Reader
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

func Add(mgr manager.Manager, _ string) error {
	return NewController(mgr).SetupWithManager(mgr)
}

func (controller *NodesController) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Node{}).
		WithEventFilter(nodeDeletionPredicate(controller)).
		Complete(controller)
}
func nodeDeletionPredicate(controller *NodesController) predicate.Predicate {
	return predicate.Funcs{
		DeleteFunc: func(deleteEvent event.DeleteEvent) bool {
			node := deleteEvent.Object.GetName()
			err := controller.reconcileNodeDeletion(context.TODO(), node)
			if err != nil {
				log.Error(err, "error while deleting node", "node", node)
			}
			return false
		},
	}
}

// NewReconciler returns a new ReconcileDynaKube
func NewController(mgr manager.Manager) *NodesController {
	return &NodesController{
		client:       mgr.GetClient(),
		apiReader:    mgr.GetAPIReader(),
		scheme:       mgr.GetScheme(),
		dtClientFunc: dynakube.BuildDynatraceClient,
		runLocal:     os.Getenv("RUN_LOCAL") == "true",
		podNamespace: os.Getenv("POD_NAMESPACE"),
	}
}

func (controller *NodesController) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	nodeName := request.NamespacedName.Name
	dynakube, err := controller.determineDynakubeForNode(nodeName)
	if err != nil {
		return reconcile.Result{}, err
	}

	nodeCache, err := controller.getCache(ctx)
	if err != nil {
		return reconcile.Result{}, err
	}

	var node corev1.Node
	if err := controller.apiReader.Get(ctx, client.ObjectKey{Name: nodeName}, &node); err != nil {
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
		if controller.isUnschedulable(&node) {
			cachedNodeData := CachedNodeInfo{
				cachedNode: cacheEntry,
				nodeCache:  nodeCache,
				nodeName:   nodeName,
			}

			if err := controller.markForTermination(dynakube, cachedNodeData); err != nil {
				return reconcile.Result{}, err
			}
		}
	}

	// check node cache for outdated nodes and remove them, to keep cache clean
	if nodeCache.IsCacheOutdated() {
		if err := controller.handleOutdatedCache(ctx, nodeCache); err != nil {
			return reconcile.Result{}, err
		}
		nodeCache.UpdateTimestamp()
	}
	return reconcile.Result{}, controller.updateCache(ctx, nodeCache)
}

func (controller *NodesController) reconcileNodeDeletion(ctx context.Context, nodeName string) error {
	nodeCache, err := controller.getCache(ctx)
	if err != nil {
		return err
	}

	dynakube, err := controller.determineDynakubeForNode(nodeName)
	if err != nil {
		return err
	}

	cachedNodeInfo, err := nodeCache.Get(nodeName)
	if err != nil {
		if err == ErrNotFound {
			// uncached node -> ignoring
			return nil
		}
		return err
	}

	if dynakube != nil {
		cachedNodeData := CachedNodeInfo{
			cachedNode: cachedNodeInfo,
			nodeCache:  nodeCache,
			nodeName:   nodeName,
		}

		if err := controller.markForTermination(dynakube, cachedNodeData); err != nil {
			return err
		}
	}

	nodeCache.Delete(nodeName)
	if err := controller.updateCache(ctx, nodeCache); err != nil {
		return err
	}
	return nil
}

func (controller *NodesController) getCache(ctx context.Context) (*Cache, error) {
	var cm corev1.ConfigMap

	err := controller.apiReader.Get(ctx, client.ObjectKey{Name: cacheName, Namespace: controller.podNamespace}, &cm)
	if err == nil {
		return &Cache{Obj: &cm}, nil
	}

	if k8serrors.IsNotFound(err) {
		cm := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      cacheName,
				Namespace: controller.podNamespace,
			},
			Data: map[string]string{},
		}

		if !controller.runLocal { // If running locally, don't set the controller.
			deploy, err := kubeobjects.GetDeployment(controller.client, os.Getenv("POD_NAME"), controller.podNamespace)
			if err != nil {
				return nil, err
			}

			if err = controllerutil.SetControllerReference(deploy, cm, controller.scheme); err != nil {
				return nil, err
			}
		}

		return &Cache{Create: true, Obj: cm}, nil
	}

	return nil, err
}

func (controller *NodesController) updateCache(ctx context.Context, nodeCache *Cache) error {
	if !nodeCache.Changed() {
		return nil
	}

	if nodeCache.Create {
		return controller.client.Create(context.TODO(), nodeCache.Obj)
	}

	if err := controller.client.Update(ctx, nodeCache.Obj); err != nil {
		return err
	}
	return nil
}

func (controller *NodesController) handleOutdatedCache(ctx context.Context, nodeCache *Cache) error {
	var nodeLst corev1.NodeList
	if err := controller.client.List(context.TODO(), &nodeLst); err != nil {
		return err
	}

	for _, cachedNodeName := range nodeCache.Keys() {
		cachedNodeInCluster := false
		for _, clusterNode := range nodeLst.Items {
			if clusterNode.Name == cachedNodeName {
				cachedNodeInfo, err := nodeCache.Get(cachedNodeName)
				if err != nil {
					log.Error(err, "failed to get node", "node", cachedNodeName)
					return err
				}
				cachedNodeInCluster = true
				// Check if node was seen less than an hour ago, otherwise do not remove from cache
				controller.removeNodeFromCache(nodeCache, cachedNodeInfo, cachedNodeName)
				break
			}
		}

		// if node is not in cluster -> probably deleted
		if !cachedNodeInCluster {
			log.Info("Removing missing cached node from cluster", "node", cachedNodeName)
			err := controller.reconcileNodeDeletion(ctx, cachedNodeName)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (controller *NodesController) removeNodeFromCache(nodeCache *Cache, cachedNode CacheEntry, nodeName string) {
	if controller.isNodeDeletable(cachedNode) {
		nodeCache.Delete(nodeName)
	}
}

func (controller *NodesController) isNodeDeletable(cachedNode CacheEntry) bool {
	if time.Now().UTC().Sub(cachedNode.LastSeen).Hours() > 1 {
		return true
	} else if cachedNode.IPAddress == "" {
		return true
	}
	return false
}

func (controller *NodesController) sendMarkedForTermination(dynakubeInstance *dynatracev1beta1.DynaKube, cachedNode CacheEntry) error {
	dtp, err := dynakube.NewDynatraceClientProperties(context.TODO(), controller.client, *dynakubeInstance)
	if err != nil {
		log.Error(err, err.Error())
	}

	dtc, err := controller.dtClientFunc(*dtp)
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

func (controller *NodesController) markForTermination(dynakube *dynatracev1beta1.DynaKube, cachedNodeData CachedNodeInfo) error {
	if !controller.isMarkableForTermination(&cachedNodeData.cachedNode) {
		return nil
	}

	if err := cachedNodeData.nodeCache.updateLastMarkedForTerminationTimestamp(cachedNodeData.cachedNode, cachedNodeData.nodeName); err != nil {
		return err
	}

	log.Info("sending mark for termination event to dynatrace server", "dynakube", dynakube.Name, "ip", cachedNodeData.cachedNode.IPAddress,
		"node", cachedNodeData.nodeName)

	return controller.sendMarkedForTermination(dynakube, cachedNodeData.cachedNode)
}

func (controller *NodesController) isUnschedulable(node *corev1.Node) bool {
	return node.Spec.Unschedulable || controller.hasUnschedulableTaint(node)
}

func (controller *NodesController) hasUnschedulableTaint(node *corev1.Node) bool {
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
func (controller *NodesController) isMarkableForTermination(nodeInfo *CacheEntry) bool {
	// If the last mark was an hour ago, mark again
	// Zero value for time.Time is 0001-01-01, so first mark is also executed
	lastMarked := nodeInfo.LastMarkedForTermination
	return lastMarked.UTC().Add(time.Hour).Before(time.Now().UTC())
}
