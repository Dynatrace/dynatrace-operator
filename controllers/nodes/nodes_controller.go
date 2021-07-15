package nodes

import (
	"context"
	"os"
	"time"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	"github.com/Dynatrace/dynatrace-operator/controllers/dynakube"
	"github.com/Dynatrace/dynatrace-operator/controllers/utils"
	"github.com/Dynatrace/dynatrace-operator/dtclient"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

const (
	cacheName = "dynatrace-node-cache"
)

var unschedulableTaints = []string{"ToBeDeletedByClusterAutoscaler"}

type ReconcileNodes struct {
	namespace    string
	client       client.Client
	cache        cache.Cache
	scheme       *runtime.Scheme
	logger       logr.Logger
	dtClientFunc dynakube.DynatraceClientFunc
	local        bool
}

// Add creates a new Nodes Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager, ns string) error {
	return mgr.Add(&ReconcileNodes{
		namespace:    ns,
		client:       mgr.GetClient(),
		cache:        mgr.GetCache(),
		scheme:       mgr.GetScheme(),
		logger:       log.Log.WithName("nodes.controller"),
		dtClientFunc: dynakube.BuildDynatraceClient,
		local:        os.Getenv("RUN_LOCAL") == "true",
	})
}

// Start starts the Nodes Reconciler, and will block until a stop signal is sent.
func (r *ReconcileNodes) Start(stop context.Context) error {
	r.cache.WaitForCacheSync(stop)

	chDels, err := r.watchDeletions(stop.Done())
	if err != nil {
		// I've seen watchDeletions() fail because the Cache Informers weren't ready. WaitForCacheSync()
		// should block until they are, however, but I believe I saw this not being true once.
		//
		// Start() failing would exit the Operator process. Since this is a minor feature, let's disable
		// for now until further investigation is done.
		r.logger.Info("failed to initialize watcher for deleted nodes - disabled", "error", err)
		chDels = make(chan string)
	}
	chUpdates, err := r.watchUpdates()
	if err != nil {
		r.logger.Info("failed to initialize watcher for updating nodes - disabled", "error", err)
		chUpdates = make(chan string)
	}

	chAll := watchTicks(stop.Done(), 5*time.Minute)

	for {
		select {
		case <-stop.Done():
			r.logger.Info("stopping nodes controller")
			return nil
		case node := <-chDels:
			if err := r.onDeletion(node); err != nil {
				r.logger.Error(err, "failed to reconcile deletion", "node", node)
			}
		case node := <-chUpdates:
			if err := r.onUpdate(node); err != nil {
				r.logger.Error(err, "failed to reconcile updates", "node", node)
			}
		case <-chAll:
			if err := r.reconcileAll(); err != nil {
				r.logger.Error(err, "failed to reconcile nodes")
			}
		}
	}
}

func (r *ReconcileNodes) onUpdate(node string) error {
	c, err := r.getCache()
	if err != nil {
		return err
	}

	if err = r.updateNode(c, node); err != nil {
		return err
	}

	return r.updateCache(c)
}

func (r *ReconcileNodes) onDeletion(node string) error {
	logger := r.logger.WithValues("node", node)

	logger.Info("node deletion notification received")

	c, err := r.getCache()
	if err != nil {
		return err
	}

	if err = r.removeNode(c, node, func(oaName string) (*dynatracev1alpha1.DynaKube, error) {
		var dynaKube dynatracev1alpha1.DynaKube
		if err := r.client.Get(context.TODO(), client.ObjectKey{Name: oaName, Namespace: r.namespace}, &dynaKube); err != nil {
			return nil, err
		}
		return &dynaKube, nil
	}); err != nil {
		return err
	}

	return r.updateCache(c)
}

