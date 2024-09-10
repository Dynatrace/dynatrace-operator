package nodes

import (
	"context"
	"os"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/scheme"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/dynatraceclient"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/token"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/deployment"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/env"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubesystem"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/timeprovider"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type Controller struct {
	client                 client.Client
	apiReader              client.Reader
	dynatraceClientBuilder dynatraceclient.Builder
	timeProvider           *timeprovider.Provider
	podNamespace           string
	runLocal               bool
}

type CachedNodeInfo struct {
	cachedNode CacheEntry
	nodeCache  *Cache
	nodeName   string
}

func Add(mgr manager.Manager, _ string) error {
	return NewController(mgr).SetupWithManager(mgr)
}

func (controller *Controller) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Node{}).
		Named("nodes-controller").
		Complete(controller)
}

func NewController(mgr manager.Manager) *Controller {
	return &Controller{
		client:                 mgr.GetClient(),
		apiReader:              mgr.GetAPIReader(),
		dynatraceClientBuilder: dynatraceclient.NewBuilder(mgr.GetAPIReader()),
		runLocal:               kubesystem.IsRunLocally(),
		podNamespace:           os.Getenv(env.PodNamespace),
		timeProvider:           timeprovider.New(),
	}
}

func (controller *Controller) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	nodeName := request.NamespacedName.Name
	dk, err := controller.determineDynakubeForNode(nodeName)
	log.Info("reconciling node name", "node", nodeName)

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
			// if there is no node it means it get deleted
			return reconcile.Result{}, controller.reconcileNodeDeletion(ctx, nodeName)
		}

		return reconcile.Result{}, err
	}

	// Node is found in the cluster, add or update to cache
	if dk != nil {
		ipAddress := dk.Status.OneAgent.Instances[nodeName].IPAddress
		cacheEntry := CacheEntry{
			Instance:  dk.Name,
			IPAddress: ipAddress,
			LastSeen:  controller.timeProvider.Now().UTC(),
		}

		if cached, err := nodeCache.Get(nodeName); err == nil {
			cacheEntry.LastMarkedForTermination = cached.LastMarkedForTermination
		}

		if err := nodeCache.Set(nodeName, cacheEntry); err != nil {
			return reconcile.Result{}, err
		}

		// Handle unschedulable Nodes, if they have a OneAgent instance
		if controller.isUnschedulable(&node) {
			cachedNodeData := CachedNodeInfo{
				cachedNode: cacheEntry,
				nodeCache:  nodeCache,
				nodeName:   nodeName,
			}

			if err := controller.markForTermination(ctx, dk, cachedNodeData); err != nil {
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

func (controller *Controller) reconcileNodeDeletion(ctx context.Context, nodeName string) error {
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
		if errors.Is(err, ErrNotFound) {
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

		if err := controller.markForTermination(ctx, dynakube, cachedNodeData); err != nil {
			return err
		}
	}

	nodeCache.Delete(nodeName)

	if err := controller.updateCache(ctx, nodeCache); err != nil {
		return err
	}

	return nil
}

func (controller *Controller) getCache(ctx context.Context) (*Cache, error) {
	var cm corev1.ConfigMap

	err := controller.apiReader.Get(ctx, client.ObjectKey{Name: cacheName, Namespace: controller.podNamespace}, &cm)
	if err == nil {
		return &Cache{Obj: &cm, timeProvider: controller.timeProvider}, nil
	}

	if k8serrors.IsNotFound(err) {
		cm := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      cacheName,
				Namespace: controller.podNamespace,
			},
			Data: map[string]string{},
		}
		// If running locally, don't set the controller.
		if !controller.runLocal {
			deploy, err := deployment.GetDeployment(controller.client, os.Getenv(env.PodName), controller.podNamespace)
			if err != nil {
				return nil, err
			}

			if err = controllerutil.SetControllerReference(deploy, cm, scheme.Scheme); err != nil {
				return nil, err
			}
		}

		return &Cache{Create: true, Obj: cm, timeProvider: controller.timeProvider}, nil
	}

	return nil, err
}

func (controller *Controller) updateCache(ctx context.Context, nodeCache *Cache) error {
	if !nodeCache.Changed() {
		return nil
	}

	if nodeCache.Create {
		return controller.client.Create(ctx, nodeCache.Obj)
	}

	if err := controller.client.Update(ctx, nodeCache.Obj); err != nil {
		return err
	}

	return nil
}

func (controller *Controller) handleOutdatedCache(ctx context.Context, nodeCache *Cache) error {
	var nodeLst corev1.NodeList
	if err := controller.client.List(ctx, &nodeLst); err != nil {
		return err
	}

	for _, cachedNodeName := range nodeCache.Keys() {
		cachedNodeInCluster := false

		for _, clusterNode := range nodeLst.Items {
			if clusterNode.Name == cachedNodeName {
				// We ignore errors because we always ask ONLY existing key from the cache,
				// because of the loop (range nodeCache.Keys()) in few lines early
				cachedNodeInfo, _ := nodeCache.Get(cachedNodeName)
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

func (controller *Controller) removeNodeFromCache(nodeCache *Cache, cachedNode CacheEntry, nodeName string) {
	if controller.isNodeDeletable(cachedNode) {
		nodeCache.Delete(nodeName)
	}
}

func (controller *Controller) isNodeDeletable(cachedNode CacheEntry) bool {
	if controller.timeProvider.Now().UTC().Sub(cachedNode.LastSeen).Hours() > 1 {
		return true
	} else if cachedNode.IPAddress == "" {
		return true
	}

	return false
}

func (controller *Controller) sendMarkedForTermination(ctx context.Context, dk *dynakube.DynaKube, cachedNode CacheEntry) error {
	tokenReader := token.NewReader(controller.apiReader, dk)

	tokens, err := tokenReader.ReadTokens(ctx)
	if err != nil {
		return err
	}

	dynatraceClient, err := controller.dynatraceClientBuilder.
		SetDynakube(*dk).
		SetTokens(tokens).
		Build()
	if err != nil {
		return err
	}

	entityID, err := dynatraceClient.GetEntityIDForIP(ctx, cachedNode.IPAddress)
	if err != nil {
		if errors.As(err, &dtclient.HostNotFoundErr{}) {
			log.Info("skipping to send mark for termination event", "dynakube", dk.Name, "nodeIP", cachedNode.IPAddress, "reason", err.Error())

			return nil
		}

		log.Info("failed to send mark for termination event",
			"reason", "failed to determine entity id", "dynakube", dk.Name, "nodeIP", cachedNode.IPAddress, "cause", err)

		return err
	}

	ts := uint64(cachedNode.LastSeen.Add(-10*time.Minute).UnixNano()) / uint64(time.Millisecond) //nolint:gosec

	return dynatraceClient.SendEvent(ctx, &dtclient.EventData{
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

func (controller *Controller) markForTermination(ctx context.Context, dk *dynakube.DynaKube, cachedNodeData CachedNodeInfo) error {
	if !controller.isMarkableForTermination(&cachedNodeData.cachedNode) {
		return nil
	}

	if err := cachedNodeData.nodeCache.updateLastMarkedForTerminationTimestamp(cachedNodeData.cachedNode, cachedNodeData.nodeName); err != nil {
		return err
	}

	log.Info("sending mark for termination event to dynatrace server", "dk", dk.Name, "ip", cachedNodeData.cachedNode.IPAddress,
		"node", cachedNodeData.nodeName)

	return controller.sendMarkedForTermination(ctx, dk, cachedNodeData.cachedNode)
}

func (controller *Controller) isUnschedulable(node *corev1.Node) bool {
	return node.Spec.Unschedulable || controller.hasUnschedulableTaint(node)
}

func (controller *Controller) hasUnschedulableTaint(node *corev1.Node) bool {
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
func (controller *Controller) isMarkableForTermination(nodeInfo *CacheEntry) bool {
	// If the last mark was an hour ago, mark again
	// Zero value for time.Time is 0001-01-01, so first mark is also executed
	lastMarked := nodeInfo.LastMarkedForTermination

	return lastMarked.UTC().Add(time.Hour).Before(controller.timeProvider.Now().UTC())
}
