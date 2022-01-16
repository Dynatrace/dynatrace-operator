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

var unschedulableTaints = []string{"ToBeDeletedByClusterAutoscaler"}

func Add(mgr manager.Manager, _ string) error {
	// avoid miss-usage and add both Reconciler in Add() function
	err := NewDeletionReconciler(mgr).SetupWithManager(mgr)
	if err != nil {
		return err
	}
	return NewReconciler(mgr).SetupWithManager(mgr)
}

func (r *ReconcileNode) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Node{}).
		WithEventFilter(ignoreDeletionPredicate()).
		Complete(r)
}

func ignoreDeletionPredicate() predicate.Predicate {
	return predicate.Funcs{
		DeleteFunc: func(e event.DeleteEvent) bool {
			// handled in other controller.
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
	}
}

type ReconcileNode struct {
	client       client.Client
	scheme       *runtime.Scheme
	dtClientFunc dynakube.DynatraceClientFunc
}

type MarkForTerminationOptions struct {
	dynakube     *dynatracev1beta1.DynaKube
	nodeCache    *Cache
	cachedNode   CacheEntry
	nodeName     string
	client       client.Client
	dtClientFunc dynakube.DynatraceClientFunc
}

func (r *ReconcileNode) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	var node corev1.Node
	nodeName := request.NamespacedName.Name
	if err := r.client.Get(ctx, client.ObjectKey{Name: nodeName}, &node); err != nil {
		if k8serrors.IsNotFound(err) {
			log.Info("node was not found in cluster", "node", nodeName)
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	dk, err := determineDynakubeForNode(nodeName, r.client)
	if err != nil {
		log.Error(err, "error while getting Dynakube for Node")
		return reconcile.Result{}, err
	}

	nodeCache, err := getCache(r.client, r.scheme)
	if err != nil {
		return reconcile.Result{}, err
	}

	// Node is found in the cluster, add or update to cache
	if dk != nil {
		var ipAddress = dk.Status.OneAgent.Instances[nodeName].IPAddress
		info := CacheEntry{
			Instance:  dk.Name,
			IPAddress: ipAddress,
			LastSeen:  time.Now().UTC(),
		}

		if cached, err := nodeCache.Get(nodeName); err == nil {
			info.LastMarkedForTermination = cached.LastMarkedForTermination
		}

		if err := nodeCache.Set(nodeName, info); err != nil {
			return reconcile.Result{}, err
		}

		//Handle unschedulable Nodes, if they have a OneAgent instance
		if isUnschedulable(&node) {
			err := markForTermination(MarkForTerminationOptions{
				nodeCache:    nodeCache,
				nodeName:     nodeName,
				cachedNode:   info,
				dynakube:     dk,
				client:       r.client,
				dtClientFunc: r.dtClientFunc,
			})

			if err != nil {
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
	return reconcile.Result{}, nodeCache.updateCache(r.client, ctx)
}

func getCache(mgrClient client.Client, scheme *runtime.Scheme) (*Cache, error) {
	var cm corev1.ConfigMap
	namespace := os.Getenv("POD_NAMESPACE")
	podName := os.Getenv("POD_NAME")

	err := mgrClient.Get(context.TODO(), client.ObjectKey{Name: cacheName, Namespace: namespace}, &cm)
	if err == nil {
		return &Cache{Obj: &cm}, nil
	}

	if k8serrors.IsNotFound(err) {
		cm := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      cacheName,
				Namespace: namespace,
			},
			Data: map[string]string{},
		}
		isLocal := os.Getenv("RUN_LOCAL") == "true"

		if !isLocal { // If running locally, don't set the controller.
			deploy, err := kubeobjects.GetDeployment(mgrClient, podName, namespace)
			if err != nil {
				return nil, err
			}

			if err = controllerutil.SetControllerReference(deploy, cm, scheme); err != nil {
				return nil, err
			}
		}

		return &Cache{Create: true, Obj: cm}, nil
	}

	return nil, err
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

				if err := r.removeNodeFromCache(nodeCache, cachedNodeInfo, cachedNodeName); err != nil {
					log.Error(err, "failed to remove node", "node", cachedNodeName)
					return err
				}
				break
			}
		}
	}
	return nil
}

func (r *ReconcileNode) removeNodeFromCache(nodeCache *Cache, cachedNode CacheEntry, nodeName string) error {
	if time.Now().UTC().Sub(cachedNode.LastSeen).Hours() > 1 {
		log.Info("removing stale node")
	} else if cachedNode.IPAddress == "" {
		log.Info("removing node with unknown IP")
	} else {
		return nil
	}
	nodeCache.Delete(nodeName)
	return nil
}

func sendMarkedForTermination(opts MarkForTerminationOptions) error {
	dtp, err := dynakube.NewDynatraceClientProperties(context.TODO(), opts.client, *opts.dynakube)
	if err != nil {
		log.Error(err, err.Error())
	}

	dtc, err := opts.dtClientFunc(*dtp)
	if err != nil {
		return err
	}

	entityID, err := dtc.GetEntityIDForIP(opts.cachedNode.IPAddress)
	if err != nil {
		log.Info("failed to send mark for termination event",
			"reason", "failed to determine entity id", "dynakube", opts.dynakube.Name, "nodeIP", opts.cachedNode.IPAddress, "cause", err)

		return err
	}

	ts := uint64(opts.cachedNode.LastSeen.Add(-10*time.Minute).UnixNano()) / uint64(time.Millisecond)
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

func markForTermination(opts MarkForTerminationOptions) error {
	if !isMarkableForTermination(&opts.cachedNode) {
		return nil
	}

	if err := opts.nodeCache.updateLastMarkedForTerminationTimestamp(&opts.cachedNode, opts.nodeName); err != nil {
		return err
	}

	log.Info("sending mark for termination event to dynatrace server", "dynakube", opts.dynakube.Name, "ip", opts.cachedNode.IPAddress,
		"node", opts.nodeName)

	return sendMarkedForTermination(opts)
}

func isUnschedulable(node *corev1.Node) bool {
	return node.Spec.Unschedulable || hasUnschedulableTaint(node)
}

func hasUnschedulableTaint(node *corev1.Node) bool {
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
func isMarkableForTermination(nodeInfo *CacheEntry) bool {
	// If the last mark was an hour ago, mark again
	// Zero value for time.Time is 0001-01-01, so first mark is also executed
	lastMarked := nodeInfo.LastMarkedForTermination
	return lastMarked.UTC().Add(time.Hour).Before(time.Now().UTC())
}