func (r *ReconcileNodes) reconcileAll() error {
	r.logger.Info("reconciling nodes")

	var oaLst dynatracev1alpha1.DynaKubeList
	if err := r.client.List(context.TODO(), &oaLst, client.InNamespace(r.namespace)); err != nil {
		return err
	}

	oas := make(map[string]*dynatracev1alpha1.DynaKube, len(oaLst.Items))
	for i := range oaLst.Items {
		oas[oaLst.Items[i].Name] = &oaLst.Items[i]
	}

	c, err := r.getCache()
	if err != nil {
		return err
	}

	var nodeLst corev1.NodeList
	if err := r.client.List(context.TODO(), &nodeLst); err != nil {
		return err
	}

	nodes := map[string]bool{}
	for i := range nodeLst.Items {
		node := nodeLst.Items[i]
		nodes[node.Name] = true

		// Sometimes Azure does not cordon off nodes before deleting them since they use taints,
		// this case is handled in the update event handler
		if isUnschedulable(&node) {
			if err = r.reconcileUnschedulableNode(&node, c); err != nil {
				return err
			}
		}
	}

	// Add or update all nodes seen on OneAgent instances to the c.
	for _, oa := range oas {
		if oa.Status.OneAgent.Instances != nil {
			for node, info := range oa.Status.OneAgent.Instances {
				if _, ok := nodes[node]; !ok {
					continue
				}

				info := CacheEntry{
					Instance:  oa.Name,
					IPAddress: info.IPAddress,
					LastSeen:  time.Now().UTC(),
				}

				if cached, err := c.Get(node); err == nil {
					info.LastMarkedForTermination = cached.LastMarkedForTermination
				}

				if err := c.Set(node, info); err != nil {
					return err
				}
			}
		}
	}

	// Notify and remove all nodes on the c that aren't in the cluster.
	for _, node := range c.Keys() {
		if _, ok := nodes[node]; ok {
			continue
		}

		if err := r.removeNode(c, node, func(name string) (*dynatracev1alpha1.DynaKube, error) {
			if oa, ok := oas[name]; ok {
				return oa, nil
			}

			return nil, errors.NewNotFound(schema.GroupResource{
				Group:    oaLst.GroupVersionKind().Group,
				Resource: oaLst.GroupVersionKind().Kind,
			}, name)
		}); err != nil {
			r.logger.Error(err, "failed to remove node", "node", node)
		}
	}

	return r.updateCache(c)
}

func (r *ReconcileNodes) getCache() (*Cache, error) {
	var cm corev1.ConfigMap

	err := r.client.Get(context.TODO(), client.ObjectKey{Name: cacheName, Namespace: r.namespace}, &cm)
	if err == nil {
		return &Cache{Obj: &cm}, nil
	}

	if errors.IsNotFound(err) {
		r.logger.Info("no cache found, creating")

		cm := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      cacheName,
				Namespace: r.namespace,
			},
			Data: map[string]string{},
		}

		if !r.local { // If running locally, don't set the controller.
			deploy, err := utils.GetDeployment(r.client, r.namespace)
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

func (r *ReconcileNodes) updateCache(c *Cache) error {
	if !c.Changed() {
		return nil
	}

	if c.Create {
		return r.client.Create(context.TODO(), c.Obj)
	}

	return r.client.Update(context.TODO(), c.Obj)
}

func (r *ReconcileNodes) removeNode(c *Cache, node string, oaFunc func(name string) (*dynatracev1alpha1.DynaKube, error)) error {
	logger := r.logger.WithValues("node", node)

	nodeInfo, err := c.Get(node)
	if err == ErrNotFound {
		logger.Info("ignoring uncached node")
		return nil
	} else if err != nil {
		return err
	}

	if time.Now().UTC().Sub(nodeInfo.LastSeen).Hours() > 1 {
		logger.Info("removing stale node")
	} else if nodeInfo.IPAddress == "" {
		logger.Info("removing node with unknown IP")
	} else {
		oa, err := oaFunc(nodeInfo.Instance)
		if errors.IsNotFound(err) {
			logger.Info("oneagent got already deleted")
			c.Delete(node)
			return nil
		}
		if err != nil {
			return err
		}

		err = r.markForTermination(c, oa, nodeInfo.IPAddress, node)
		if err != nil {
			return err
		}
	}

	c.Delete(node)
	return nil
}

func (r *ReconcileNodes) updateNode(c *Cache, nodeName string) error {
	node := &corev1.Node{}
	err := r.client.Get(context.TODO(), client.ObjectKey{Name: nodeName}, node)
	if err != nil {
		return err
	}

	if !isUnschedulable(node) {
		return nil
	}

	return r.reconcileUnschedulableNode(node, c)
}

func (r *ReconcileNodes) sendMarkedForTermination(dk *dynatracev1alpha1.DynaKube, nodeIP string, lastSeen time.Time) error {
	var secret corev1.Secret
	if err := r.client.Get(context.TODO(), client.ObjectKey{Name: dk.Tokens(), Namespace: dk.Namespace}, &secret); err != nil {
		r.logger.Error(err, "Failed to query for tokens")
	}

	dtc, err := r.dtClientFunc(r.client, dk, &secret)
	if err != nil {
		return err
	}

	entityID, err := dtc.GetEntityIDForIP(nodeIP)
	if err != nil {
		r.logger.Info("failed to send mark for termination event",
			"reason", "failed to determine entity id", "dynakube", dk.Name, "nodeIP", nodeIP, "cause", err)

		return nil
	}

	ts := uint64(lastSeen.Add(-10*time.Minute).UnixNano()) / uint64(time.Millisecond)
	return dtc.SendEvent(&dtclient.EventData{
		EventType:     dtclient.MarkedForTerminationEvent,
		Source:        "OneAgent Operator",
		Description:   "Kubernetes node cordoned. Node might be drained or terminated.",
		StartInMillis: ts,
		EndInMillis:   ts,
		AttachRules: dtclient.EventDataAttachRules{
			EntityIDs: []string{entityID},
		},
	})
}

func (r *ReconcileNodes) reconcileUnschedulableNode(node *corev1.Node, c *Cache) error {
	oneAgent, err := r.determineOneAgentForNode(node.Name)
	if err != nil {
		return err
	}
	if oneAgent == nil {
		return nil
	}

	// determineOneAgentForNode  only returns a oneagent object if a node instance is present
	instance := oneAgent.Status.OneAgent.Instances[node.Name]
	if _, err = c.Get(node.Name); err != nil {
		if err == ErrNotFound {
			// If node not found in c add it
			cachedNode := CacheEntry{
				Instance:  oneAgent.Name,
				IPAddress: instance.IPAddress,
				LastSeen:  time.Now().UTC(),
			}
			err = c.Set(node.Name, cachedNode)
			if err != nil {
				return err
			}
		} else {
			return err
		}
	}
	return r.markForTermination(c, oneAgent, instance.IPAddress, node.Name)
}

func (r *ReconcileNodes) markForTermination(c *Cache, dk *dynatracev1alpha1.DynaKube,
	ipAddress string, nodeName string) error {
	cachedNode, err := c.Get(nodeName)
	if err != nil {
		return err
	}

	if !isMarkableForTermination(&cachedNode) {
		return nil
	}

	if err = updateLastMarkedForTerminationTimestamp(c, &cachedNode, nodeName); err != nil {
		return err
	}

	r.logger.Info("sending mark for termination event to dynatrace server", "dynakube", dk.Name, "ip", ipAddress,
		"node", nodeName)

	return r.sendMarkedForTermination(dk, ipAddress, cachedNode.LastSeen)
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

func updateLastMarkedForTerminationTimestamp(c *Cache, nodeInfo *CacheEntry, nodeName string) error {
	nodeInfo.LastMarkedForTermination = time.Now().UTC()
	return c.Set(nodeName, *nodeInfo)
}
